/*
Copyright 2020 KubeSphere Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package devopscredential

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informer "k8s.io/client-go/informers/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	devopsv1alpha3 "github.com/kubesphere/ks-devops/pkg/api/devops/v1alpha3"

	devopsClient "github.com/kubesphere/ks-devops/pkg/client/devops"
	"github.com/kubesphere/ks-devops/pkg/constants"
	"github.com/kubesphere/ks-devops/pkg/utils"
	"github.com/kubesphere/ks-devops/pkg/utils/sliceutil"
)

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;update;watch

// Controller is the controller for DevOpsProject
type Controller struct {
	client           clientset.Interface
	eventBroadcaster record.EventBroadcaster
	eventRecorder    record.EventRecorder

	secretLister corev1lister.SecretLister
	secretSynced cache.InformerSynced

	namespaceLister corev1lister.NamespaceLister
	namespaceSynced cache.InformerSynced

	workqueue workqueue.RateLimitingInterface

	workerLoopPeriod time.Duration

	devopsClient devopsClient.Interface
}

// NewController creates an instance of the DevOpsProject controller
func NewController(client clientset.Interface,
	devopsClient devopsClient.Interface,
	namespaceInformer corev1informer.NamespaceInformer,
	secretInformer corev1informer.SecretInformer) *Controller {

	broadcaster := record.NewBroadcaster()
	broadcaster.StartLogging(func(format string, args ...interface{}) {
		klog.Info(fmt.Sprintf(format, args))
	})
	broadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: client.CoreV1().Events("")})
	recorder := broadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: "devopscredential-controller"})

	v := &Controller{
		client:           client,
		devopsClient:     devopsClient,
		workqueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "devopscredential"),
		secretLister:     secretInformer.Lister(),
		secretSynced:     secretInformer.Informer().HasSynced,
		namespaceLister:  namespaceInformer.Lister(),
		namespaceSynced:  namespaceInformer.Informer().HasSynced,
		workerLoopPeriod: time.Second,
	}

	v.eventBroadcaster = broadcaster
	v.eventRecorder = recorder

	secretInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			secret, ok := obj.(*v1.Secret)
			if ok && strings.HasPrefix(string(secret.Type), devopsv1alpha3.DevOpsCredentialPrefix) {
				v.enqueueSecret(obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old, ook := oldObj.(*v1.Secret)
			new, nok := newObj.(*v1.Secret)
			if ook && nok && old.ResourceVersion == new.ResourceVersion {
				return
			}
			if ook && nok && strings.HasPrefix(string(new.Type), devopsv1alpha3.DevOpsCredentialPrefix) {
				v.enqueueSecret(newObj)
			}
		},
		DeleteFunc: func(obj interface{}) {
			secret, ok := obj.(*v1.Secret)
			if ok && strings.HasPrefix(string(secret.Type), devopsv1alpha3.DevOpsCredentialPrefix) {
				v.enqueueSecret(obj)
			}
		},
	})
	return v
}

// enqueueSecret takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work workqueue. This method should *not* be
// passed resources of any type other than DevOpsProject.
func (c *Controller) enqueueSecret(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		klog.V(5).Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		klog.Error(err, "could not reconcile devopsProject")
		utilruntime.HandleError(err)
		return true
	}

	return true
}

func (c *Controller) worker() {

	for c.processNextWorkItem() {
	}
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	return c.Run(1, ctx.Done())
}

// Run runs the controller
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("starting devopscredential controller")
	defer klog.Info("shutting down  devopscredential controller")

	if !cache.WaitForCacheSync(stopCh, c.secretSynced) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.worker, c.workerLoopPeriod, stopCh)
	}

	<-stopCh
	return nil
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the secret resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	nsName, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Error(err, fmt.Sprintf("could not split copySecret meta %s ", key))
		return nil
	}
	namespace, err := c.namespaceLister.Get(nsName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("namespace '%s' in work queue no longer exists ", key))
			return nil
		}
		klog.Error(err, fmt.Sprintf("could not get namespace %s ", key))
		return err
	}
	if !isDevOpsProjectAdminNamespace(namespace) {
		err := fmt.Errorf("cound not create or update credential '%s' in normal namespaces %s", name, namespace.Name)
		klog.Warning(err)
		return err
	}

	secret, err := c.secretLister.Secrets(nsName).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Info(fmt.Sprintf("secret '%s' in work queue no longer exists ", key))
			return nil
		}
		klog.Error(err, fmt.Sprintf("could not get secret %s ", key))
		return err
	}

	copySecret := secret.DeepCopy()
	// DeletionTimestamp.IsZero() means copySecret has not been deleted.
	if copySecret.ObjectMeta.DeletionTimestamp.IsZero() {
		// make sure Annotations is not nil
		if copySecret.Annotations == nil {
			copySecret.Annotations = map[string]string{}
		}

		//If the sync is successful, return handle
		if state, ok := copySecret.Annotations[devopsv1alpha3.CredentialSyncStatusAnnoKey]; ok && state == constants.StatusSuccessful {
			specHash := utils.ComputeHash(copySecret.Data)
			oldHash := copySecret.Annotations[devopsv1alpha3.DevOpsCredentialDataHash] // don't need to check if it's nil, only compare if they're different
			if specHash == oldHash {
				// it was synced successfully, and there's any change with the Pipeline spec, skip this round
				return nil
			}
			copySecret.Annotations[devopsv1alpha3.DevOpsCredentialDataHash] = specHash
		}

		// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/#finalizers
		if !sliceutil.HasString(secret.ObjectMeta.Finalizers, devopsv1alpha3.CredentialFinalizerName) {
			copySecret.ObjectMeta.Finalizers = append(copySecret.ObjectMeta.Finalizers, devopsv1alpha3.CredentialFinalizerName)
		}
		// Check secret config exists, otherwise we will create it.
		// if secret exists, update config
		_, err := c.devopsClient.GetCredentialInProject(nsName, copySecret.Name)
		if err == nil {
			if _, ok := copySecret.Annotations[devopsv1alpha3.CredentialAutoSyncAnnoKey]; ok {
				_, err := c.devopsClient.UpdateCredentialInProject(nsName, copySecret)
				if err != nil {
					klog.V(8).Info(err, fmt.Sprintf("failed to update secret %s ", key))
					return err
				}
			}
		} else {
			_, err = c.devopsClient.CreateCredentialInProject(nsName, copySecret)
			if err != nil {
				klog.V(8).Info(err, fmt.Sprintf("failed to create secret %s ", key))
				return err
			}
		}
		//If there is no early return, then the sync is successful.
		copySecret.Annotations[devopsv1alpha3.CredentialSyncStatusAnnoKey] = constants.StatusSuccessful
	} else {
		// Finalizers processing logic
		if sliceutil.HasString(copySecret.ObjectMeta.Finalizers, devopsv1alpha3.CredentialFinalizerName) {
			delSuccess := false
			if _, err := c.devopsClient.DeleteCredentialInProject(nsName, secret.Name); err != nil {
				// the status code should be 404 if the credential does not exists
				if srvErr, ok := err.(restful.ServiceError); ok {
					delSuccess = srvErr.Code == http.StatusNotFound
				} else if srvErr, ok := err.(*devopsClient.ErrorResponse); ok {
					delSuccess = srvErr.Response.StatusCode == http.StatusNotFound
				} else {
					klog.Error(fmt.Sprintf("unexpected error type: %v, should be *restful.ServiceError", err))
				}

				klog.V(8).Info(err, fmt.Sprintf("failed to delete secret %s in devops", key))
			} else {
				delSuccess = true
			}

			if delSuccess {
				copySecret.ObjectMeta.Finalizers = sliceutil.RemoveString(copySecret.ObjectMeta.Finalizers, func(item string) bool {
					return item == devopsv1alpha3.CredentialFinalizerName
				})
			} else {
				// make sure the corresponding Jenkins credentials can be clean
				// You can remove the finalizer via kubectl manually in a very special case that Jenkins might be not able to available anymore
				return fmt.Errorf("failed to remove devops credential finalizer due to bad communication with Jenkins")
			}

		}
	}
	if !reflect.DeepEqual(secret, copySecret) {
		_, err = c.client.CoreV1().Secrets(nsName).Update(context.Background(), copySecret, metav1.UpdateOptions{})
		if err != nil {
			klog.V(8).Info(err, fmt.Sprintf("failed to update secret %s ", key))
			return err
		}
	}
	return nil
}

func isDevOpsProjectAdminNamespace(namespace *v1.Namespace) bool {
	_, ok := namespace.Labels[constants.DevOpsProjectLabelKey]
	return ok
}
