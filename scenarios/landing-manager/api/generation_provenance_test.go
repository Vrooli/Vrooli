package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"landing-manager/services"
)

// [REQ:TMPL-PROVENANCE]
func TestTMPL_PROVENANCE_ProvenanceStamping(t *testing.T) {
	registry, outputDir := setupGenerationTest(t)
	generator := services.NewScenarioGenerator(registry)

	_, err := generator.GenerateScenario("test-template", "Provenance Test", "prov-test", nil)
	if err != nil {
		t.Fatalf("GenerateScenario() returned error: %v", err)
	}

	// Check template.json exists with provenance
	provenancePath := filepath.Join(outputDir, "prov-test", ".vrooli", "template.json")
	if _, err := os.Stat(provenancePath); os.IsNotExist(err) {
		t.Fatal("Expected template.json to exist for provenance tracking")
	}

	// Read and verify provenance data
	provData, err := os.ReadFile(provenancePath)
	if err != nil {
		t.Fatalf("Failed to read template.json: %v", err)
	}

	var prov map[string]interface{}
	if err := json.Unmarshal(provData, &prov); err != nil {
		t.Fatalf("Failed to unmarshal provenance: %v", err)
	}

	// Verify provenance contains template_id
	if templateID, ok := prov["template_id"].(string); !ok || templateID != "test-template" {
		t.Errorf("Expected template_id 'test-template', got '%v'", prov["template_id"])
	}

	// Verify provenance contains template_version
	if templateVer, ok := prov["template_version"].(string); !ok || templateVer != "1.0.0" {
		t.Errorf("Expected template_version '1.0.0', got '%v'", prov["template_version"])
	}

	// Verify provenance contains generated_at timestamp
	generatedAt, ok := prov["generated_at"].(string)
	if !ok {
		t.Error("Expected provenance to contain 'generated_at' timestamp")
	}

	// Verify timestamp is valid RFC3339 format
	_, err = time.Parse(time.RFC3339, generatedAt)
	if err != nil {
		t.Errorf("generated_at timestamp is not valid RFC3339 format: %v", err)
	}
}
