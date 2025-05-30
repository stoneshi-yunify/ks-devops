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
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	fakeDevOps "github.com/kubesphere/ks-devops/pkg/client/devops/fake"
	"github.com/kubesphere/ks-devops/pkg/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	devops "github.com/kubesphere/ks-devops/pkg/api/devops/v1alpha3"
)

var (
	alwaysReady        = func() bool { return true }
	noResyncPeriodFunc = func() time.Duration { return 0 }
)

type fixture struct {
	t *testing.T

	kubeclient      *k8sfake.Clientset
	namespaceLister []*v1.Namespace
	secretLister    []*v1.Secret
	kubeactions     []core.Action

	kubeobjects []runtime.Object
	// Objects from here preloaded into NewSimpleFake.
	objects []runtime.Object
	// Objects from here preloaded into devops
	initDevOpsProject string
	initCredential    []*v1.Secret
	expectCredential  []*v1.Secret
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{}
	f.t = t
	f.objects = []runtime.Object{}
	return f
}

func newNamespace(name string, projectName string) *v1.Namespace {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{constants.DevOpsProjectLabelKey: projectName},
		},
	}
	TRUE := true
	ns.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         devops.GroupVersion.String(),
			Kind:               devops.ResourceKindDevOpsProject,
			Name:               projectName,
			BlockOwnerDeletion: &TRUE,
			Controller:         &TRUE,
		},
	}

	return ns
}

func newSecret(namespace, name string, data map[string][]byte, withFinalizers bool, autoSync bool, syncOk bool) *v1.Secret {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       devops.ResourceKindPipeline,
			APIVersion: devops.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Name:        name,
			Annotations: map[string]string{},
		},
		Data: data,
		Type: devops.DevOpsCredentialPrefix + "test",
	}
	if withFinalizers {
		secret.Finalizers = append(secret.Finalizers, devops.CredentialFinalizerName)
	}
	if autoSync {
		secret.Annotations[devops.CredentialAutoSyncAnnoKey] = "true"
	}
	if syncOk {
		secret.Annotations[devops.CredentialSyncStatusAnnoKey] = constants.StatusSuccessful
	}
	return secret
}

func newDeletingSecret(namespace, name string) *v1.Secret {
	now := metav1.Now()
	pipeline := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       devops.ResourceKindPipeline,
			APIVersion: devops.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         namespace,
			Name:              name,
			DeletionTimestamp: &now,
		},
		Type: devops.DevOpsCredentialPrefix + "test",
	}
	pipeline.Finalizers = append(pipeline.Finalizers, devops.CredentialFinalizerName)

	return pipeline
}

func (f *fixture) newController() (*Controller, kubeinformers.SharedInformerFactory, *fakeDevOps.Devops) {
	f.kubeclient = k8sfake.NewSimpleClientset(f.kubeobjects...)

	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriodFunc())
	dI := fakeDevOps.NewWithCredentials(f.initDevOpsProject, f.initCredential...)

	c := NewController(f.kubeclient, dI, k8sI.Core().V1().Namespaces(),
		k8sI.Core().V1().Secrets())

	c.secretSynced = alwaysReady
	c.eventRecorder = &record.FakeRecorder{}
	for _, f := range f.secretLister {
		_ = k8sI.Core().V1().Secrets().Informer().GetIndexer().Add(f)
	}

	for _, d := range f.namespaceLister {
		_ = k8sI.Core().V1().Namespaces().Informer().GetIndexer().Add(d)
	}

	return c, k8sI, dI
}

func (f *fixture) run(fooName string) {
	f.runController(fooName, true, false)
}

func (f *fixture) runController(name string, startInformers bool, expectError bool) {
	c, k8sI, dI := f.newController()
	if startInformers {
		stopCh := make(chan struct{})
		defer close(stopCh)
		k8sI.Start(stopCh)
	}

	err := c.syncHandler(name)
	if !expectError && err != nil {
		f.t.Errorf("error syncing foo: %v", err)
	} else if expectError && err == nil {
		f.t.Error("expected error syncing foo, got nil")
	}

	k8sActions := filterInformerActions(f.kubeclient.Actions())

	if len(f.kubeactions) > len(k8sActions) {
		f.t.Errorf("%d additional expected actions:%+v", len(f.kubeactions)-len(k8sActions), f.kubeactions[len(k8sActions):])
	}

	if len(dI.Credentials[f.initDevOpsProject]) != len(f.expectCredential) {
		f.t.Errorf(" unexpected objects: %v", dI.Projects)
	}
	for _, credential := range f.expectCredential {
		actualCredential := dI.Credentials[f.initDevOpsProject][credential.Name]
		if !reflect.DeepEqual(actualCredential, credential) {
			f.t.Errorf(" credential %+v not match \n %+v", credential, actualCredential)
		}
	}
}

