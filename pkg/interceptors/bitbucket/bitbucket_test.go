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

package bitbucket

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/stash"
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
	var pushEvent = mustReadFixture("testdata/repo_refs_changed.json")
	type args struct {
		payload   io.ReadCloser
		secret    *corev1.Secret
		signature string
		eventType string
	}
	tests := []struct {
		name            string
		Bitbucket       *triggersv1.BitBucketInterceptor
		args            args
		want            []byte
		wantErr         bool
		wantContextHook scm.Webhook
	}{
		{
			name:      "no secret",
			Bitbucket: &triggersv1.BitBucketInterceptor{},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
				eventType: "repo:refs_changed",
				signature: "foo",
			},
			want:    pushEvent,
			wantErr: false,
		},
		{
			name: "invalid header for secret",
			Bitbucket: &triggersv1.BitBucketInterceptor{
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
				payload: ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		},
		{
			name: "valid header for secret",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
			},
			args: args{
				signature: test.SHA1Signature(t, testSecret, pushEvent),
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": testSecret,
					},
				},
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
				eventType: "repo:refs_changed",
			},
			wantErr: false,
			want:    pushEvent,
		},
		{
			name: "matching event",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
				eventType: "repo:refs_changed",
			},
			wantErr: false,
			want:    pushEvent,
		},
		{
			name: "no matching event",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				EventTypes: []string{"pr:opened"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
				eventType: "repo:refs_changed",
			},
			wantErr: true,
		},
		{
			name: "valid header for secret and matching event",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				signature: test.SHA1Signature(t, testSecret, pushEvent),
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": testSecret,
					},
				},
				eventType: "repo:refs_changed",
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
			},
			wantErr: false,
			want:    pushEvent,
		},
		{
			name: "valid header for secret, but no matching event",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
			},
			args: args{
				signature: test.SHA1Signature(t, testSecret, pushEvent),
				secret: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "mysecret",
					},
					Data: map[string][]byte{
						"token": testSecret,
					},
				},
				eventType: "event",
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
			},
			wantErr: true,
		},
		{
			name: "invalid header for secret, but matching event",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				SecretRef: &triggersv1.SecretRef{
					SecretName: "mysecret",
					SecretKey:  "token",
				},
				EventTypes: []string{"pr:opened", "repo:refs_changed"},
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
				eventType: "pr:opened",
				payload:   ioutil.NopCloser(bytes.NewBufferString("somepayload")),
			},
			wantErr: true,
		}, {
			name:      "nil body does not panic",
			Bitbucket: &triggersv1.BitBucketInterceptor{},
			args: args{
				payload:   nil,
				signature: "foo",
				eventType: "pr:opened",
			},
			want:    []byte{},
			wantErr: false,
		},
		{
			name: "adding the hook to the context",
			Bitbucket: &triggersv1.BitBucketInterceptor{
				EventTypes: []string{"repo:refs_changed"},
			},
			args: args{
				payload:   ioutil.NopCloser(bytes.NewBuffer(pushEvent)),
				eventType: "repo:refs_changed",
			},
			wantContextHook: &scm.PushHook{Ref: "refs/heads/master"},
			want:            pushEvent,
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
					"Content-Type": []string{"application/json"},
				},
			}
			if tt.args.eventType != "" {
				request.Header.Add("X-Event-Key", tt.args.eventType)
			}
			if tt.args.signature != "" {
				request.Header.Add("X-Hub-Signature", tt.args.signature)
			}
			if tt.args.secret != nil {
				ns := tt.Bitbucket.SecretRef.Namespace
				if ns == "" {
					ns = metav1.NamespaceDefault
				}
				if _, err := kubeClient.CoreV1().Secrets(ns).Create(tt.args.secret); err != nil {
					t.Error(err)
				}
			}
			w := &Interceptor{
				KubeClientSet: kubeClient,
				Bitbucket:     tt.Bitbucket,
				Logger:        logger,
				scmClient:     stash.NewDefault(),
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
					t.Errorf("failed to find the intercepted hook")
				}
			}

		})
	}
}

func mustReadFixture(filename string) []byte {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return b
}
