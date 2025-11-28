package main

import (
	"reflect"
	"testing"
)

// [REQ:PCT-REQ-INTEGRATION] Template flag formatting for scenario creation
func TestFormatTemplateFlag(t *testing.T) {
	tests := []struct {
		name     string
		def      templateVarDefinition
		value    string
		expected []string
	}{
		{
			name: "explicit flag name",
			def: templateVarDefinition{
				Name: "DatabaseURL",
				Flag: "db-url",
			},
			value:    "postgres://localhost/testdb",
			expected: []string{"--db-url", "postgres://localhost/testdb"},
		},
		{
			name: "generated flag from name",
			def: templateVarDefinition{
				Name: "ServiceName",
				Flag: "",
			},
			value:    "my-service",
			expected: []string{"--servicename", "my-service"},
		},
		{
			name: "flag with empty value",
			def: templateVarDefinition{
				Name: "OptionalField",
				Flag: "optional",
			},
			value:    "",
			expected: []string{"--optional", ""},
		},
		{
			name: "flag with special characters in value",
			def: templateVarDefinition{
				Name: "Description",
				Flag: "desc",
			},
			value:    "Test scenario with \"quotes\" and spaces",
			expected: []string{"--desc", "Test scenario with \"quotes\" and spaces"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTemplateFlag(tt.def, tt.value)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("formatTemplateFlag() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// [REQ:PCT-REQ-INTEGRATION] Sort template keys for deterministic ordering
func TestSortedTemplateKeys(t *testing.T) {
	tests := []struct {
		name     string
		vars     map[string]templateVarDefinition
		expected []string
	}{
		{
			name: "single variable",
			vars: map[string]templateVarDefinition{
				"name": {Name: "ServiceName"},
			},
			expected: []string{"name"},
		},
		{
			name: "multiple variables sorted alphabetically",
			vars: map[string]templateVarDefinition{
				"database": {Name: "DatabaseURL"},
				"auth":     {Name: "AuthEnabled"},
				"cache":    {Name: "CacheURL"},
			},
			expected: []string{"auth", "cache", "database"},
		},
		{
			name:     "empty map",
			vars:     map[string]templateVarDefinition{},
			expected: []string{},
		},
		{
			name: "numeric keys sorted alphabetically (as strings)",
			vars: map[string]templateVarDefinition{
				"10": {Name: "Ten"},
				"2":  {Name: "Two"},
				"1":  {Name: "One"},
			},
			expected: []string{"1", "10", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sortedTemplateKeys(tt.vars)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("sortedTemplateKeys() = %v, want %v", result, tt.expected)
			}
		})
	}
}
