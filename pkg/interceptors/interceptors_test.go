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

package interceptors

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
)

func TestInterceptedHookWithHook(t *testing.T) {
	ctx := context.Background()
	pushHook := &scm.PushHook{Ref: "my-test-ref"}
	hookCtx := WithHook(ctx, pushHook)

	hook, ok := InterceptedHook(hookCtx)

	if !ok {
		t.Fatal("failed to find the hook in the context")
	}
	if diff := cmp.Diff(pushHook, hook); diff != "" {
		t.Fatalf("hook comparison failed:\n%s", diff)
	}
}

func TestInterceptedHookWithNoHook(t *testing.T) {
	ctx := context.Background()

	hook, ok := InterceptedHook(ctx)
	if ok {
		t.Errorf("found a hook in the context: %#v", hook)
	}
	if hook != nil {
		t.Errorf("InterceptedHook with no hook, got %#v, want nil", hook)
	}
}
