/*
Copyright 2019 The Tekton Authors

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

package gitlab

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/gitlab"
	"github.com/tektoncd/triggers/pkg/interceptors"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Interceptor struct {
	KubeClientSet          kubernetes.Interface
	Logger                 *zap.SugaredLogger
	GitLab                 *triggersv1.GitLabInterceptor
	EventListenerNamespace string
	scmClient              *scm.Client
}

func NewInterceptor(gl *triggersv1.GitLabInterceptor, k kubernetes.Interface, ns string, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:                 l,
		GitLab:                 gl,
		KubeClientSet:          k,
		EventListenerNamespace: ns,
		scmClient:              gitlab.NewDefault(),
	}
}

func (w *Interceptor) ExecuteTrigger(request *http.Request) (context.Context, *http.Response, error) {
	payload := []byte{}
	var err error

	defer request.Body.Close()
	payload, err = ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read request body: %w", err)
	}

	ctx := request.Context()
	request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
	hook, err := w.scmClient.Webhooks.Parse(request, func(scm.Webhook) (string, error) {
		if w.GitLab.SecretRef == nil {
			return "", nil
		}
		b, err := interceptors.GetSecretToken(w.KubeClientSet, w.GitLab.SecretRef, w.EventListenerNamespace)
		if err != nil {
			return "", nil
		}
		return string(b), nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse the webhook: %w", err)
	}
	ctx = interceptors.WithHook(request.Context(), hook)

	if w.GitLab.EventTypes != nil {
		actualEvent := request.Header.Get("X-GitLab-Event")
		isAllowed := false
		for _, allowedEvent := range w.GitLab.EventTypes {
			if actualEvent == allowedEvent {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return nil, nil, fmt.Errorf("event type %s is not allowed", actualEvent)
		}
	}

	return ctx, &http.Response{
		Header: request.Header,
		Body:   ioutil.NopCloser(bytes.NewBuffer(payload)),
	}, nil
}
