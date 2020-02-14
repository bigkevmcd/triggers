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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
)

type Interceptor struct {
	Logger     *zap.SugaredLogger
	FormDecode *triggersv1.FormDecodeInterceptor
}

func NewInterceptor(fd *triggersv1.FormDecodeInterceptor, l *zap.SugaredLogger) interceptors.Interceptor {
	return &Interceptor{
		Logger:     l,
		FormDecode: fd,
	}
}

func (w *Interceptor) ExecuteTrigger(r *http.Request) (*http.Response, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse form data: %w", err)
	}
	var data interface{} = r.PostForm
	if w.FormDecode.Flatten {
		data = flattenMap(r.PostForm)
	}

	response := map[string]interface{}{w.FormDecode.Prefix: data}
	payload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal form data: %w", err)
	}

	r.Header.Set("Content-Type", "application/json")
	return &http.Response{
		Header: r.Header,
		Body:   ioutil.NopCloser(bytes.NewBuffer(payload)),
	}, nil
}

func flattenMap(m url.Values) map[string]string {
	flattened := make(map[string]string)
	for k, v := range m {
		flattened[k] = v[0]
	}
	return flattened
}
