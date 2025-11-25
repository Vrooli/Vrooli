package typeconv

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
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
