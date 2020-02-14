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

package formdecode

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"testing"

	"github.com/tektoncd/pipeline/pkg/logging"
	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
)

const (
	mimeJSON = "application/json"
	mimeForm = "application/x-www-form-urlencoded"
)

func TestInterceptor_ExecuteTrigger_Signature(t *testing.T) {
	tests := []struct {
		name       string
		FormDecode *triggersv1.FormDecodeInterceptor
		payload    io.ReadCloser
		want       []byte
	}{
		{
			name:       "unflattened body",
			FormDecode: &triggersv1.FormDecodeInterceptor{Prefix: "foo", Flatten: false},
			payload:    ioutil.NopCloser(bytes.NewBufferString(`field1=value1&field2=value2`)),
			want:       []byte(`{"foo":{"field1":["value1"],"field2":["value2"]}}`),
		},
		{
			name:       "flattened body",
			FormDecode: &triggersv1.FormDecodeInterceptor{Prefix: "bar", Flatten: true},
			payload:    ioutil.NopCloser(bytes.NewBufferString(`field1=value1&field2=value2`)),
			want:       []byte(`{"bar":{"field1":"value1","field2":"value2"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			request := &http.Request{
				Method: http.MethodPost,
				Body:   tt.payload,
				Header: http.Header{
					"Content-Type": []string{mimeForm},
				},
			}
			w := &Interceptor{
				FormDecode: tt.FormDecode,
				Logger:     logger,
			}
			resp, err := w.ExecuteTrigger(request)
			if err != nil {
				t.Errorf("Interceptor.ExecuteTrigger() error = %v", err)
				return
			}
			if v := resp.Header.Get("Content-Type"); v != mimeJSON {
				t.Errorf("Interceptor.ExecuteTrigger() %s Content-Type incorrect got %s, want %s", tt.name, v, mimeJSON)
			}
			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("error reading response body %v", err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Interceptor.ExecuteTrigger() %s got %s, want %s", tt.name, got, tt.want)
			}
		})
	}
}

func TestInterceptor_ExecuteTrigger_Errors(t *testing.T) {
	tests := []struct {
		name        string
		FormDecode  *triggersv1.FormDecodeInterceptor
		payload     []byte
		contentType string
		want        string
	}{
		{
			name: "invalid query body",
			FormDecode: &triggersv1.FormDecodeInterceptor{
				Prefix: "foo",
			},
			payload:     []byte(`key1&&=2`),
			contentType: mimeForm,
			want:        "failed to parse form data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logging.NewLogger("", "")
			w := &Interceptor{
				FormDecode: tt.FormDecode,
				Logger:     logger,
			}
			request := &http.Request{
				Body: ioutil.NopCloser(bytes.NewBuffer(tt.payload)),
				Header: http.Header{
					"Content-Type": []string{mimeForm},
				},
			}
			_, err := w.ExecuteTrigger(request)
			if !matchError(t, tt.want, err) {
				t.Errorf("executeTrigger() got %v, want %s", err, tt.want)
				return
			}
		})
	}
}

func matchError(t *testing.T, s string, e error) bool {
	t.Helper()
	if e == nil {
		return false
	}
	match, err := regexp.MatchString(s, e.Error())
	if err != nil {
		t.Fatal(err)
	}
	return match
}
