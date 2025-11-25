package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
