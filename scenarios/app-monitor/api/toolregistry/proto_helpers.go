// Package toolregistry provides proto helper functions for app-monitor.
//
// This file provides helper functions for working with proto-generated
// tool types and converting between proto and native Go representations.
package toolregistry

import (
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
	commonpb "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// -----------------------------------------------------------------------------
// JsonValue Helpers
// -----------------------------------------------------------------------------

// StringValue creates a JsonValue containing a string.
func StringValue(s string) *commonpb.JsonValue {
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_StringValue{StringValue: s},
	}
}

// IntValue creates a JsonValue containing an integer.
func IntValue(i int) *commonpb.JsonValue {
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_IntValue{IntValue: int64(i)},
	}
}

// FloatValue creates a JsonValue containing a float.
func FloatValue(f float64) *commonpb.JsonValue {
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_DoubleValue{DoubleValue: f},
	}
}

// BoolValue creates a JsonValue containing a boolean.
func BoolValue(b bool) *commonpb.JsonValue {
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_BoolValue{BoolValue: b},
	}
}

// NullValue creates a JsonValue representing null.
func NullValue() *commonpb.JsonValue {
	return &commonpb.JsonValue{
		Kind: &commonpb.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE},
	}
}

// -----------------------------------------------------------------------------
// Tool Example Helpers
// -----------------------------------------------------------------------------

// NewToolExample creates a ToolExample with the given description and input.
func NewToolExample(description string, input map[string]interface{}) *toolspb.ToolExample {
	return &toolspb.ToolExample{
		Description: description,
		Input:       InterfaceMapToJsonObject(input),
	}
}

// InterfaceMapToJsonObject converts a Go map to a proto JsonObject.
func InterfaceMapToJsonObject(m map[string]interface{}) *commonpb.JsonObject {
	if m == nil {
		return nil
	}
	fields := make(map[string]*commonpb.JsonValue)
	for k, v := range m {
		fields[k] = InterfaceToJsonValue(v)
	}
	return &commonpb.JsonObject{Fields: fields}
}

// InterfaceToJsonValue converts a Go interface{} to a proto JsonValue.
func InterfaceToJsonValue(v interface{}) *commonpb.JsonValue {
	if v == nil {
		return NullValue()
	}

	switch val := v.(type) {
	case string:
		return StringValue(val)
	case int:
		return IntValue(val)
	case int32:
		return IntValue(int(val))
	case int64:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_IntValue{IntValue: val},
		}
	case float64:
		return FloatValue(val)
	case float32:
		return FloatValue(float64(val))
	case bool:
		return BoolValue(val)
	case []interface{}:
		values := make([]*commonpb.JsonValue, len(val))
		for i, item := range val {
			values[i] = InterfaceToJsonValue(item)
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ListValue{
				ListValue: &commonpb.JsonList{Values: values},
			},
		}
	case []string:
		values := make([]*commonpb.JsonValue, len(val))
		for i, s := range val {
			values[i] = StringValue(s)
		}
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ListValue{
				ListValue: &commonpb.JsonList{Values: values},
			},
		}
	case map[string]interface{}:
		return &commonpb.JsonValue{
			Kind: &commonpb.JsonValue_ObjectValue{
				ObjectValue: InterfaceMapToJsonObject(val),
			},
		}
	default:
		// Fallback to null for unsupported types
		return NullValue()
	}
}
