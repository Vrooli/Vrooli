package typeconv

import (
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"byte slice", []byte("world"), "world"},
		{"int", 42, ""},
		{"nil", nil, ""},
		{"float", 3.14, ""},
		{"bool", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("ToString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", 42, 42},
		{"int32", int32(100), 100},
		{"int64", int64(200), 200},
		{"float64", 3.14, 3},
		{"json.Number int", json.Number("42"), 42},
		{"string int", "123", 123},
		{"string invalid", "abc", 0},
		{"nil", nil, 0},
		{"empty string", "", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ToInt(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string false", "false", false},
		{"string 1", "1", true},
		{"string 0", "0", false},
		{"string invalid", "abc", false},
		{"int", 1, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToBool(tt.input)
			if result != tt.expected {
				t.Errorf("ToBool(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float64", 3.14, 3.14},
		{"float32", float32(2.5), 2.5},
		{"int", 42, 42.0},
		{"int32", int32(100), 100.0},
		{"int64", int64(200), 200.0},
		{"json.Number float", json.Number("3.14"), 3.14},
		{"string float", "2.5", 2.5},
		{"string invalid", "abc", 0.0},
		{"nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToFloat(tt.input)
			if result != tt.expected {
				t.Errorf("ToFloat(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTimePtr(t *testing.T) {
	now := time.Now()
	rfc3339Time := "2024-01-15T10:30:00Z"
	rfc3339NanoTime := "2024-01-15T10:30:00.123456789Z"

	tests := []struct {
		name      string
		input     any
		expectNil bool
		validate  func(*time.Time) bool
	}{
		{
			name:      "time.Time",
			input:     now,
			expectNil: false,
			validate:  func(t *time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "pointer to time.Time",
			input:     &now,
			expectNil: false,
			validate:  func(t *time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "RFC3339 string",
			input:     rfc3339Time,
			expectNil: false,
			validate:  func(t *time.Time) bool { return !t.IsZero() },
		},
		{
			name:      "RFC3339Nano string",
			input:     rfc3339NanoTime,
			expectNil: false,
			validate:  func(t *time.Time) bool { return !t.IsZero() },
		},
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
		{
			name:      "invalid string",
			input:     "not a time",
			expectNil: true,
		},
		{
			name:      "nil",
			input:     nil,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTimePtr(tt.input)
			if tt.expectNil {
				if result != nil {
					t.Errorf("ToTimePtr(%v) should return nil, got %v", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("ToTimePtr(%v) should not return nil", tt.input)
				} else if tt.validate != nil && !tt.validate(result) {
					t.Errorf("ToTimePtr(%v) validation failed", tt.input)
				}
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{"[]string", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"[]any with strings", []any{"x", "y", "z"}, []string{"x", "y", "z"}},
		{"[]any with mixed", []any{"a", 42, "b"}, []string{"a", "b"}},
		{"single string", "hello", []string{"hello"}},
		{"empty string", "", []string{}},
		{"nil", nil, []string{}},
		{"int", 42, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToStringSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ToStringSlice(%v) length = %d, expected %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ToStringSlice(%v)[%d] = %q, expected %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestToInterfaceSlice(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expectedLen int
	}{
		{"nil", nil, 0},
		{"[]any", []any{"a", 1, true}, 3},
		{"[]map[string]any", []map[string]any{{"a": 1}, {"b": 2}}, 2},
		{"map[string]any", map[string]any{"a": 1, "b": 2}, 2},
		// Note: JSON string parsing only works for types that json.Marshal returns something parseable as array
		// A plain string is not a valid JSON array
		{"string (not array)", "hello", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToInterfaceSlice(tt.input)
			if len(result) != tt.expectedLen {
				t.Errorf("ToInterfaceSlice(%v) length = %d, expected %d", tt.input, len(result), tt.expectedLen)
			}
		})
	}
}

func TestDeepCloneMap(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := DeepCloneMap(nil)
		if result != nil {
			t.Errorf("DeepCloneMap(nil) should return nil")
		}
	})

	t.Run("simple map", func(t *testing.T) {
		input := map[string]any{"a": 1, "b": "hello"}
		result := DeepCloneMap(input)

		// Verify values are the same
		if result["a"] != 1 || result["b"] != "hello" {
			t.Errorf("DeepCloneMap values don't match")
		}

		// Verify it's a different map
		input["c"] = "modified"
		if _, exists := result["c"]; exists {
			t.Errorf("DeepCloneMap should create independent copy")
		}
	})

	t.Run("nested map", func(t *testing.T) {
		input := map[string]any{
			"outer": map[string]any{"inner": "value"},
		}
		result := DeepCloneMap(input)

		// Modify the input nested map
		inner := input["outer"].(map[string]any)
		inner["inner"] = "modified"

		// Verify the clone is not affected
		resultInner := result["outer"].(map[string]any)
		if resultInner["inner"] != "value" {
			t.Errorf("DeepCloneMap should deep copy nested maps")
		}
	})

	t.Run("nested slice", func(t *testing.T) {
		input := map[string]any{
			"items": []any{"a", "b", "c"},
		}
		result := DeepCloneMap(input)

		// Modify the input slice
		inputItems := input["items"].([]any)
		inputItems[0] = "modified"

		// Verify the clone is not affected
		resultItems := result["items"].([]any)
		if resultItems[0] != "a" {
			t.Errorf("DeepCloneMap should deep copy slices")
		}
	})
}

func TestDeepCloneValue(t *testing.T) {
	t.Run("primitive int", func(t *testing.T) {
		result := DeepCloneValue(42)
		if result != 42 {
			t.Errorf("DeepCloneValue should preserve int")
		}
	})

	t.Run("primitive string", func(t *testing.T) {
		result := DeepCloneValue("hello")
		if result != "hello" {
			t.Errorf("DeepCloneValue should preserve string")
		}
	})

	t.Run("string slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := DeepCloneValue(input).([]string)

		input[0] = "modified"
		if result[0] != "a" {
			t.Errorf("DeepCloneValue should copy string slices")
		}
	})

	t.Run("any slice with maps", func(t *testing.T) {
		input := []any{
			map[string]any{"key": "value"},
		}
		result := DeepCloneValue(input).([]any)

		// Modify the input
		inputMap := input[0].(map[string]any)
		inputMap["key"] = "modified"

		// Verify the clone is not affected
		resultMap := result[0].(map[string]any)
		if resultMap["key"] != "value" {
			t.Errorf("DeepCloneValue should deep copy slices with maps")
		}
	})
}

func TestAnyToJsonValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		wantKind string // describes the expected Kind type
	}{
		{"nil", nil, "NullValue"},
		{"bool true", true, "BoolValue"},
		{"bool false", false, "BoolValue"},
		{"int", 42, "IntValue"},
		{"int32", int32(100), "IntValue"},
		{"int64", int64(200), "IntValue"},
		{"uint", uint(300), "IntValue"},
		{"uint32", uint32(400), "IntValue"},
		{"uint64", uint64(500), "IntValue"},
		{"float32", float32(3.14), "DoubleValue"},
		{"float64", 3.14159, "DoubleValue"},
		{"string", "hello", "StringValue"},
		{"bytes", []byte("world"), "BytesValue"},
		{"map", map[string]any{"key": "value"}, "ObjectValue"},
		{"slice", []any{1, "two", 3.0}, "ListValue"},
		{"struct fallback", struct{ Name string }{"test"}, "ObjectValue"}, // JSON round-trip converts to object
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnyToJsonValue(tt.input)
			if result == nil {
				t.Fatal("AnyToJsonValue returned nil")
			}

			// Check the kind matches
			switch tt.wantKind {
			case "NullValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_NullValue); !ok {
					t.Errorf("expected NullValue, got %T", result.Kind)
				}
			case "BoolValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_BoolValue); !ok {
					t.Errorf("expected BoolValue, got %T", result.Kind)
				}
			case "IntValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_IntValue); !ok {
					t.Errorf("expected IntValue, got %T", result.Kind)
				}
			case "DoubleValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_DoubleValue); !ok {
					t.Errorf("expected DoubleValue, got %T", result.Kind)
				}
			case "StringValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_StringValue); !ok {
					t.Errorf("expected StringValue, got %T", result.Kind)
				}
			case "BytesValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_BytesValue); !ok {
					t.Errorf("expected BytesValue, got %T", result.Kind)
				}
			case "ObjectValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_ObjectValue); !ok {
					t.Errorf("expected ObjectValue, got %T", result.Kind)
				}
			case "ListValue":
				if _, ok := result.Kind.(*commonv1.JsonValue_ListValue); !ok {
					t.Errorf("expected ListValue, got %T", result.Kind)
				}
			}
		})
	}

	// Test nested map/slice
	t.Run("nested structures", func(t *testing.T) {
		input := map[string]any{
			"items": []any{1, 2, 3},
			"meta":  map[string]any{"count": 3},
		}
		result := AnyToJsonValue(input)
		if result == nil {
			t.Fatal("AnyToJsonValue returned nil for nested structure")
		}
		obj, ok := result.Kind.(*commonv1.JsonValue_ObjectValue)
		if !ok {
			t.Fatalf("expected ObjectValue, got %T", result.Kind)
		}
		if len(obj.ObjectValue.Fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(obj.ObjectValue.Fields))
		}
	})
}

