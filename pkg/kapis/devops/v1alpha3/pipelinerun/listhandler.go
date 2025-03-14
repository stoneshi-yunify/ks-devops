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

package pipelinerun

import (
	"context"
	"strings"

	"github.com/kubesphere/ks-devops/pkg/api"
	"github.com/kubesphere/ks-devops/pkg/api/devops/v1alpha3"
	cmstore "github.com/kubesphere/ks-devops/pkg/store/configmap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"kubesphere.io/kubesphere/pkg/apiserver/query"
	resourcesv1alpha3 "kubesphere.io/kubesphere/pkg/models/resources/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// listHandler is default implementation for PipelineRun.
type listHandler struct {
	ctx    context.Context
	client client.Client
}

// Comparator compares times first, which is from start time and creation time(only when start time is nil or zero).
// If times are equal, we will compare the unique name at last to
// ensure that the order result is stable forever.
func (b listHandler) Comparator() resourcesv1alpha3.CompareFunc {
	return func(left, right runtime.Object, f query.Field) bool {
		leftPipelineRun, ok := left.(*v1alpha3.PipelineRun)
		if !ok {
			return false
		}
		rightPipelineRun, ok := right.(*v1alpha3.PipelineRun)
		if !ok {
			return false
		}
		// Compare start time and creation time(if missing former)
		leftTime := leftPipelineRun.Status.StartTime
		if leftTime.IsZero() {
			leftTime = &leftPipelineRun.CreationTimestamp
		}
		rightTime := rightPipelineRun.Status.StartTime
		if rightTime.IsZero() {
			rightTime = &rightPipelineRun.CreationTimestamp
		}
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime.Time)
		}
		return strings.Compare(leftPipelineRun.Name, rightPipelineRun.Name) < 0
	}
}

func (b listHandler) Filter() resourcesv1alpha3.FilterFunc {
	return api.DefaultFilterFunc
}

func (b listHandler) Transformer() resourcesv1alpha3.TransformFunc {
	return func(obj runtime.Object) runtime.Object {
		pr, ok := obj.(*v1alpha3.PipelineRun)
		if !ok {
			return obj
		}

		// get status
		if _, ok := pr.Annotations[v1alpha3.JenkinsPipelineRunStatusAnnoKey]; !ok {
			pipelineRunStore, err := cmstore.NewConfigMapStore(b.ctx, types.NamespacedName{
				Namespace: pr.Namespace, Name: pr.Name}, b.client)
			if err == nil {
				pr.Annotations[v1alpha3.JenkinsPipelineRunStatusAnnoKey] = pipelineRunStore.GetStatus()
			} else {
				klog.Error(err, "failed to get status from configmap store")
			}
		}

		return pr
	}
}
