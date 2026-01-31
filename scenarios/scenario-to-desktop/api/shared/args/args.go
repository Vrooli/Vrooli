// Package args provides argument extraction helpers for tool execution.
package args

// GetString extracts a string argument with a default fallback.
func GetString(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultValue
}

// GetStringArray extracts a string array argument.
func GetStringArray(args map[string]interface{}, key string) []string {
	if v, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if v, ok := args[key].([]string); ok {
		return v
	}
	return nil
}

// GetBool extracts a boolean argument with a default fallback.
func GetBool(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// GetInt extracts an integer argument with a default fallback.
// Handles both int and float64 (JSON unmarshals numbers as float64).
func GetInt(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultValue
}

// GetMap extracts a map argument.
func GetMap(args map[string]interface{}, key string) map[string]interface{} {
	if v, ok := args[key].(map[string]interface{}); ok {
		return v
	}
	return nil
}

// RequireString extracts a string argument and returns an error if it's empty.
func RequireString(args map[string]interface{}, key string) (string, error) {
	v := GetString(args, key, "")
	if v == "" {
		return "", &RequiredArgError{Key: key}
	}
	return v, nil
}

// RequireStringArray extracts a string array argument and returns an error if it's empty.
func RequireStringArray(args map[string]interface{}, key string) ([]string, error) {
	v := GetStringArray(args, key)
	if len(v) == 0 {
		return nil, &RequiredArgError{Key: key}
	}
	return v, nil
}

// RequireMap extracts a map argument and returns an error if it's nil.
func RequireMap(args map[string]interface{}, key string) (map[string]interface{}, error) {
	v := GetMap(args, key)
	if v == nil {
		return nil, &RequiredArgError{Key: key}
	}
	return v, nil
}

// RequiredArgError is returned when a required argument is missing or empty.
type RequiredArgError struct {
	Key string
}

func (e *RequiredArgError) Error() string {
	return e.Key + " is required"
}
