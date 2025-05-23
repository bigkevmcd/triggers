/*
Copyright 2020 The Tekton Authors

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

package v1

import (
	context "context"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	scheme "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// PipelineRunsGetter has a method to return a PipelineRunInterface.
// A group's client should implement this interface.
type PipelineRunsGetter interface {
	PipelineRuns(namespace string) PipelineRunInterface
}

// PipelineRunInterface has methods to work with PipelineRun resources.
type PipelineRunInterface interface {
	Create(ctx context.Context, pipelineRun *pipelinev1.PipelineRun, opts metav1.CreateOptions) (*pipelinev1.PipelineRun, error)
	Update(ctx context.Context, pipelineRun *pipelinev1.PipelineRun, opts metav1.UpdateOptions) (*pipelinev1.PipelineRun, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, pipelineRun *pipelinev1.PipelineRun, opts metav1.UpdateOptions) (*pipelinev1.PipelineRun, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*pipelinev1.PipelineRun, error)
	List(ctx context.Context, opts metav1.ListOptions) (*pipelinev1.PipelineRunList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *pipelinev1.PipelineRun, err error)
	PipelineRunExpansion
}

// pipelineRuns implements PipelineRunInterface
type pipelineRuns struct {
	*gentype.ClientWithList[*pipelinev1.PipelineRun, *pipelinev1.PipelineRunList]
}

// newPipelineRuns returns a PipelineRuns
func newPipelineRuns(c *TektonV1Client, namespace string) *pipelineRuns {
	return &pipelineRuns{
		gentype.NewClientWithList[*pipelinev1.PipelineRun, *pipelinev1.PipelineRunList](
			"pipelineruns",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *pipelinev1.PipelineRun { return &pipelinev1.PipelineRun{} },
			func() *pipelinev1.PipelineRunList { return &pipelinev1.PipelineRunList{} },
		),
	}
}
