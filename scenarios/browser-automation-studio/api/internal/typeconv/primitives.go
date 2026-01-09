package typeconv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToString safely converts various types to string.
// Returns empty string if conversion fails.
func ToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case []byte:
		return string(v)
	default:
		return ""
	}
}

// ToInt safely converts various numeric types to int.
// Returns 0 if conversion fails.
func ToInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return 0
}

// ToBool safely converts various types to bool.
// Returns false if conversion fails.
func ToBool(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return false
}

// ToFloat safely converts various numeric types to float64.
// Returns 0 if conversion fails.
func ToFloat(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return 0
}

// ToTimePtr safely converts various types to *time.Time.
// Returns nil if conversion fails or value is empty.
func ToTimePtr(value any) *time.Time {
	switch v := value.(type) {
	case time.Time:
		return &v
	case *time.Time:
		return v
	case string:
		if v == "" {
			return nil
		}
		if ts, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return &ts
		}
		if ts, err := time.Parse(time.RFC3339, v); err == nil {
			return &ts
		}
	}
	return nil
}

// ToStringSlice safely converts various types to []string.
// Returns empty slice if conversion fails.
func ToStringSlice(value any) []string {
	result := make([]string, 0)
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		for _, item := range v {
			if str := ToString(item); str != "" {
				result = append(result, str)
			}
		}
	case string:
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// ToInterfaceSlice safely converts various types to []any.
// Handles []any, []map[string]any, map[string]any (values), and JSON-serializable types.
// Returns empty slice if conversion fails.
func ToInterfaceSlice(value any) []any {
	switch typed := value.(type) {
	case nil:
		return []any{}
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = typed[i]
		}
		return result
	case map[string]any:
		result := make([]any, 0, len(typed))
		for _, v := range typed {
			result = append(result, v)
		}
		return result
	default:
		bytes, err := json.Marshal(typed)
		if err != nil {
			return []any{}
		}
		var arr []any
		if err := json.Unmarshal(bytes, &arr); err != nil {
			return []any{}
		}
		return arr
	}
}

// DeepCloneMap creates a deep copy of a map[string]any, recursively cloning nested maps and slices.
// Returns nil if input is nil.
func DeepCloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	clone := make(map[string]any, len(source))
	for k, v := range source {
		clone[k] = DeepCloneValue(v)
	}
	return clone
}

// DeepCloneValue creates a deep copy of a value, handling maps, slices, and primitives.
// Used by DeepCloneMap for recursive cloning.
func DeepCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for k, v := range typed {
			cloned[k] = DeepCloneValue(v)
		}
		return cloned
	case []any:
		result := make([]any, len(typed))
		for i := range typed {
			result[i] = DeepCloneValue(typed[i])
		}
		return result
	case []string:
		return append([]string{}, typed...)
	default:
		// Primitives and other types are returned as-is (they're copied by value)
		return typed
	}
}

// AnyToJsonValue converts any Go value to a commonv1.JsonValue proto message.
// Handles primitives (bool, int, float, string, bytes), maps, slices, json.Number,
// and structpb.Value recursively. Falls back to JSON round-trip for unsupported types.
func AnyToJsonValue(v any) *commonv1.JsonValue {
	if v == nil {
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	}
	switch val := v.(type) {
	case *structpb.Value:
		// Unwrap structpb.Value and recursively convert
		return AnyToJsonValue(val.AsInterface())
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int8:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int16:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case uint:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case json.Number:
		// Prefer int if it parses cleanly, otherwise use float
		if i, err := val.Int64(); err == nil {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: i}}
		}
		if f, err := val.Float64(); err == nil {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: f}}
		}
		return nil
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: ToJsonObject(val)}}
	case []any:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: ToJsonList(val)}}
	default:
		// Fallback: JSON round-trip unknown types into a generic shape
		raw, err := json.Marshal(val)
		if err != nil {
			return nil
		}
		var tmp any
		if err := json.Unmarshal(raw, &tmp); err != nil {
			return nil
		}
		return AnyToJsonValue(tmp)
	}
}

