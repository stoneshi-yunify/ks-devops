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

package jenkins

import (
	"github.com/emicklei/go-restful/v3"
	"k8s.io/klog/v2"

	"github.com/kubesphere/ks-devops/pkg/client/devops"
)

func (j *Jenkins) CreateDevOpsProject(projectId string) (string, error) {
	_, err := j.CreateFolder(projectId, "")
	if err != nil {
		klog.Errorf("%+v", err)
		return "", restful.NewError(devops.GetDevOpsStatusCode(err), err.Error())
	}
	return projectId, nil
}

func (j *Jenkins) DeleteDevOpsProject(projectId string) (err error) {
	_, err = j.DeleteJob(projectId)
	if err != nil {
		return restful.NewError(devops.GetDevOpsStatusCode(err), err.Error())
	}
	return
}

func (j *Jenkins) GetDevOpsProject(projectId string) (string, error) {
	job, err := j.GetJob(projectId)
	if err != nil {
		klog.Errorf("%+v", err)
		return "", restful.NewError(devops.GetDevOpsStatusCode(err), err.Error())

	}
	return job.GetName(), nil
}
