/*
Copyright 2022 The Tekton Authors
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

package v1beta1

import (
	"context"
	"fmt"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"knative.dev/pkg/apis"
)

// Validate implements apis.Validatable
func (tr TaskResult) Validate(ctx context.Context) (errs *apis.FieldError) {
	if !resultNameFormatRegex.MatchString(tr.Name) {
		return apis.ErrInvalidKeyName(tr.Name, "name", fmt.Sprintf("Name must consist of alphanumeric characters, '-', '_', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my-name',  or 'my_name', regex used for validation is '%s')", ResultNameFormat))
	}

	switch {
	case tr.Type == ResultsTypeObject:
		errs = errs.Also(validateObjectResult(tr))
	case tr.Type == ResultsTypeArray:
	// Resources created before the result. Type was introduced may not have Type set
	// and should be considered valid
	case tr.Type == "":
	// By default, the result type is string
	case tr.Type != ResultsTypeString:
		errs = errs.Also(apis.ErrInvalidValue(tr.Type, "type", "type must be string"))
	}
	return errs.Also(tr.validateValue(ctx))
}

// validateObjectResult validates the object result and check if the Properties is missing
// for Properties values it will check if the type is string.
func validateObjectResult(tr TaskResult) (errs *apis.FieldError) {
	if ParamType(tr.Type) == ParamTypeObject && tr.Properties == nil {
		return apis.ErrMissingField(tr.Name + ".properties")
	}

	invalidKeys := []string{}
	for key, propertySpec := range tr.Properties {
		if propertySpec.Type != ParamTypeString {
			invalidKeys = append(invalidKeys, key)
		}
	}

	if len(invalidKeys) != 0 {
		return &apis.FieldError{
			Message: fmt.Sprintf("The value type specified for these keys %v is invalid, the type must be string", invalidKeys),
			Paths:   []string{tr.Name + ".properties"},
		}
	}
	return nil
}

// validateValue validates the value of the TaskResult.
// It requires the value is of type string
// and format $(steps.<stepName>.results.<resultName>)
func (tr TaskResult) validateValue(ctx context.Context) (errs *apis.FieldError) {
	if tr.Value == nil {
		return nil
	}
	if tr.Value.Type != ParamTypeString {
		return &apis.FieldError{
			Message: fmt.Sprintf(
				"Invalid Type. Wanted string but got: \"%v\"", tr.Value.Type),
			Paths: []string{
				tr.Name + ".type",
			},
		}
	}
	if tr.Value.StringVal != "" {
		stepName, resultName, err := v1.ExtractStepResultName(tr.Value.StringVal)
		if err != nil {
			return &apis.FieldError{
				Message: err.Error(),
				Paths:   []string{tr.Name + ".value"},
			}
		}
		if e := validation.IsDNS1123Label(stepName); len(e) > 0 {
			errs = errs.Also(&apis.FieldError{
				Message: fmt.Sprintf("invalid extracted step name %q", stepName),
				Paths:   []string{tr.Name + ".value"},
				Details: "stepName in $(steps.<stepName>.results.<resultName>) must be a valid DNS Label, For more info refer to https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names",
			})
		}
		if !resultNameFormatRegex.MatchString(resultName) {
			errs = errs.Also(&apis.FieldError{
				Message: fmt.Sprintf("invalid extracted result name %q", resultName),
				Paths:   []string{tr.Name + ".value"},
				Details: fmt.Sprintf("resultName in $(steps.<stepName>.results.<resultName>) must consist of alphanumeric characters, '-', '_', and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my-name',  or 'my_name', regex used for validation is '%s')", ResultNameFormat),
			})
		}
	}
	return errs
}