// JsonValueToAny converts a commonv1.JsonValue proto message back to a Go value.
// Handles all JsonValue kinds: bool, int, double, string, null, object, and list.
// Returns nil for nil input or unrecognized kinds.
func JsonValueToAny(v *commonv1.JsonValue) any {
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
	case *commonv1.JsonValue_BytesValue:
		return k.BytesValue
	case *commonv1.JsonValue_ObjectValue:
		if k.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any, len(k.ObjectValue.Fields))
		for key, val := range k.ObjectValue.Fields {
			result[key] = JsonValueToAny(val)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if k.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(k.ListValue.Values))
		for _, val := range k.ListValue.Values {
			result = append(result, JsonValueToAny(val))
		}
		return result
	default:
		return nil
	}
}

// JsonValueMapToAny converts a map[string]*commonv1.JsonValue to a native Go map[string]any.
// Returns nil for nil or empty input.
func JsonValueMapToAny(source map[string]*commonv1.JsonValue) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = JsonValueToAny(value)
	}
	return result
}

// ToJsonValueMap converts a map[string]any to a map[string]*commonv1.JsonValue.
// Returns nil for nil or empty input. Skips entries that fail conversion.
func ToJsonValueMap(source map[string]any) map[string]*commonv1.JsonValue {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]*commonv1.JsonValue, len(source))
	for key, value := range source {
		if jsonVal := AnyToJsonValue(value); jsonVal != nil {
			result[key] = jsonVal
		}
	}
	return result
}

// ToJsonObject converts a map[string]any to a commonv1.JsonObject.
// Returns nil for nil or empty input. Skips entries that fail conversion.
func ToJsonObject(source map[string]any) *commonv1.JsonObject {
	if len(source) == 0 {
		return nil
	}
	result := &commonv1.JsonObject{
		Fields: make(map[string]*commonv1.JsonValue, len(source)),
	}
	for key, value := range source {
		if jsonVal := AnyToJsonValue(value); jsonVal != nil {
			result.Fields[key] = jsonVal
		}
	}
	return result
}

// ToJsonList converts a []any slice to a commonv1.JsonList.
// Returns nil for nil or empty input. Skips items that fail conversion.
func ToJsonList(items []any) *commonv1.JsonList {
	if len(items) == 0 {
		return nil
	}
	result := &commonv1.JsonList{
		Values: make([]*commonv1.JsonValue, 0, len(items)),
	}
	for _, item := range items {
		if jsonVal := AnyToJsonValue(item); jsonVal != nil {
			result.Values = append(result.Values, jsonVal)
		}
	}
	return result
}

// ToJsonObjectFromAny attempts to convert any value to a commonv1.JsonObject.
// For map[string]any, uses ToJsonObject directly. For other types, attempts
// conversion via AnyToJsonValue and extracts the object value.
func ToJsonObjectFromAny(value any) *commonv1.JsonObject {
	switch v := value.(type) {
	case map[string]any:
		return ToJsonObject(v)
	default:
		if jsonVal := AnyToJsonValue(v); jsonVal != nil {
			return jsonVal.GetObjectValue()
		}
		return nil
	}
}

// ToInt32Val safely converts various numeric types to int32.
// Returns 0 on failure. For a version that indicates success/failure, use ToInt32.
func ToInt32Val(v any) int32 {
	val, _ := ToInt32(v)
	return val
}

// ToInt32 safely converts various numeric types to int32.
// Returns value and ok=true on success, 0 and ok=false otherwise.
func ToInt32(v any) (int32, bool) {
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

// ToFloat64 safely converts various numeric types to float64.
// Returns value and ok=true on success, 0 and ok=false otherwise.
func ToFloat64(v any) (float64, bool) {
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