// filterInformerActions filters list and watch actions for testing resources.
// Since list and watch don't change resource state we can filter it to lower
// nose level in our tests.
func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if len(action.GetNamespace()) == 0 &&
			(action.Matches("list", "secrets") ||
				action.Matches("watch", "secrets") ||
				action.Matches("list", "namespaces") ||
				action.Matches("watch", "namespaces")) {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func (f *fixture) expectUpdateSecretAction(p *v1.Secret) {
	action := core.NewUpdateAction(schema.GroupVersionResource{
		Version:  "v1",
		Resource: "secrets",
	}, p.Namespace, p)
	f.kubeactions = append(f.kubeactions, action)
}

func getKey(p *v1.Secret, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(p)
	if err != nil {
		t.Errorf("Unexpected error getting key for pipeline %v: %v", p.Name, err)
		return ""
	}
	return key
}

func TestDoNothing(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	secret := newSecret(nsName, secretName, nil, true, true, false)
	expectSecret := newSecret(nsName, secretName, nil, true, true, true)

	f.secretLister = append(f.secretLister, secret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, secret)
	f.initDevOpsProject = nsName
	f.initCredential = []*v1.Secret{secret}
	f.expectCredential = []*v1.Secret{expectSecret}

	f.run(getKey(secret, t))
}

func TestAddCredentialFinalizers(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	secret := newSecret(nsName, secretName, nil, false, true, false)

	expectSecret := newSecret(nsName, secretName, nil, true, true, true)

	f.secretLister = append(f.secretLister, secret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, secret)
	f.initDevOpsProject = nsName
	f.initCredential = []*v1.Secret{secret}
	f.expectCredential = []*v1.Secret{expectSecret}
	f.expectUpdateSecretAction(expectSecret)
	f.run(getKey(secret, t))
}

func TestCreateCredential(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	secret := newSecret(nsName, secretName, nil, true, true, false)
	expectSecret := newSecret(nsName, secretName, nil, true, true, true)

	f.secretLister = append(f.secretLister, secret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, secret)
	f.initDevOpsProject = nsName
	f.expectCredential = []*v1.Secret{expectSecret}
	f.run(getKey(secret, t))
}

func TestDeleteCredential(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	secret := newDeletingSecret(nsName, secretName)

	expectSecret := secret.DeepCopy()
	expectSecret.Finalizers = []string{}
	f.secretLister = append(f.secretLister, secret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, secret)
	f.initDevOpsProject = nsName
	f.initCredential = []*v1.Secret{secret}
	f.expectCredential = []*v1.Secret{}
	f.expectUpdateSecretAction(expectSecret)
	f.run(getKey(secret, t))
}

func TestUpdateCredential(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	initSecret := newSecret(nsName, secretName, nil, true, true, false)
	modifiedSecret := newSecret(nsName, secretName, map[string][]byte{"a": []byte("aa")}, true, true, false)
	expectSecret := newSecret(nsName, secretName, map[string][]byte{"a": []byte("aa")}, true, true, true)
	f.secretLister = append(f.secretLister, modifiedSecret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, modifiedSecret)
	f.initDevOpsProject = nsName
	f.initCredential = []*v1.Secret{initSecret}
	f.expectCredential = []*v1.Secret{expectSecret}
	f.run(getKey(modifiedSecret, t))
}

func TestNotUpdateCredential(t *testing.T) {
	f := newFixture(t)
	nsName := "test-123"
	secretName := "test"
	projectName := "test_project"

	ns := newNamespace(nsName, projectName)
	initSecret := newSecret(nsName, secretName, nil, true, false, false)
	expectSecret := newSecret(nsName, secretName, map[string][]byte{"a": []byte("aa")}, true, false, true)
	f.secretLister = append(f.secretLister, expectSecret)
	f.namespaceLister = append(f.namespaceLister, ns)
	f.kubeobjects = append(f.kubeobjects, expectSecret)
	f.initDevOpsProject = nsName
	f.initCredential = []*v1.Secret{initSecret}
	f.expectCredential = []*v1.Secret{initSecret}
	f.run(getKey(expectSecret, t))
}
