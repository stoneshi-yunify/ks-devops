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

package k8s

import (
	"github.com/kubesphere/ks-devops/pkg/client/clientset/versioned"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type FakeClient struct {
	// kubernetes client interface
	K8sClient kubernetes.Interface

	// discovery client
	DiscoveryClient *discovery.DiscoveryClient

	ApiExtensionClient apiextensionsclient.Interface
	MasterURL          string
	KubeConfig         *rest.Config

	ksClient versioned.Interface
}

func (n *FakeClient) KubeSphere() versioned.Interface {
	return n.ksClient
}

func NewFakeClientSets(k8sClient kubernetes.Interface, discoveryClient *discovery.DiscoveryClient,
	apiextensionsclient apiextensionsclient.Interface,
	masterURL string, kubeConfig *rest.Config, ksClient versioned.Interface) Client {
	return &FakeClient{
		K8sClient:          k8sClient,
		DiscoveryClient:    discoveryClient,
		ApiExtensionClient: apiextensionsclient,
		MasterURL:          masterURL,
		KubeConfig:         kubeConfig,
		ksClient:           ksClient,
	}
}

func (n *FakeClient) Kubernetes() kubernetes.Interface {
	return n.K8sClient
}

func (n *FakeClient) ApiExtensions() apiextensionsclient.Interface {
	return n.ApiExtensionClient
}

func (n *FakeClient) Discovery() discovery.DiscoveryInterface {
	return n.DiscoveryClient
}

func (n *FakeClient) Master() string {
	return n.MasterURL
}

func (n *FakeClient) Config() *rest.Config {
	return n.KubeConfig
}
