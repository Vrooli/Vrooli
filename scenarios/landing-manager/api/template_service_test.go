package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:TMPL-AVAILABILITY][REQ:TMPL-METADATA]
func TestTemplateService_ListTemplates(t *testing.T) {
	t.Run("REQ:TMPL-AVAILABILITY", func(t *testing.T) {
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
	})

	t.Run("REQ:TMPL-METADATA", func(t *testing.T) {
		// Verify metadata fields are populated
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

		if len(templates) > 0 {
			if templates[0].Metadata == nil {
				t.Errorf("Expected metadata to be populated")
			}
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		ts := &TemplateService{templatesDir: tmpDir}
		templates, err := ts.ListTemplates()

		if err != nil {
			t.Errorf("ListTemplates() returned error: %v", err)
		}

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates in empty directory, got %d", len(templates))
		}
	})

	t.Run("non-JSON files ignored", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a non-JSON file
		if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text file"), 0644); err != nil {
			t.Fatalf("Failed to write text file: %v", err)
		}

		ts := &TemplateService{templatesDir: tmpDir}
		templates, err := ts.ListTemplates()

		if err != nil {
			t.Errorf("ListTemplates() returned error: %v", err)
		}

		if len(templates) != 0 {
			t.Errorf("Expected 0 templates (non-JSON files should be ignored), got %d", len(templates))
		}
	})
}

// [REQ:TMPL-MULTIPLE]
func TestTemplateService_ListMultipleTemplates(t *testing.T) {
	t.Run("REQ:TMPL-MULTIPLE", func(t *testing.T) {
		// Create temporary templates directory
		tmpDir := t.TempDir()

		// Create first template
		template1 := Template{
			ID:          "saas-landing",
			Name:        "SaaS Landing Page",
			Description: "SaaS product landing page",
			Version:     "1.0.0",
			Metadata: map[string]interface{}{
				"author":   "Vrooli Team",
				"category": "saas",
			},
		}
		data1, _ := json.Marshal(template1)
		if err := os.WriteFile(filepath.Join(tmpDir, "saas-landing.json"), data1, 0644); err != nil {
			t.Fatalf("Failed to write first template: %v", err)
		}

		// Create second template
		template2 := Template{
			ID:          "lead-magnet",
			Name:        "Lead Magnet",
			Description: "Lead capture landing page",
			Version:     "1.0.0",
			Metadata: map[string]interface{}{
				"author":   "Vrooli Team",
				"category": "lead-generation",
			},
		}
		data2, _ := json.Marshal(template2)
		if err := os.WriteFile(filepath.Join(tmpDir, "lead-magnet.json"), data2, 0644); err != nil {
			t.Fatalf("Failed to write second template: %v", err)
		}

		ts := &TemplateService{templatesDir: tmpDir}
		templates, err := ts.ListTemplates()

		if err != nil {
			t.Errorf("ListTemplates() returned error: %v", err)
		}

		// Verify we have exactly 2 templates
		if len(templates) != 2 {
			t.Fatalf("Expected 2 templates, got %d", len(templates))
		}

		// Verify both template IDs are present (order not guaranteed)
		foundIDs := make(map[string]bool)
		for _, tmpl := range templates {
			foundIDs[tmpl.ID] = true
		}

		if !foundIDs["saas-landing"] {
			t.Error("Expected to find 'saas-landing' template")
		}
		if !foundIDs["lead-magnet"] {
			t.Error("Expected to find 'lead-magnet' template")
		}

		// Verify template metadata is preserved
		for _, tmpl := range templates {
			if tmpl.Name == "" {
				t.Errorf("Template %s is missing name", tmpl.ID)
			}
			if tmpl.Description == "" {
				t.Errorf("Template %s is missing description", tmpl.ID)
			}
			if tmpl.Version != "1.0.0" {
				t.Errorf("Template %s has wrong version: %s", tmpl.ID, tmpl.Version)
			}
		}
	})

	t.Run("three templates", func(t *testing.T) {
		tmpDir := t.TempDir()

		templates := []Template{
			{ID: "template-1", Name: "Template 1", Description: "First", Version: "1.0.0"},
			{ID: "template-2", Name: "Template 2", Description: "Second", Version: "1.0.0"},
			{ID: "template-3", Name: "Template 3", Description: "Third", Version: "1.0.0"},
		}

		for _, tmpl := range templates {
			data, _ := json.Marshal(tmpl)
			if err := os.WriteFile(filepath.Join(tmpDir, tmpl.ID+".json"), data, 0644); err != nil {
				t.Fatalf("Failed to write template: %v", err)
			}
		}

		ts := &TemplateService{templatesDir: tmpDir}
		result, err := ts.ListTemplates()

		if err != nil {
			t.Errorf("ListTemplates() returned error: %v", err)
		}

		if len(result) != 3 {
			t.Errorf("Expected 3 templates, got %d", len(result))
		}
	})
}

// [REQ:TMPL-AVAILABILITY][REQ:TMPL-METADATA]
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

	t.Run("REQ:TMPL-AVAILABILITY/existing-template", func(t *testing.T) {
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

	t.Run("REQ:TMPL-AVAILABILITY/non-existing-template", func(t *testing.T) {
		template, err := ts.GetTemplate("non-existing")
		if err == nil {
			t.Error("GetTemplate() should return error for non-existing template")
		}
		if template != nil {
			t.Error("GetTemplate() should return nil template for non-existing ID")
		}
	})

	t.Run("REQ:TMPL-METADATA/template-fields", func(t *testing.T) {
		template, err := ts.GetTemplate("test-template")
		if err != nil {
			t.Fatalf("GetTemplate() returned error: %v", err)
		}

		if template.Name == "" {
			t.Error("Template Name should not be empty")
		}
		if template.Description == "" {
			t.Error("Template Description should not be empty")
		}
		if template.Version == "" {
			t.Error("Template Version should not be empty")
		}
	})

	t.Run("empty template ID", func(t *testing.T) {
		_, err := ts.GetTemplate("")
		if err == nil {
			t.Error("GetTemplate() should return error for empty template ID")
		}
	})
}
