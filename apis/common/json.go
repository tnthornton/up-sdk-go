// Copyright 2024 Upbound Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/fieldpath"
)

// JSON represents any valid JSON value.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
//
// +protobuf=true
// +protobuf.options.marshal=false
// +protobuf.as=ProtoJSON
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:conversion-gen=false
type JSON struct {
	Object interface{} `json:"-"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ JSON) OpenAPISchemaType() []string { // nolint:golint
	// TODO: return actual types when anyOf is supported
	return nil
}

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (_ JSON) OpenAPISchemaFormat() string { return "" } // nolint:golint

// DeepCopy returns a deep copy of the JSON.
func (j *JSON) DeepCopy() *JSON {
	if j == nil {
		return nil
	}
	return &JSON{Object: runtime.DeepCopyJSONValue(j.Object)}
}

// DeepCopyInto copies the receiver, writing into out.
func (j *JSON) DeepCopyInto(target *JSON) {
	if target == nil {
		return
	}
	if j == nil {
		target.Object = nil // shouldn't happen
		return
	}
	target.Object = runtime.DeepCopyJSONValue(j.Object)
}

// MarshalJSON implements json.Marshaler.
func (j JSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Object)
}

// UnmarshalJSON implements json.Marshaler.
func (j *JSON) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Object)
}

// String returns the JSON representation of the object.
func (j *JSON) String() string {
	bs, _ := json.Marshal(j) // no way to handle error here
	return string(bs)
}

// JSONObject represents any valid JSON value of an object.
// These types are supported: bool, int64, float64, string, []interface{}, map[string]interface{} and nil.
//
// +protobuf=true
// +protobuf.options.marshal=false
// +protobuf.as=ProtoJSON
// +protobuf.options.(gogoproto.goproto_stringer)=false
// +k8s:conversion-gen=false
type JSONObject struct {
	Object map[string]interface{} `json:"-"`
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (j JSONObject) OpenAPISchemaType() []string {
	// TODO: return actual types when anyOf is supported
	return nil
} // nolint:golint

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (j JSONObject) OpenAPISchemaFormat() string { return "" } // nolint:golint

// DeepCopy returns a deep copy of the JSONObject.
func (j *JSONObject) DeepCopy() *JSONObject {
	if j == nil {
		return nil
	}
	return &JSONObject{Object: runtime.DeepCopyJSONValue(j.Object).(map[string]interface{})}
}

// DeepCopyInto copies the receiver, writing into out.
func (j *JSONObject) DeepCopyInto(target *JSONObject) {
	if target == nil {
		return
	}
	if j == nil {
		target.Object = nil // shouldn't happen
		return
	}
	target.Object = runtime.DeepCopyJSONValue(j.Object).(map[string]interface{})
}

// MarshalJSON implements json.Marshaler.
func (j JSONObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Object)
}

// UnmarshalJSON implements json.Marshaler.
func (j *JSONObject) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Object)
}

// String returns the JSON representation of the object.
func (j *JSONObject) String() string {
	bs, _ := json.Marshal(j) // nolint:errcheck // no way to handle error here
	return string(bs)
}

// GetCondition implements the Conditioned interface.
func (j *JSONObject) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	conditioned := xpv1.ConditionedStatus{}

	// The path is directly `status` because conditions are inline.
	if err := fieldpath.Pave(j.Object).GetValueInto("status", &conditioned); err != nil {
		return xpv1.Condition{}
	}
	return conditioned.GetCondition(ct)
}

// SetConditions implements the Conditioned interface.
func (j *JSONObject) SetConditions(c ...xpv1.Condition) {
	conditioned := xpv1.ConditionedStatus{}
	// The path is directly `status` because conditions are inline.
	_ = fieldpath.Pave(j.Object).GetValueInto("status", &conditioned)
	conditioned.SetConditions(c...)
	_ = fieldpath.Pave(j.Object).SetValue("status.conditions", conditioned.Conditions)
}
