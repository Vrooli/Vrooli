package workflow

import (
	"encoding/json"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// Utility functions for type conversion used across the workflow package.

func toInt32(v any) (int32, bool) {
	switch val := v.(type) {
	case int:
		return int32(val), true
	case int32:
		return val, true
	case int64:
		return int32(val), true
	case float64:
		return int32(val), true
	case float32:
		return int32(val), true
	case json.Number:
		if i, err := val.Int64(); err == nil {
			return int32(i), true
		}
	}
	return 0, false
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case json.Number:
		if f, err := val.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

func anyToJsonValue(v any) *commonv1.JsonValue {
	if v == nil {
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_NullValue{},
		}
	}
	switch val := v.(type) {
	case bool:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_BoolValue{BoolValue: val},
		}
	case int:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)},
		}
	case int32:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)},
		}
	case int64:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_IntValue{IntValue: val},
		}
	case float64:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val},
		}
	case float32:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)},
		}
	case string:
		return &commonv1.JsonValue{
			Kind: &commonv1.JsonValue_StringValue{StringValue: val},
		}
	default:
		// For complex types, marshal to string
		if b, err := json.Marshal(v); err == nil {
			return &commonv1.JsonValue{
				Kind: &commonv1.JsonValue_StringValue{StringValue: string(b)},
			}
		}
		return nil
	}
}

func jsonValueToAny(v *commonv1.JsonValue) any {
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
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_ObjectValue:
		if k.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any)
		for key, val := range k.ObjectValue.Fields {
			result[key] = jsonValueToAny(val)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if k.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(k.ListValue.Values))
		for _, val := range k.ListValue.Values {
			result = append(result, jsonValueToAny(val))
		}
		return result
	default:
		return nil
	}
}
