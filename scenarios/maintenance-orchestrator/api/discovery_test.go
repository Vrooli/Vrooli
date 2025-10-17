package main

import (
	"testing"
)

func TestGetStringField(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]interface{}
		field        string
		defaultValue string
		expected     string
	}{
		{
			name:         "ValidString",
			data:         map[string]interface{}{"key": "value"},
			field:        "key",
			defaultValue: "default",
			expected:     "value",
		},
		{
			name:         "MissingField",
			data:         map[string]interface{}{},
			field:        "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "NonStringValue",
			data:         map[string]interface{}{"key": 123},
			field:        "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "NilValue",
			data:         map[string]interface{}{"key": nil},
			field:        "key",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "EmptyDefault",
			data:         map[string]interface{}{},
			field:        "key",
			defaultValue: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringField(tt.data, tt.field, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
