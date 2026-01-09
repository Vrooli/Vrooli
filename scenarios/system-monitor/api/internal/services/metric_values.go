package services

import (
	"strings"
)

func getFloat64Value(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

func getIntValue(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

func getInt64Value(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(int64); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	return 0
}

func getFloat64Slice(m map[string]interface{}, key string) []float64 {
	if val, ok := m[key].([]float64); ok {
		return val
	}
	if val, ok := m[key].([]interface{}); ok {
		result := make([]float64, 0, len(val))
		for _, v := range val {
			if f, ok := v.(float64); ok {
				result = append(result, f)
			}
		}
		return result
	}
	return []float64{}
}

func getStringValue(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getBoolValue(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	if val, ok := m[key].(string); ok {
		return strings.EqualFold(val, "true") || val == "1"
	}
	if val, ok := m[key].(float64); ok {
		return val != 0
	}
	return false
}
