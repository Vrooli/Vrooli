// Package typeconv provides type conversion utilities.
package typeconv

import (
	"fmt"
	"math"
)

// WrapJsonValue wraps a Go value in the common.v1.JsonValue oneof shape.
// This is used when converting plain JSON values to proto JsonValue format.
//
// The resulting map can be marshaled directly to protojson and will be
// correctly parsed into a common.v1.JsonValue message.
//
// Mappings:
//   - nil -> {"null_value": "NULL_VALUE"}
//   - string -> {"string_value": ...}
//   - bool -> {"bool_value": ...}
//   - int/int32/int64/uint/uint32 -> {"int_value": ...}
//   - uint64 (if > MaxInt64) -> {"double_value": ...}
//   - float32/float64 (integral) -> {"int_value": ...}
//   - float32/float64 (non-integral) -> {"double_value": ...}
//   - []any -> {"list_value": {"values": [...]}}
//   - map[string]any -> {"object_value": {"fields": {...}}}
//   - other -> {"string_value": fmt.Sprintf("%v", v)}
//
// If the value is already wrapped (detected by looking for known JsonValue keys),
// it is returned as-is.
func WrapJsonValue(v any) any {
	if v == nil {
		return map[string]any{"null_value": "NULL_VALUE"}
	}
	switch vv := v.(type) {
	case map[string]any:
		// Already in JsonValue JSON form (oneof wrapper), keep as-is.
		if looksLikeJsonValueWrapper(vv) {
			return vv
		}
		fields := make(map[string]any, len(vv))
		for k, child := range vv {
			fields[k] = WrapJsonValue(child)
		}
		return map[string]any{
			"object_value": map[string]any{
				"fields": fields,
			},
		}
	case []any:
		values := make([]any, 0, len(vv))
		for _, child := range vv {
			values = append(values, WrapJsonValue(child))
		}
		return map[string]any{
			"list_value": map[string]any{
				"values": values,
			},
		}
	case string:
		return map[string]any{"string_value": vv}
	case bool:
		return map[string]any{"bool_value": vv}
	case float64:
		if isJSONIntegral(vv) {
			return map[string]any{"int_value": int64(vv)}
		}
		return map[string]any{"double_value": vv}
	case float32:
		f := float64(vv)
		if isJSONIntegral(f) {
			return map[string]any{"int_value": int64(f)}
		}
		return map[string]any{"double_value": f}
	case int:
		return map[string]any{"int_value": int64(vv)}
	case int32:
		return map[string]any{"int_value": int64(vv)}
	case int64:
		return map[string]any{"int_value": vv}
	case uint:
		return map[string]any{"int_value": int64(vv)}
	case uint32:
		return map[string]any{"int_value": int64(vv)}
	case uint64:
		if vv > math.MaxInt64 {
			return map[string]any{"double_value": float64(vv)}
		}
		return map[string]any{"int_value": int64(vv)}
	default:
		// Best-effort: preserve as string rather than failing hard.
		return map[string]any{"string_value": fmt.Sprintf("%v", v)}
	}
}

// looksLikeJsonValueWrapper returns true if the map appears to already be
// a JsonValue wrapper (has one of the known oneof field names).
func looksLikeJsonValueWrapper(m map[string]any) bool {
	if m == nil {
		return false
	}
	// Accept both proto field names (snake_case) and JSON names (lowerCamel).
	for _, key := range []string{
		"bool_value", "boolValue",
		"int_value", "intValue",
		"double_value", "doubleValue",
		"string_value", "stringValue",
		"object_value", "objectValue",
		"list_value", "listValue",
		"null_value", "nullValue",
		"bytes_value", "bytesValue",
	} {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

// isJSONIntegral returns true if the float64 value represents an integer
// that can be safely converted to int64.
func isJSONIntegral(v float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	if math.Trunc(v) != v {
		return false
	}
	if v < float64(math.MinInt64) || v > float64(math.MaxInt64) {
		return false
	}
	return true
}

// NormalizeJsonValueMaps wraps all values in the specified map keys with JsonValue format.
// This is useful for normalizing request parameters that should be JsonValue maps.
func NormalizeJsonValueMaps(params map[string]any, keys ...string) {
	for _, key := range keys {
		rawMap, ok := params[key].(map[string]any)
		if !ok || rawMap == nil {
			continue
		}
		normalized := make(map[string]any, len(rawMap))
		for k, v := range rawMap {
			normalized[k] = WrapJsonValue(v)
		}
		params[key] = normalized
	}
}