func TestJsonValueToAny(t *testing.T) {
	t.Run("nil input", func(t *testing.T) {
		result := JsonValueToAny(nil)
		if result != nil {
			t.Errorf("expected nil for nil input, got %v", result)
		}
	})

	t.Run("bool value", func(t *testing.T) {
		input := AnyToJsonValue(true)
		result := JsonValueToAny(input)
		if result != true {
			t.Errorf("expected true, got %v", result)
		}
	})

	t.Run("int value", func(t *testing.T) {
		input := AnyToJsonValue(42)
		result := JsonValueToAny(input)
		if result != int64(42) {
			t.Errorf("expected 42, got %v", result)
		}
	})

	t.Run("string value", func(t *testing.T) {
		input := AnyToJsonValue("hello")
		result := JsonValueToAny(input)
		if result != "hello" {
			t.Errorf("expected 'hello', got %v", result)
		}
	})

	t.Run("null value", func(t *testing.T) {
		input := AnyToJsonValue(nil)
		result := JsonValueToAny(input)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("roundtrip map", func(t *testing.T) {
		original := map[string]any{"key": "value", "num": int64(42)}
		jsonVal := AnyToJsonValue(original)
		result := JsonValueToAny(jsonVal)

		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result)
		}
		if resultMap["key"] != "value" {
			t.Errorf("expected key='value', got %v", resultMap["key"])
		}
	})

	t.Run("roundtrip slice", func(t *testing.T) {
		original := []any{"a", int64(1), true}
		jsonVal := AnyToJsonValue(original)
		result := JsonValueToAny(jsonVal)

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", result)
		}
		if len(resultSlice) != 3 {
			t.Errorf("expected length 3, got %d", len(resultSlice))
		}
	})
}

func TestToInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int32
		ok       bool
	}{
		{"int", 42, 42, true},
		{"int32", int32(100), 100, true},
		{"int64", int64(200), 200, true},
		{"float64", 3.9, 3, true},
		{"float32", float32(4.5), 4, true},
		{"json.Number", json.Number("123"), 123, true},
		{"json.Number invalid", json.Number("abc"), 0, false},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := ToInt32(tt.input)
			if ok != tt.ok {
				t.Errorf("ToInt32(%v) ok = %v, expected %v", tt.input, ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("ToInt32(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFloat64Func(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		ok       bool
	}{
		{"float64", 3.14, 3.14, true},
		{"float32", float32(2.5), 2.5, true},
		{"int", 42, 42.0, true},
		{"int32", int32(100), 100.0, true},
		{"int64", int64(200), 200.0, true},
		{"json.Number", json.Number("3.14"), 3.14, true},
		{"json.Number invalid", json.Number("abc"), 0, false},
		{"string", "3.14", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := ToFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("ToFloat64(%v) ok = %v, expected %v", tt.input, ok, tt.ok)
			}
			if result != tt.expected {
				t.Errorf("ToFloat64(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

// ActionType tests moved to internal/protoconv/enum_convert_test.go
