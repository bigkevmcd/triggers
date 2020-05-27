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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/gitlab"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"

	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
)

func TestInterceptor_ExecuteTrigger(t *testing.T) {
	type args struct {
		payload   []byte
		secret    *corev1.Secret
		token     string
		eventType string
	}
	tests := []struct {
		name            string
		GitLab          *triggersv1.GitLabInterceptor
		args            args
		want            []byte
		wantErr         bool
		wantContextHook scm.Webhook
	}{
		{
			name:   "no secret",
			GitLab: &triggersv1.GitLabInterceptor{},
			args: args{
				payload:   []byte("{}"),
				token:     "foo",
				eventType: "Push Hook",
			},
			want:    []byte("{}"),
			wantErr: false,
		},
		{
			name: "invalid header for secret",
			GitLab: &triggersv1.GitLabInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				token: "foo",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
				payload: []byte("somepayload"),
			},
			wantErr: true,
		},
		{
			name: "valid header for secret",
			GitLab: &triggersv1.GitLabInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				token: "secret",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secret"),
					},
				},
				payload:   []byte("{}"),
				eventType: "Push Hook",
			},
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name: "valid event",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"Push Hook", "Merge Request Hook"},
			},
			args: args{
				eventType: "Push Hook",
				payload:   []byte("{}"),
			},
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name: "invalid event",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"Push Hook", "Merge Request Hook"},
			},
			args: args{
				eventType: "baz",
				payload:   []byte("somepayload"),
			},
			wantErr: true,
		},
		{
			name: "valid event, invalid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "Push Hook",
				payload:   []byte("somepayload"),
				token:     "foo",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid event, valid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"foo", "bar"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "baz",
				payload:   []byte("somepayload"),
				token:     "secrettoken",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid event, valid secret",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"Push Hook", "Merge Request Hook"},
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				eventType: "Push Hook",
				payload:   []byte("{}"),
				token:     "secrettoken",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
			},
			want: []byte("{}"),
		},
		{
			name: "adding the hook to the context",
			GitLab: &triggersv1.GitLabInterceptor{
				EventTypes: []string{"Push Hook"},
			},
			args: args{
				payload:   []byte(`{"ref":"refs/heads/master","project":{"id":15}}`),
				eventType: "Push Hook",
			},
			wantContextHook: &scm.PushHook{Ref: "refs/heads/master", Repo: scm.Repository{ID: "15"}},
			want:            []byte(`{"ref":"refs/heads/master","project":{"id":15}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger, _ := logging.NewLogger("", "")
			kubeClient := fakekubeclient.Get(ctx)
			request := &http.Request{
				Body: ioutil.NopCloser(bytes.NewReader(tt.args.payload)),
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
			}
			if tt.args.token != "" {
				request.Header.Add("X-GitLab-Token", tt.args.token)
			}
			if tt.args.eventType != "" {
				request.Header.Add("X-GitLab-Event", tt.args.eventType)
			}
			if tt.args.secret != nil {
				ns := tt.GitLab.SecretRef.Namespace
				if ns == "" {
					ns = metav1.NamespaceDefault
				}
				if _, err := kubeClient.CoreV1().Secrets(ns).Create(tt.args.secret); err != nil {
					t.Error(err)
				}
			}
			w := &Interceptor{
				KubeClientSet: kubeClient,
				GitLab:        tt.GitLab,
				Logger:        logger,
				scmClient:     gitlab.NewDefault(),
			}
			ctx, resp, err := w.ExecuteTrigger(request)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Interceptor.ExecuteTrigger() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("error reading response: %v", err)
			}
			defer resp.Body.Close()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Interceptor.ExecuteTrigger (-want, +got) = %s", diff)
			}

			if tt.wantContextHook != nil {
				h, ok := interceptors.InterceptedHook(ctx)
				if !ok {
					t.Fatal("failed to find the intercepted hook")
				}
				if diff := cmp.Diff(tt.wantContextHook, h); diff != "" {
					t.Errorf("hook comparison failed:\n%s", diff)
				}
			}
		})
	}
}
