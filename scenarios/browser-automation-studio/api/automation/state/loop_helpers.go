package state

import (
	"encoding/json"
	"strings"
)

// ExtractLoopItems extracts loop items from params, supporting multiple parameter names
// and variable sources for backward compatibility.
func ExtractLoopItems(params map[string]any, state *ExecutionState) []any {
	if params == nil {
		return nil
	}
	// Try direct items array
	if raw, ok := params["loopItems"]; ok {
		if arr := coerceArray(raw); len(arr) > 0 {
			return arr
		}
	}
	if raw, ok := params["items"]; ok {
		if arr := coerceArray(raw); len(arr) > 0 {
			return arr
		}
	}
	// Try variable source
	source := ""
	if raw, ok := params["arraySource"]; ok {
		source, _ = raw.(string)
	}
	if source == "" {
		if raw, ok := params["loopArraySource"]; ok {
			source, _ = raw.(string)
		}
	}
	if source != "" {
		if v, ok := state.Get(source); ok {
			if arr := coerceArray(v); len(arr) > 0 {
				return arr
			}
		}
	}
	return nil
}

// EvaluateLoopCondition evaluates a loop condition using the given params and state.
func EvaluateLoopCondition(params map[string]any, state *ExecutionState) bool {
	if params == nil {
		return false
	}
	condType := strings.ToLower(strings.TrimSpace(StringValue(params, "conditionType")))
	if condType == "" {
		condType = strings.ToLower(strings.TrimSpace(StringValue(params, "loopConditionType")))
	}
	if condType == "" {
		condType = "variable"
	}

	interp := NewInterpolator(state)

	switch condType {
	case "variable":
		name := StringValue(params, "conditionVariable")
		if name == "" {
			name = StringValue(params, "loopConditionVariable")
		}
		if name == "" {
			return false
		}
		op := strings.ToLower(strings.TrimSpace(StringValue(params, "conditionOperator")))
		if op == "" {
			op = strings.ToLower(strings.TrimSpace(StringValue(params, "loopConditionOperator")))
		}
		expected := firstPresent(params, "conditionValue", "loopConditionValue")
		current, ok := state.Get(name)
		if !ok {
			return false
		}
		return CompareValues(current, expected, op)
	case "expression":
		expr := strings.TrimSpace(StringValue(params, "conditionExpression"))
		if expr == "" {
			expr = strings.TrimSpace(StringValue(params, "loopConditionExpression"))
		}
		if expr == "" {
			return false
		}
		result, ok := interp.EvaluateExpression(expr)
		if !ok {
			return false
		}
		return result
	default:
		return false
	}
}

// StringValue safely extracts a string from params.
func StringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// IntValue safely extracts an integer from params, supporting int, int64, float64, float32, and string types.
func IntValue(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	return CoerceToInt(v)
}

// CoerceToInt converts various types to int, handling JSON's float64 numbers and strings.
func CoerceToInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case string:
		// Support string-encoded numbers (e.g., "1000" from form data)
		if i, err := parseInt(t); err == nil {
			return i
		}
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

// FloatValue safely extracts a float64 from params.
func FloatValue(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, _ := CoerceToFloat(v)
	return f
}

// CoerceToFloat converts various types to float64.
func CoerceToFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case int32:
		return float64(t), true
	case string:
		if f, err := parseFloat(t); err == nil {
			return f, true
		}
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// BoolValue safely extracts a boolean from params.
func BoolValue(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := parseBool(t)
		return b
	case int:
		return t != 0
	case float64:
		return t != 0
	}
	return false
}

// CoerceArray coerces various types to []any.
func CoerceArray(raw any) []any {
	return coerceArray(raw)
}

func coerceArray(raw any) []any {
	switch v := raw.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		var decoded []any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}
	return nil
}

func firstPresent(params map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := params[k]; ok {
			return v
		}
	}
	return nil
}

// NormalizeVariableValue normalizes a variable value based on its declared type.
func NormalizeVariableValue(value any, valueType string) any {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "boolean", "bool":
		if b, ok := parseBoolValue(value); ok {
			return b
		}
	case "number", "float", "int":
		if f, ok := ToFloat(value); ok {
			if strings.Contains(strings.ToLower(valueType), "int") {
				return int(f)
			}
			return f
		}
	case "json":
		if s, ok := value.(string); ok {
			var decoded any
			if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &decoded); err == nil {
				return decoded
			}
		}
	}
	return value
}

func parseBoolValue(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		return parseBool(v)
	}
	return false, false
}

// Simple parsing helpers to avoid importing strconv in multiple places
func parseInt(s string) (int, error) {
	var i int
	err := json.Unmarshal([]byte(s), &i)
	if err != nil {
		// Try float then truncate
		var f float64
		if err := json.Unmarshal([]byte(s), &f); err == nil {
			return int(f), nil
		}
	}
	return i, err
}

func parseFloat(s string) (float64, error) {
	var f float64
	err := json.Unmarshal([]byte(s), &f)
	return f, err
}

func parseBool(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true, true
	case "false", "0", "no":
		return false, true
	}
	return false, false
}
