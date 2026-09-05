// Package jsonval bridges Postgres JSONB payloads and the common.v1.JsonValue
// wire type used by the landing-page protobuf messages. Domain services store
// arbitrary metadata as JSONB and expose it to clients as
// map[string]*commonv1.JsonValue; these helpers convert losslessly in both
// directions and provide small typed constructors/readers for the handful of
// well-known metadata keys (features, subtitle, badge, highlight, …).
package jsonval

import (
	"encoding/json"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// FromJSONB decodes a JSONB byte slice into a JsonValue field map. Empty or
// invalid input yields a nil map (treated as "no metadata").
func FromJSONB(raw []byte) map[string]*commonv1.JsonValue {
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}
	if len(decoded) == 0 {
		return map[string]*commonv1.JsonValue{}
	}
	out := make(map[string]*commonv1.JsonValue, len(decoded))
	for k, v := range decoded {
		out[k] = New(v)
	}
	return out
}

// ToJSONB encodes a JsonValue field map back to a JSONB byte slice. A nil map
// encodes as an empty JSON object.
func ToJSONB(fields map[string]*commonv1.JsonValue) ([]byte, error) {
	if len(fields) == 0 {
		return []byte("{}"), nil
	}
	plain := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		plain[k] = AsInterface(v)
	}
	return json.Marshal(plain)
}

// New converts an arbitrary decoded-JSON Go value into a JsonValue.
func New(v interface{}) *commonv1.JsonValue {
	switch val := v.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []interface{}:
		values := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			values = append(values, New(item))
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: &commonv1.JsonList{Values: values}}}
	case map[string]interface{}:
		fields := make(map[string]*commonv1.JsonValue, len(val))
		for k, item := range val {
			fields[k] = New(item)
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: &commonv1.JsonObject{Fields: fields}}}
	default:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	}
}

// AsInterface converts a JsonValue back into a plain decoded-JSON Go value.
func AsInterface(v *commonv1.JsonValue) interface{} {
	if v == nil {
		return nil
	}
	switch k := v.Kind.(type) {
	case *commonv1.JsonValue_BoolValue:
		return k.BoolValue
	case *commonv1.JsonValue_IntValue:
		return k.IntValue
	case *commonv1.JsonValue_DoubleValue:
		return k.DoubleValue
	case *commonv1.JsonValue_StringValue:
		return k.StringValue
	case *commonv1.JsonValue_BytesValue:
		return k.BytesValue
	case *commonv1.JsonValue_ListValue:
		if k.ListValue == nil {
			return []interface{}{}
		}
		out := make([]interface{}, 0, len(k.ListValue.Values))
		for _, item := range k.ListValue.Values {
			out = append(out, AsInterface(item))
		}
		return out
	case *commonv1.JsonValue_ObjectValue:
		if k.ObjectValue == nil {
			return map[string]interface{}{}
		}
		out := make(map[string]interface{}, len(k.ObjectValue.Fields))
		for key, item := range k.ObjectValue.Fields {
			out[key] = AsInterface(item)
		}
		return out
	default:
		return nil
	}
}

// String builds a string-kind JsonValue.
func String(s string) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: s}}
}

// Bool builds a bool-kind JsonValue.
func Bool(b bool) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: b}}
}

// StringList builds a list-kind JsonValue from string items.
func StringList(items []string) *commonv1.JsonValue {
	values := make([]*commonv1.JsonValue, 0, len(items))
	for _, item := range items {
		values = append(values, String(item))
	}
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: &commonv1.JsonList{Values: values}}}
}

// StringSlice extracts a []string from a list-kind JsonValue, skipping
// non-string / empty entries. Returns nil when the key is absent or not a list.
func StringSlice(v *commonv1.JsonValue) []string {
	if v == nil {
		return nil
	}
	list, ok := v.Kind.(*commonv1.JsonValue_ListValue)
	if !ok || list.ListValue == nil {
		return nil
	}
	var out []string
	for _, item := range list.ListValue.Values {
		if s, ok := item.Kind.(*commonv1.JsonValue_StringValue); ok && s.StringValue != "" {
			out = append(out, s.StringValue)
		}
	}
	return out
}
