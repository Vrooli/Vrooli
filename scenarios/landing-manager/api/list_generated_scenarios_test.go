package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"landing-manager/services"
)

// [REQ:TMPL-LIFECYCLE] Comprehensive test suite for ListGeneratedScenarios
// This file focuses on increasing coverage for ListGeneratedScenarios (target: >90%)
// Current coverage: 70.6% - need to test all parsing paths and edge cases

func TestListGeneratedScenarios_WithProvenanceAndServiceMetadata(t *testing.T) {
	// Test the full happy path with both template.json and service.json (lines 667-692)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	// Create generated scenarios directory
	scenarioPath := filepath.Join(genDir, "test-scenario-full")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create template.json with provenance data
	provenance := map[string]interface{}{
		"template_id":      "test-template",
		"template_version": "1.0.0",
		"generated_at":     "2025-11-26T12:00:00Z",
	}
	provData, _ := json.Marshal(provenance)
	if err := os.WriteFile(filepath.Join(vrooliDir, "template.json"), provData, 0644); err != nil {
		t.Fatalf("Failed to write template.json: %v", err)
	}

	// Create service.json with displayName
	service := map[string]interface{}{
		"service": map[string]interface{}{
			"displayName": "Test Scenario Full",
			"description": "A test scenario",
		},
	}
	serviceData, _ := json.Marshal(service)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), serviceData, 0644); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	gs := scenarios[0]

	// Verify all fields are populated correctly
	if gs.ScenarioID != "test-scenario-full" {
		t.Errorf("Expected ScenarioID 'test-scenario-full', got '%s'", gs.ScenarioID)
	}

	if gs.Name != "Test Scenario Full" {
		t.Errorf("Expected Name 'Test Scenario Full' (from service.json), got '%s'", gs.Name)
	}

	if gs.TemplateID != "test-template" {
		t.Errorf("Expected TemplateID 'test-template', got '%s'", gs.TemplateID)
	}

	if gs.TemplateVersion != "1.0.0" {
		t.Errorf("Expected TemplateVersion '1.0.0', got '%s'", gs.TemplateVersion)
	}

	if gs.GeneratedAt != "2025-11-26T12:00:00Z" {
		t.Errorf("Expected GeneratedAt '2025-11-26T12:00:00Z', got '%s'", gs.GeneratedAt)
	}

	if gs.Status != "present" {
		t.Errorf("Expected Status 'present', got '%s'", gs.Status)
	}

	if gs.Path != scenarioPath {
		t.Errorf("Expected Path '%s', got '%s'", scenarioPath, gs.Path)
	}
}

func TestListGeneratedScenarios_WithProvenanceOnly(t *testing.T) {
	// Test scenario with template.json but no service.json (lines 667-680, name fallback)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-scenario-prov-only")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create only template.json
	provenance := map[string]interface{}{
		"template_id":      "prov-template",
		"template_version": "2.0.0",
		"generated_at":     "2025-11-26T13:00:00Z",
	}
	provData, _ := json.Marshal(provenance)
	if err := os.WriteFile(filepath.Join(vrooliDir, "template.json"), provData, 0644); err != nil {
		t.Fatalf("Failed to write template.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	gs := scenarios[0]

	// Name should fall back to scenario slug (line 661)
	if gs.Name != "test-scenario-prov-only" {
		t.Errorf("Expected Name to fallback to slug 'test-scenario-prov-only', got '%s'", gs.Name)
	}

	// Provenance fields should be populated
	if gs.TemplateID != "prov-template" {
		t.Errorf("Expected TemplateID 'prov-template', got '%s'", gs.TemplateID)
	}

	if gs.TemplateVersion != "2.0.0" {
		t.Errorf("Expected TemplateVersion '2.0.0', got '%s'", gs.TemplateVersion)
	}
}

func TestListGeneratedScenarios_WithServiceMetadataOnly(t *testing.T) {
	// Test scenario with service.json but no template.json (lines 683-692)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-scenario-svc-only")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create only service.json
	service := map[string]interface{}{
		"service": map[string]interface{}{
			"displayName": "Service Metadata Only",
		},
	}
	serviceData, _ := json.Marshal(service)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), serviceData, 0644); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	gs := scenarios[0]

	// Name should come from service.json
	if gs.Name != "Service Metadata Only" {
		t.Errorf("Expected Name 'Service Metadata Only', got '%s'", gs.Name)
	}

	// Provenance fields should be empty
	if gs.TemplateID != "" {
		t.Errorf("Expected empty TemplateID, got '%s'", gs.TemplateID)
	}

	if gs.TemplateVersion != "" {
		t.Errorf("Expected empty TemplateVersion, got '%s'", gs.TemplateVersion)
	}

	if gs.GeneratedAt != "" {
		t.Errorf("Expected empty GeneratedAt, got '%s'", gs.GeneratedAt)
	}
}

