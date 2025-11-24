package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateService_ListTemplates(t *testing.T) {
	// Create temporary templates directory
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test-template.json")

	testTemplate := Template{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Version:     "1.0.0",
		Metadata: map[string]interface{}{
			"author": "Test Author",
		},
	}

	data, err := json.Marshal(testTemplate)
	if err != nil {
		t.Fatalf("Failed to marshal test template: %v", err)
	}

	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	ts := &TemplateService{templatesDir: tmpDir}
	templates, err := ts.ListTemplates()

	if err != nil {
		t.Errorf("ListTemplates() returned error: %v", err)
	}

	if len(templates) != 1 {
		t.Errorf("Expected 1 template, got %d", len(templates))
	}

	if len(templates) > 0 && templates[0].ID != "test-template" {
		t.Errorf("Expected template ID 'test-template', got '%s'", templates[0].ID)
	}
}

func TestTemplateService_GetTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test-template.json")

	testTemplate := Template{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Version:     "1.0.0",
	}

	data, err := json.Marshal(testTemplate)
	if err != nil {
		t.Fatalf("Failed to marshal test template: %v", err)
	}

	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	ts := &TemplateService{templatesDir: tmpDir}

	t.Run("existing template", func(t *testing.T) {
		template, err := ts.GetTemplate("test-template")
		if err != nil {
			t.Errorf("GetTemplate() returned error: %v", err)
		}
		if template == nil {
			t.Fatal("GetTemplate() returned nil template")
		}
		if template.ID != "test-template" {
			t.Errorf("Expected template ID 'test-template', got '%s'", template.ID)
		}
	})

	t.Run("non-existing template", func(t *testing.T) {
		template, err := ts.GetTemplate("non-existing")
		if err == nil {
			t.Error("GetTemplate() should return error for non-existing template")
		}
		if template != nil {
			t.Error("GetTemplate() should return nil template for non-existing ID")
		}
	})
}

func TestTemplateService_GenerateScenario(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "test-template.json")

	testTemplate := Template{
		ID:          "test-template",
		Name:        "Test Template",
		Description: "A test template",
		Version:     "1.0.0",
	}

	data, err := json.Marshal(testTemplate)
	if err != nil {
		t.Fatalf("Failed to marshal test template: %v", err)
	}

	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	ts := &TemplateService{templatesDir: tmpDir}

	t.Run("valid generation", func(t *testing.T) {
		// Provide a minimal payload so scaffolding succeeds without touching the real repo
		payload := t.TempDir()
		minimalFiles := []string{
			filepath.Join(payload, "api", "main.go"),
			filepath.Join(payload, "ui", "src", "App.tsx"),
			filepath.Join(payload, "requirements", "index.json"),
			filepath.Join(payload, "initialization", "configuration", "landing-manager.env"),
			filepath.Join(payload, "Makefile"),
			filepath.Join(payload, "PRD.md"),
			filepath.Join(payload, ".vrooli", "service.json"),
		}
		for _, p := range minimalFiles {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatalf("failed to create payload dir %s: %v", p, err)
			}
			// lightweight file contents to satisfy rewriteServiceConfig
			if strings.HasSuffix(p, "service.json") {
				content := `{
  "service": {"name": "stub", "displayName": "Stub", "description": "stub", "repository": {"directory": "/scenarios/stub"}},
  "lifecycle": {"develop": {"steps": [{"name": "start-api", "run": ""}]} }
}`
				if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
					t.Fatalf("failed to write stub service.json: %v", err)
				}
				continue
			}
			if err := os.WriteFile(p, []byte("// stub"), 0o644); err != nil {
				t.Fatalf("failed to write stub file %s: %v", p, err)
			}
		}

		t.Setenv("TEMPLATE_PAYLOAD_DIR", payload)
		t.Setenv("GEN_OUTPUT_DIR", filepath.Join(t.TempDir(), "generated"))

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
	})

	t.Run("missing name", func(t *testing.T) {
		_, err := ts.GenerateScenario("test-template", "", "my-landing", nil)
		if err == nil {
			t.Error("GenerateScenario() should return error when name is empty")
		}
	})

	t.Run("missing slug", func(t *testing.T) {
		_, err := ts.GenerateScenario("test-template", "My Landing Page", "", nil)
		if err == nil {
			t.Error("GenerateScenario() should return error when slug is empty")
		}
	})

	t.Run("non-existing template", func(t *testing.T) {
		_, err := ts.GenerateScenario("non-existing", "My Landing Page", "my-landing", nil)
		if err == nil {
			t.Error("GenerateScenario() should return error for non-existing template")
		}
	})
}
