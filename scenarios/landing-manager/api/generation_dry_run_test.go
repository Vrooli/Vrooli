package main

import (
	"os"
	"path/filepath"
	"testing"

	"landing-manager/services"
)

// [REQ:TMPL-DRY-RUN]
func TestTMPL_DRY_RUN_DryRunMode(t *testing.T) {
	registry, outputDir := setupGenerationTest(t)
	generator := services.NewScenarioGenerator(registry)

	opts := map[string]interface{}{"dry_run": true}
	result, err := generator.GenerateScenario("test-template", "Test Landing", "test-landing", opts)
	if err != nil {
		t.Errorf("GenerateScenario() in dry-run mode returned error: %v", err)
	}
	if result == nil {
		t.Fatal("GenerateScenario() returned nil result")
	}

	// Verify status is dry_run
	status, ok := result["status"].(string)
	if !ok || status != "dry_run" {
		t.Errorf("Expected status 'dry_run', got '%v'", result["status"])
	}

	// Check that plan was generated
	plan, ok := result["plan"].(map[string]interface{})
	if !ok || plan == nil {
		t.Fatal("Expected plan to be generated in dry-run mode")
	}

	// Verify plan contains paths
	paths, ok := plan["paths"].([]string)
	if !ok {
		t.Fatalf("Expected paths to be []string, got %T", plan["paths"])
	}

	if len(paths) == 0 {
		t.Error("Expected plan to contain file paths")
	}

	// Verify no actual files were written
	genPath := filepath.Join(outputDir, "test-landing")
	if _, err := os.Stat(genPath); !os.IsNotExist(err) {
		t.Error("Expected dry-run to NOT create output directory, but it exists")
	}
}

// [REQ:TMPL-DRY-RUN]
func TestTMPL_DRY_RUN_ValidationStillApplies(t *testing.T) {
	registry, _ := setupGenerationTest(t)
	generator := services.NewScenarioGenerator(registry)

	opts := map[string]interface{}{"dry_run": true}

	tests := []struct {
		name        string
		templateID  string
		displayName string
		slug        string
	}{
		{
			name:        "missing name",
			templateID:  "test-template",
			displayName: "",
			slug:        "test-slug",
		},
		{
			name:        "missing slug",
			templateID:  "test-template",
			displayName: "Test Name",
			slug:        "",
		},
		{
			name:        "non-existent template",
			templateID:  "non-existent-template",
			displayName: "Test Name",
			slug:        "test-slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := generator.GenerateScenario(tt.templateID, tt.displayName, tt.slug, opts)
			if err == nil {
				t.Errorf("Expected error for %s in dry-run mode", tt.name)
			}
		})
	}
}