func TestListGeneratedScenarios_MinimalScenario(t *testing.T) {
	// Test scenario with no metadata files (lines 659-664 only)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-scenario-minimal")
	if err := os.MkdirAll(scenarioPath, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	gs := scenarios[0]

	// All fields should have default/slug values
	if gs.ScenarioID != "test-scenario-minimal" {
		t.Errorf("Expected ScenarioID 'test-scenario-minimal', got '%s'", gs.ScenarioID)
	}

	if gs.Name != "test-scenario-minimal" {
		t.Errorf("Expected Name to default to slug, got '%s'", gs.Name)
	}

	if gs.Status != "present" {
		t.Errorf("Expected Status 'present', got '%s'", gs.Status)
	}

	if gs.TemplateID != "" || gs.TemplateVersion != "" || gs.GeneratedAt != "" {
		t.Error("Expected all provenance fields to be empty")
	}
}

func TestListGeneratedScenarios_EmptyDisplayName(t *testing.T) {
	// Test that empty displayName is ignored and slug is used (line 687 check)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-empty-display")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create service.json with empty displayName
	service := map[string]interface{}{
		"service": map[string]interface{}{
			"displayName": "",
			"description": "Test",
		},
	}
	serviceData, _ := json.Marshal(service)
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), serviceData, 0644); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	// Empty displayName should not override slug name (line 687: && display != "")
	if scenarios[0].Name != "test-empty-display" {
		t.Errorf("Expected Name to remain as slug when displayName is empty, got '%s'", scenarios[0].Name)
	}
}

func TestListGeneratedScenarios_MalformedJSON(t *testing.T) {
	// Test that malformed JSON is gracefully ignored (lines 669, 685 error paths)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-malformed-json")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Write malformed JSON to both files
	if err := os.WriteFile(filepath.Join(vrooliDir, "template.json"), []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write template.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte("{invalid}"), 0644); err != nil {
		t.Fatalf("Failed to write service.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error (malformed JSON should be ignored), got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	// Should fall back to defaults when JSON is malformed
	gs := scenarios[0]
	if gs.Name != "test-malformed-json" {
		t.Errorf("Expected Name to fallback to slug, got '%s'", gs.Name)
	}

	if gs.TemplateID != "" || gs.TemplateVersion != "" || gs.GeneratedAt != "" {
		t.Error("Expected provenance fields to be empty when template.json is malformed")
	}
}

func TestListGeneratedScenarios_SkipsNonDirectories(t *testing.T) {
	// Test that non-directory entries are skipped (lines 653-655)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	if err := os.MkdirAll(genDir, 0755); err != nil {
		t.Fatalf("Failed to create generated directory: %v", err)
	}

	// Create a file (not a directory)
	filePath := filepath.Join(genDir, "not-a-scenario.txt")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a valid scenario directory
	scenarioPath := filepath.Join(genDir, "valid-scenario")
	if err := os.MkdirAll(scenarioPath, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only return the directory, not the file
	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario (file should be skipped), got %d", len(scenarios))
	}

	if scenarios[0].ScenarioID != "valid-scenario" {
		t.Errorf("Expected ScenarioID 'valid-scenario', got '%s'", scenarios[0].ScenarioID)
	}
}

func TestListGeneratedScenarios_MultipleScenariosSorted(t *testing.T) {
	// Test that multiple scenarios are returned in sorted order (lines 697-699)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	// Create scenarios in non-alphabetical order
	for _, name := range []string{"charlie", "alpha", "bravo", "delta"} {
		scenarioPath := filepath.Join(genDir, name)
		if err := os.MkdirAll(scenarioPath, 0755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 4 {
		t.Fatalf("Expected 4 scenarios, got %d", len(scenarios))
	}

	// Verify sorted order
	expected := []string{"alpha", "bravo", "charlie", "delta"}
	for i, exp := range expected {
		if scenarios[i].ScenarioID != exp {
			t.Errorf("Expected scenario[%d] to be '%s', got '%s'", i, exp, scenarios[i].ScenarioID)
		}
	}
}

func TestListGeneratedScenarios_PartialProvenance(t *testing.T) {
	// Test scenarios with partial provenance data (only some fields)
	tmpRoot := t.TempDir()
	genDir := filepath.Join(tmpRoot, "generated")
	os.Setenv("GEN_OUTPUT_DIR", genDir)
	defer os.Unsetenv("GEN_OUTPUT_DIR")

	scenarioPath := filepath.Join(genDir, "test-partial-prov")
	vrooliDir := filepath.Join(scenarioPath, ".vrooli")
	if err := os.MkdirAll(vrooliDir, 0755); err != nil {
		t.Fatalf("Failed to create scenario directory: %v", err)
	}

	// Create template.json with only template_id (missing version and timestamp)
	provenance := map[string]interface{}{
		"template_id": "partial-template",
		// Intentionally missing template_version and generated_at
	}
	provData, _ := json.Marshal(provenance)
	if err := os.WriteFile(filepath.Join(vrooliDir, "template.json"), provData, 0644); err != nil {
		t.Fatalf("Failed to write template.json: %v", err)
	}

	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)

	scenarios, err := generator.ListGeneratedScenarios()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(scenarios) != 1 {
		t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
	}

	gs := scenarios[0]

	// Only template_id should be populated (lines 670-678)
	if gs.TemplateID != "partial-template" {
		t.Errorf("Expected TemplateID 'partial-template', got '%s'", gs.TemplateID)
	}

	if gs.TemplateVersion != "" {
		t.Errorf("Expected empty TemplateVersion, got '%s'", gs.TemplateVersion)
	}

	if gs.GeneratedAt != "" {
		t.Errorf("Expected empty GeneratedAt, got '%s'", gs.GeneratedAt)
	}
}
