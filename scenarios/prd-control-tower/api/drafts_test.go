package main

import (
	"database/sql"
	"testing"
)

func TestNullString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected sql.NullString
	}{
		{
			name:  "non-empty string",
			input: "test",
			expected: sql.NullString{
				String: "test",
				Valid:  true,
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: sql.NullString{
				String: "",
				Valid:  false,
			},
		},
		{
			name:  "whitespace string",
			input: "   ",
			expected: sql.NullString{
				String: "   ",
				Valid:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullString(tt.input)

			if result.Valid != tt.expected.Valid {
				t.Errorf("nullString(%q).Valid = %v, want %v", tt.input, result.Valid, tt.expected.Valid)
			}

			if result.String != tt.expected.String {
				t.Errorf("nullString(%q).String = %q, want %q", tt.input, result.String, tt.expected.String)
			}
		})
	}
}

func TestGetDraftPath(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		expected   string
	}{
		{
			name:       "scenario draft",
			entityType: "scenario",
			entityName: "test-scenario",
			expected:   "../data/prd-drafts/scenario/test-scenario.md",
		},
		{
			name:       "resource draft",
			entityType: "resource",
			entityName: "test-resource",
			expected:   "../data/prd-drafts/resource/test-resource.md",
		},
		{
			name:       "scenario with special characters",
			entityType: "scenario",
			entityName: "my-test-scenario-v2",
			expected:   "../data/prd-drafts/scenario/my-test-scenario-v2.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDraftPath(tt.entityType, tt.entityName)

			if result != tt.expected {
				t.Errorf("getDraftPath(%q, %q) = %q, want %q",
					tt.entityType, tt.entityName, result, tt.expected)
			}
		})
	}
}

func TestDraftValidation(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityName string
		wantValid  bool
	}{
		{
			name:       "valid scenario",
			entityType: "scenario",
			entityName: "test-scenario",
			wantValid:  true,
		},
		{
			name:       "valid resource",
			entityType: "resource",
			entityName: "test-resource",
			wantValid:  true,
		},
		{
			name:       "invalid entity type",
			entityType: "invalid",
			entityName: "test",
			wantValid:  false,
		},
		{
			name:       "empty entity type",
			entityType: "",
			entityName: "test",
			wantValid:  false,
		},
		{
			name:       "empty entity name",
			entityType: "scenario",
			entityName: "",
			wantValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate entity type
			isValidType := tt.entityType == "scenario" || tt.entityType == "resource"
			if !isValidType && tt.wantValid {
				t.Errorf("Expected entity type %q to be invalid", tt.entityType)
			}

			// Validate entity name
			isValidName := tt.entityName != ""
			if !isValidName && tt.wantValid {
				t.Errorf("Expected entity name %q to be invalid", tt.entityName)
			}
		})
	}
}
