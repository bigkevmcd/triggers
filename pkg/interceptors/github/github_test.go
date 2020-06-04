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

package github

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/github"
	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"github.com/tektoncd/triggers/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	rtesting "knative.dev/pkg/reconciler/testing"
)

var testSecret = []byte("secret")

func TestInterceptor_ExecuteTrigger_Signature(t *testing.T) {
	type args struct {
		payload   io.ReadCloser
		secret    *corev1.Secret
		signature string
		eventType string
	}
	tests := []struct {
		name            string
		GitHub          *triggersv1.GitHubInterceptor
		args            args
		want            []byte
		wantErr         bool
		wantContextHook scm.Webhook
	}{
		{
			name:   "no secret",
			GitHub: &triggersv1.GitHubInterceptor{},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("{}")),
				signature: "foo",
				eventType: "push",
			},
			want:    []byte("{}"),
			wantErr: false,
		},
		{
			name: "invalid header for secret",
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				signature: "foo",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
				payload: ioutil.NopCloser(bytes.NewBufferString("{}")),
			},
			wantErr: true,
		},
		{
			name: "valid header for secret",
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				signature: test.SHA1Signature(t, testSecret, []byte("{}")),
				eventType: "push",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": testSecret,
					},
				},
				payload: ioutil.NopCloser(bytes.NewBufferString("{}")),
			},
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name: "no secret, matching event",
			GitHub: &triggersv1.GitHubInterceptor{
				EventTypes: []string{"push", "pull_request"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("{}")),
				eventType: "push",
			},
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name: "no secret, failing event",
			GitHub: &triggersv1.GitHubInterceptor{
				EventTypes: []string{"push", "pull_request"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("{}")),
				eventType: "deployment_status",
			},
			wantErr: true,
		},
		{
			name: "valid header for secret and matching event",
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"push", "pull_request"},
			},
			args: args{
				signature: test.SHA1Signature(t, testSecret, []byte("{}")),
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte(testSecret),
					},
				},
				eventType: "push",
				payload:   ioutil.NopCloser(bytes.NewBufferString("{}")),
			},
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name: "valid header for secret, failing event",
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
			},
			args: args{
				// This was generated by using SHA1 and hmac from go stdlib on secret and payload.
				// https://play.golang.org/p/otp1o_cJTd7 for a sample.
				signature: "sha1=38e005ef7dd3faee13204505532011257023654e",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte(testSecret),
					},
				},
				eventType: "OTHER_EVENT",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		},
		{
			name: "invalid header for secret, matching event",
			GitHub: &triggersv1.GitHubInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"MY_EVENT", "YOUR_EVENT"},
			},
			args: args{
				signature: "foo",
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": []byte("secrettoken"),
					},
				},
				eventType: "MY_EVENT",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		}, {
			name:   "nil body does not panic",
			GitHub: &triggersv1.GitHubInterceptor{},
			args: args{
				payload:   nil,
				signature: "foo",
				eventType: "pull_request",
			},
			want:    []byte{},
			wantErr: false,
		},
		{
			name: "adding the hook to the context",
			GitHub: &triggersv1.GitHubInterceptor{
				EventTypes: []string{"push"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString(`{"ref": "refs/heads/master"}`)),
				eventType: "push",
			},
			wantContextHook: &scm.PushHook{Ref: "refs/heads/master"},
			want:            []byte(`{"ref": "refs/heads/master"}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := rtesting.SetupFakeContext(t)
			logger, _ := logging.NewLogger("", "")
			kubeClient := fakekubeclient.Get(ctx)
			request := &http.Request{
				Body: tt.args.payload,
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Github-Delivery": []string{"72555f37-eb79-4034-89f9-a3bf1eacc3e9"},
				},
			}
			if tt.args.eventType != "" {
				request.Header.Add("X-GITHUB-EVENT", tt.args.eventType)
			}
			if tt.args.signature != "" {
				request.Header.Add("X-Hub-Signature", tt.args.signature)
			}
			if tt.args.secret != nil {
				ns := tt.GitHub.SecretRef.Namespace
				if ns == "" {
					ns = metav1.NamespaceDefault
				}
				if _, err := kubeClient.CoreV1().Secrets(ns).Create(tt.args.secret); err != nil {
					t.Error(err)
				}
			}
			w := &Interceptor{
				KubeClientSet: kubeClient,
				GitHub:        tt.GitHub,
				Logger:        logger,
				scmClient:     github.NewDefault(),
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
				t.Fatalf("error reading response body %v", err)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Interceptor.ExecuteTrigger (-want, +got) = %s", diff)
			}
			if tt.wantContextHook != nil {
				_, ok := interceptors.InterceptedHook(ctx)
				if !ok {
					t.Error("failed to find the intercepted hook")
				}
			}
		})
	}
}
