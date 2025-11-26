package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TMPL-OUTPUT-VALIDATION]
func TestTMPL_OUTPUT_VALIDATION_GeneratedStructure(t *testing.T) {
	ts, outputDir := setupGenerationTest(t)

	result, err := ts.GenerateScenario("test-template", "Test App", "test-app", nil)
	if err != nil {
		t.Fatalf("GenerateScenario() returned error: %v", err)
	}

	// Verify expected directories exist
	genPath := filepath.Join(outputDir, "test-app")
	requiredDirs := []string{
		filepath.Join(genPath, "api"),
		filepath.Join(genPath, "ui"),
		filepath.Join(genPath, ".vrooli"),
		filepath.Join(genPath, "requirements"),
	}

	for _, dir := range requiredDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s to exist after generation", dir)
		}
	}

	// Verify service.json was rewritten
	svcPath := filepath.Join(genPath, ".vrooli", "service.json")
	if _, err := os.Stat(svcPath); os.IsNotExist(err) {
		t.Error("Expected service.json to exist after generation")
	}

	// Verify result contains path
	if path, ok := result["path"].(string); !ok || path == "" {
		t.Error("Expected result to contain non-empty 'path' field")
	}

	// Verify service.json is valid JSON
	svcData, err := os.ReadFile(svcPath)
	if err != nil {
		t.Fatalf("Failed to read generated service.json: %v", err)
	}

	var svc map[string]interface{}
	if err := json.Unmarshal(svcData, &svc); err != nil {
		t.Fatalf("Generated service.json is not valid JSON: %v", err)
	}

	// Verify service section exists
	if _, ok := svc["service"]; !ok {
		t.Error("Generated service.json missing 'service' section")
	}

	// Verify lifecycle section exists
	if _, ok := svc["lifecycle"]; !ok {
		t.Error("Generated service.json missing 'lifecycle' section")
	}
}
