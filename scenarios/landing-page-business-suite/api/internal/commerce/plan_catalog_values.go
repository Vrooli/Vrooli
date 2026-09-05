package commerce

import (
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// toJsonValue converts a Go value to a commonv1.JsonValue.
func toJsonValue(v any) *commonv1.JsonValue {
	switch val := v.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		// JSON numbers are parsed as float64; check if it's a whole number
		if val == float64(int64(val)) {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for key, value := range val {
			if nested := toJsonValue(value); nested != nil {
				obj[key] = nested
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{
			ObjectValue: &commonv1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := toJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
			ListValue: &commonv1.JsonList{Values: items},
		}}
	default:
		return nil
	}
}

// jsonValueToAny converts a JsonValue to a Go any type.
func jsonValueToAny(v *commonv1.JsonValue) any {
	if v == nil {
		return nil
	}
	switch kind := v.Kind.(type) {
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_BoolValue:
		return kind.BoolValue
	case *commonv1.JsonValue_IntValue:
		return kind.IntValue
	case *commonv1.JsonValue_DoubleValue:
		return kind.DoubleValue
	case *commonv1.JsonValue_StringValue:
		return kind.StringValue
	case *commonv1.JsonValue_BytesValue:
		return kind.BytesValue
	case *commonv1.JsonValue_ObjectValue:
		if kind.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any, len(kind.ObjectValue.Fields))
		for k, fv := range kind.ObjectValue.Fields {
			result[k] = jsonValueToAny(fv)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if kind.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(kind.ListValue.Values))
		for _, item := range kind.ListValue.Values {
			result = append(result, jsonValueToAny(item))
		}
		return result
	default:
		return nil
	}
}

// newStringJsonValue creates a JsonValue with a string.
func newStringJsonValue(s string) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: s}}
}

// newBoolJsonValue creates a JsonValue with a bool.
func newBoolJsonValue(b bool) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: b}}
}

// newListJsonValue creates a JsonValue with a list of JsonValues.
func newListJsonValue(values []*commonv1.JsonValue) *commonv1.JsonValue {
	return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
		ListValue: &commonv1.JsonList{Values: values},
	}}
}

func buildPlanMetadata(subtitle, badge, ctaLabel *string, highlight *bool, features []string) map[string]*commonv1.JsonValue {
	metadata := make(map[string]*commonv1.JsonValue)

	if subtitle != nil {
		if trimmed := strings.TrimSpace(*subtitle); trimmed != "" {
			metadata["subtitle"] = newStringJsonValue(trimmed)
		}
	}
	if badge != nil {
		if trimmed := strings.TrimSpace(*badge); trimmed != "" {
			metadata["badge"] = newStringJsonValue(trimmed)
		}
	}
	if ctaLabel != nil {
		if trimmed := strings.TrimSpace(*ctaLabel); trimmed != "" {
			metadata["cta_label"] = newStringJsonValue(trimmed)
		}
	}
	if highlight != nil && *highlight {
		metadata["highlight"] = newBoolJsonValue(true)
	}

	var sanitized []string
	for _, feature := range features {
		if trimmed := strings.TrimSpace(feature); trimmed != "" {
			sanitized = append(sanitized, trimmed)
		}
	}
	if len(sanitized) > 0 {
		listValues := make([]*commonv1.JsonValue, 0, len(sanitized))
		for _, feature := range sanitized {
			listValues = append(listValues, newStringJsonValue(feature))
		}
		metadata["features"] = newListJsonValue(listValues)
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}
