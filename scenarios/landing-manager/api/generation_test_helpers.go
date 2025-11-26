package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// createMinimalPayload creates a minimal template payload for testing
func createMinimalPayload(t *testing.T) string {
	t.Helper()
	payload := t.TempDir()

	minimalFiles := map[string]string{
		filepath.Join(payload, "api", "main.go"):                          "package main",
		filepath.Join(payload, "ui", "src", "App.tsx"):                    "// React app",
		filepath.Join(payload, "requirements", "index.json"):              "{}",
		filepath.Join(payload, "initialization", "configuration", "config.env"): "",
		filepath.Join(payload, "Makefile"):                                "all:\n\techo test",
		filepath.Join(payload, "PRD.md"):                                  "# PRD",
		filepath.Join(payload, ".vrooli", "service.json"):                 `{"service": {"name": "stub", "displayName": "Stub", "description": "stub", "repository": {"directory": "/scenarios/stub"}}, "lifecycle": {"develop": {"steps": [{"name": "start-api", "run": ""}]}}}`,
	}

	for p, content := range minimalFiles {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("failed to create dir for %s: %v", p, err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", p, err)
		}
	}

	return payload
}

// createTestTemplate creates a test template and returns the service
func createTestTemplate(t *testing.T, id, name, version string) (*TemplateService, string) {
	t.Helper()
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, id+".json")

	testTemplate := Template{
		ID:          id,
		Name:        name,
		Description: "Test template for " + name,
		Version:     version,
	}

	data, err := json.Marshal(testTemplate)
	if err != nil {
		t.Fatalf("Failed to marshal test template: %v", err)
	}

	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	return &TemplateService{templatesDir: tmpDir}, tmpDir
}

// setupGenerationTest sets up a test environment with template, payload, and output dir
func setupGenerationTest(t *testing.T) (*TemplateService, string) {
	t.Helper()
	ts, _ := createTestTemplate(t, "test-template", "Test Template", "1.0.0")
	payload := createMinimalPayload(t)
	outputDir := t.TempDir()

	t.Setenv("TEMPLATE_PAYLOAD_DIR", payload)
	t.Setenv("GEN_OUTPUT_DIR", outputDir)

	return ts, outputDir
}
