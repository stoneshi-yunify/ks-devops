/*
Copyright 2022 The KubeSphere Authors.
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

package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/jenkins-zh/jenkins-client/pkg/core"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/kubesphere/ks-devops/pkg/api/devops"
)

// AgentLabelsReconciler responsible for the Jenkins agent labels sync
type AgentLabelsReconciler struct {
	// TargetNamespace indicate which namespace the target ConfigMap located in
	TargetNamespace string
	JenkinsClient   core.Client

	targetName string
	client.Client
	log      logr.Logger
	recorder record.EventRecorder
}

// Reconcile makes sure the target ConfigMap has all the Jenkins agent labels
func (r *AgentLabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	var cm *v1.ConfigMap
	if cm, err = r.getConfigMap(); err != nil {
		if apierrors.IsNotFound(err) {
			// retry to wait the configmap created
			result = ctrl.Result{RequeueAfter: time.Second * 30}
		}

		if err != nil {
			return
		}
	}

	var labels []string
	if labels, err = r.getLabels(); err != nil {
		err = fmt.Errorf("failed to get labels, error: %v", err)
		return
	}

	if changed := setLabelsToConfigMap(labels, cm); changed {
		err = r.Update(context.Background(), cm)
	}

	// make sure we could always get the latest Jenkins labels
	result = ctrl.Result{
		RequeueAfter: time.Minute * 5,
	}
	return
}

func (r *AgentLabelsReconciler) getLabels() (labels []string, err error) {
	var labelRes *core.LabelsResponse
	if labelRes, err = r.JenkinsClient.GetLabels(); err != nil {
		err = fmt.Errorf("failed to get lables from Jenkins, error: %v", err)
		return
	}
	labels = labelRes.GetLabels()
	return
}

func setLabelsToConfigMap(labels []string, cm *v1.ConfigMap) (changed bool) {
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	if cm.Data[devops.JenkinsAgentLabelsKey] != strings.Join(labels, ",") {
		cm.Data[devops.JenkinsAgentLabelsKey] = strings.Join(labels, ",")
		changed = true
	}
	return
}

func (r *AgentLabelsReconciler) getConfigMap() (cm *v1.ConfigMap, err error) {
	cm = &v1.ConfigMap{}
	err = r.Get(context.Background(), types.NamespacedName{
		Namespace: r.TargetNamespace,
		Name:      r.targetName,
	}, cm)
	return
}

// GetName returns the name of this reconciler
func (r *AgentLabelsReconciler) GetName() string {
	return "JenkinsAgentLabelReconciler"
}

// GetGroupName returns the group name of this reconciler
func (r *AgentLabelsReconciler) GetGroupName() string {
	return reconcilerGroupName
}

func getSpecificConfigMapPredicate(name, namespace string) predicate.Funcs {
	return predicate.NewPredicateFuncs(func(object client.Object) (ok bool) {
		ok = object.GetName() == name && object.GetNamespace() == namespace
		return
	})
}

// SetupWithManager setups the all necessary fields
func (r *AgentLabelsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.targetName == "" {
		r.targetName = "jenkins-agent-config"
	}
	if r.TargetNamespace == "" {
		return errors.New("the target namespace is required")
	}

	r.log = ctrl.Log.WithName(r.GetName())
	r.recorder = mgr.GetEventRecorderFor(r.GetName())
	return ctrl.NewControllerManagedBy(mgr).
		Named("jenkins_config_labels_controller").
		WithEventFilter(getSpecificConfigMapPredicate(r.targetName, r.TargetNamespace)).
		For(&v1.ConfigMap{}).
		Complete(r)
}
