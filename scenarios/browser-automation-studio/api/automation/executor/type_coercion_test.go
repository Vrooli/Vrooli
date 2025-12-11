package executor

import (
	"encoding/json"
	"testing"
)

func TestCoerceToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		// Direct integer types
		{"int", 42, 42},
		{"int8", int8(127), 127},
		{"int16", int16(32767), 32767},
		{"int32", int32(2147483647), 2147483647},
		{"int64", int64(9223372036854775807), 9223372036854775807},

		// Unsigned integer types
		{"uint", uint(42), 42},
		{"uint8", uint8(255), 255},
		{"uint16", uint16(65535), 65535},
		{"uint32", uint32(4294967295), 4294967295},

		// Float types (common from JSON decoding)
		{"float32", float32(42.9), 42},
		{"float64", float64(42.9), 42},
		{"float64_whole", float64(100.0), 100},

		// String types (common from form data or query params)
		{"string_int", "1000", 1000},
		{"string_negative", "-42", -42},
		{"string_float", "1.5", 1},
		{"string_invalid", "not_a_number", 0},
		{"string_empty", "", 0},

		// json.Number (from json.Decoder with UseNumber())
		{"json_number", json.Number("12345"), 12345},

		// Edge cases
		{"nil", nil, 0},
		{"bool_true", true, 0},
		{"bool_false", false, 0},
		{"map", map[string]any{}, 0},
		{"slice", []int{1, 2, 3}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := coerceToInt(tt.input)
			if result != tt.expected {
				t.Errorf("coerceToInt(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCoerceToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		expectOK bool
	}{
		// Float types
		{"float64", float64(3.14159), 3.14159, true},
		{"float32", float32(2.5), 2.5, true},

		// Integer types
		{"int", 42, 42.0, true},
		{"int64", int64(1000), 1000.0, true},
		{"int32", int32(500), 500.0, true},

		// String types
		{"string_float", "3.14159", 3.14159, true},
		{"string_int", "42", 42.0, true},
		{"string_invalid", "not_a_number", 0, false},

		// json.Number
		{"json_number", json.Number("99.99"), 99.99, true},

		// Invalid types
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
		{"map", map[string]any{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := coerceToFloat(tt.input)
			if ok != tt.expectOK {
				t.Errorf("coerceToFloat(%v) ok = %v, expected %v", tt.input, ok, tt.expectOK)
			}
			if tt.expectOK && result != tt.expected {
				t.Errorf("coerceToFloat(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIntValue(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		key      string
		expected int
	}{
		{"nil_map", nil, "timeout", 0},
		{"missing_key", map[string]any{}, "timeout", 0},
		{"int_value", map[string]any{"timeout": 5000}, "timeout", 5000},
		{"float64_value", map[string]any{"timeout": float64(3000)}, "timeout", 3000},
		{"string_value", map[string]any{"timeout": "2500"}, "timeout", 2500},
		{"json_decoded", map[string]any{"timeout": float64(1500)}, "timeout", 1500}, // JSON numbers are float64
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := intValue(tt.params, tt.key)
			if result != tt.expected {
				t.Errorf("intValue(%v, %q) = %d, expected %d", tt.params, tt.key, result, tt.expected)
			}
		})
	}
}

func TestFloatValue(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		key      string
		expected float64
	}{
		{"nil_map", nil, "ratio", 0},
		{"missing_key", map[string]any{}, "ratio", 0},
		{"float_value", map[string]any{"ratio": 0.75}, "ratio", 0.75},
		{"int_value", map[string]any{"ratio": 50}, "ratio", 50.0},
		{"string_value", map[string]any{"ratio": "0.5"}, "ratio", 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := floatValue(tt.params, tt.key)
			if result != tt.expected {
				t.Errorf("floatValue(%v, %q) = %f, expected %f", tt.params, tt.key, result, tt.expected)
			}
		})
	}
}

func TestBoolValue(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]any
		key      string
		expected bool
	}{
		{"nil_map", nil, "enabled", false},
		{"missing_key", map[string]any{}, "enabled", false},
		{"bool_true", map[string]any{"enabled": true}, "enabled", true},
		{"bool_false", map[string]any{"enabled": false}, "enabled", false},
		{"string_true", map[string]any{"enabled": "true"}, "enabled", true},
		{"string_false", map[string]any{"enabled": "false"}, "enabled", false},
		{"string_1", map[string]any{"enabled": "1"}, "enabled", true},
		{"string_0", map[string]any{"enabled": "0"}, "enabled", false},
		{"int_nonzero", map[string]any{"enabled": 1}, "enabled", true},
		{"int_zero", map[string]any{"enabled": 0}, "enabled", false},
		{"float_nonzero", map[string]any{"enabled": 1.0}, "enabled", true},
		{"float_zero", map[string]any{"enabled": 0.0}, "enabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := boolValue(tt.params, tt.key)
			if result != tt.expected {
				t.Errorf("boolValue(%v, %q) = %v, expected %v", tt.params, tt.key, result, tt.expected)
			}
		})
	}
}

// TestJSONRoundTrip ensures coercion works correctly with JSON-decoded values
func TestJSONRoundTrip(t *testing.T) {
	// Simulate a params object coming from JSON decoding
	jsonData := `{"timeoutMs": 5000, "retryAttempts": 3, "ratio": 0.5, "enabled": true}`
	var params map[string]any
	if err := json.Unmarshal([]byte(jsonData), &params); err != nil {
		t.Fatalf("failed to unmarshal test data: %v", err)
	}

	// intValue should work with JSON-decoded float64
	if got := intValue(params, "timeoutMs"); got != 5000 {
		t.Errorf("intValue(params, 'timeoutMs') = %d, expected 5000", got)
	}
	if got := intValue(params, "retryAttempts"); got != 3 {
		t.Errorf("intValue(params, 'retryAttempts') = %d, expected 3", got)
	}

	// floatValue should work
	if got := floatValue(params, "ratio"); got != 0.5 {
		t.Errorf("floatValue(params, 'ratio') = %f, expected 0.5", got)
	}

	// boolValue should work
	if got := boolValue(params, "enabled"); got != true {
		t.Errorf("boolValue(params, 'enabled') = %v, expected true", got)
	}
}
