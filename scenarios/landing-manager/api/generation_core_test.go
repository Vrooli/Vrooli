package main

import (
	"testing"
)

// [REQ:TMPL-GENERATION]
func TestTMPL_GENERATION_BasicGeneration(t *testing.T) {
	ts, _ := setupGenerationTest(t)

	result, err := ts.GenerateScenario("test-template", "My Landing Page", "my-landing", nil)
	if err != nil {
		t.Errorf("GenerateScenario() returned error: %v", err)
	}
	if result == nil {
		t.Fatal("GenerateScenario() returned nil result")
	}

	scenarioID, ok := result["scenario_id"].(string)
	if !ok || scenarioID != "my-landing" {
		t.Errorf("Expected scenario_id 'my-landing', got '%v'", result["scenario_id"])
	}

	status, ok := result["status"].(string)
	if !ok || status != "created" {
		t.Errorf("Expected status 'created', got '%v'", result["status"])
	}

	// Verify template is included in result
	templateID, ok := result["template"].(string)
	if !ok || templateID != "test-template" {
		t.Errorf("Expected template 'test-template', got '%v'", result["template"])
	}
}

// [REQ:TMPL-GENERATION]
func TestTMPL_GENERATION_ErrorHandling(t *testing.T) {
	ts, _ := setupGenerationTest(t)

	tests := []struct {
		name        string
		templateID  string
		displayName string
		slug        string
		wantErr     bool
	}{
		{
			name:        "missing name",
			templateID:  "test-template",
			displayName: "",
			slug:        "my-landing",
			wantErr:     true,
		},
		{
			name:        "missing slug",
			templateID:  "test-template",
			displayName: "My Landing Page",
			slug:        "",
			wantErr:     true,
		},
		{
			name:        "non-existing template",
			templateID:  "non-existing",
			displayName: "My Landing Page",
			slug:        "my-landing",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ts.GenerateScenario(tt.templateID, tt.displayName, tt.slug, nil)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
