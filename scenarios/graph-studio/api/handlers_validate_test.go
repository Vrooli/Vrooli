package main

import (
	"database/sql"
	"strings"
	"testing"
)

func TestComputeValidationResult(t *testing.T) {
	api := NewAPI()
	api.plugins["mind-maps"] = &Plugin{ID: "mind-maps", Name: "Mind Maps", Enabled: true}
	api.plugins["bpmn"] = &Plugin{ID: "bpmn", Name: "BPMN", Enabled: true}
	api.validator = NewGraphValidator(api.plugins)

	tests := []struct {
		name        string
		graphType   string
		data        sql.NullString
		wantValid   bool
		wantError   string
		wantWarning string
	}{
		{
			name:      "missing data",
			graphType: "mind-maps",
			data:      sql.NullString{Valid: false},
			wantValid: false,
			wantError: "Graph has no data",
		},
		{
			name:      "empty object",
			graphType: "mind-maps",
			data:      sql.NullString{String: "{}", Valid: true},
			wantValid: false,
			wantError: "Graph data is empty",
		},
		{
			name:      "invalid json",
			graphType: "mind-maps",
			data:      sql.NullString{String: "{invalid", Valid: true},
			wantValid: false,
			wantError: "Invalid JSON format",
		},
		{
			name:      "mind map missing root",
			graphType: "mind-maps",
			data:      sql.NullString{String: `{"nodes":[]}`, Valid: true},
			wantValid: false,
			wantError: "Mind map must have a root node",
		},
		{
			name:      "mind map valid",
			graphType: "mind-maps",
			data:      sql.NullString{String: `{"root":{"id":"root"}}`, Valid: true},
			wantValid: true,
		},
		{
			name:        "bpmn missing events",
			graphType:   "bpmn",
			data:        sql.NullString{String: `{"nodes":[{"id":"Task_1"}]}`, Valid: true},
			wantValid:   true,
			wantWarning: "start event",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := api.computeValidationResult(tt.graphType, tt.data)

			if result.Valid != tt.wantValid {
				t.Fatalf("expected valid=%v, got %v", tt.wantValid, result.Valid)
			}

			if tt.wantError != "" {
				if !containsFragment(result.Errors, tt.wantError) {
					t.Fatalf("expected error containing %q, got %#v", tt.wantError, result.Errors)
				}
			} else if len(result.Errors) > 0 {
				t.Fatalf("expected no errors, got %#v", result.Errors)
			}

			if tt.wantWarning != "" {
				if !containsFragment(result.Warnings, tt.wantWarning) {
					t.Fatalf("expected warning containing %q, got %#v", tt.wantWarning, result.Warnings)
				}
			}
		})
	}
}

func containsFragment(items []string, fragment string) bool {
	for _, item := range items {
		if strings.Contains(item, fragment) {
			return true
		}
	}
	return false
}
