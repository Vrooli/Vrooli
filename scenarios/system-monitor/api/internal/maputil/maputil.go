package maputil

import "strings"

// GetFloat64 extracts a float64 from a map, returning 0.0 if missing or wrong type.
func GetFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0.0
}

// GetInt extracts an int from a map, handling float64 JSON decoding.
func GetInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

// GetInt64 extracts an int64 from a map, handling float64 JSON decoding.
func GetInt64(m map[string]interface{}, key string) int64 {
	if val, ok := m[key].(int64); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int64(val)
	}
	return 0
}

// GetFloat64Slice extracts a []float64 from a map, handling []interface{} from JSON.
func GetFloat64Slice(m map[string]interface{}, key string) []float64 {
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

// GetString extracts a string from a map, returning "" if missing or wrong type.
func GetString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetBool extracts a bool from a map, handling string and numeric representations.
func GetBool(m map[string]interface{}, key string) bool {
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
