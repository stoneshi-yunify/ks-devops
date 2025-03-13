/*
Copyright 2020 The KubeSphere Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha3

import (
	"context"
	"time"

	v1alpha3 "github.com/kubesphere/ks-devops/pkg/api/devops/v1alpha3"
	scheme "github.com/kubesphere/ks-devops/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// PipelinesGetter has a method to return a PipelineInterface.
// A group's client should implement this interface.
type PipelinesGetter interface {
	Pipelines(namespace string) PipelineInterface
}

// PipelineInterface has methods to work with Pipeline resources.
type PipelineInterface interface {
	Create(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.CreateOptions) (*v1alpha3.Pipeline, error)
	Update(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.UpdateOptions) (*v1alpha3.Pipeline, error)
	UpdateStatus(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.UpdateOptions) (*v1alpha3.Pipeline, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha3.Pipeline, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha3.PipelineList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha3.Pipeline, err error)
	PipelineExpansion
}

// pipelines implements PipelineInterface
type pipelines struct {
	client rest.Interface
	ns     string
}

// newPipelines returns a Pipelines
func newPipelines(c *DevopsV1alpha3Client, namespace string) *pipelines {
	return &pipelines{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the pipeline, and returns the corresponding pipeline object, and an error if there is any.
func (c *pipelines) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha3.Pipeline, err error) {
	result = &v1alpha3.Pipeline{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("pipelines").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Pipelines that match those selectors.
func (c *pipelines) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha3.PipelineList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha3.PipelineList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("pipelines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested pipelines.
func (c *pipelines) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("pipelines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a pipeline and creates it.  Returns the server's representation of the pipeline, and an error, if there is any.
func (c *pipelines) Create(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.CreateOptions) (result *v1alpha3.Pipeline, err error) {
	result = &v1alpha3.Pipeline{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("pipelines").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(pipeline).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a pipeline and updates it. Returns the server's representation of the pipeline, and an error, if there is any.
func (c *pipelines) Update(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.UpdateOptions) (result *v1alpha3.Pipeline, err error) {
	result = &v1alpha3.Pipeline{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("pipelines").
		Name(pipeline.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(pipeline).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *pipelines) UpdateStatus(ctx context.Context, pipeline *v1alpha3.Pipeline, opts v1.UpdateOptions) (result *v1alpha3.Pipeline, err error) {
	result = &v1alpha3.Pipeline{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("pipelines").
		Name(pipeline.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(pipeline).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the pipeline and deletes it. Returns an error if one occurs.
func (c *pipelines) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("pipelines").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *pipelines) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("pipelines").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched pipeline.
func (c *pipelines) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha3.Pipeline, err error) {
	result = &v1alpha3.Pipeline{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("pipelines").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
