package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template represents a landing page template with full metadata
type Template struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Version              string                 `json:"version"`
	Metadata             map[string]interface{} `json:"metadata"`
	Sections             map[string]interface{} `json:"sections"`
	MetricsHooks         []MetricHook           `json:"metrics_hooks"`
	CustomizationSchema  map[string]interface{} `json:"customization_schema"`
	FrontendAesthetics   map[string]interface{} `json:"frontend_aesthetics,omitempty"`
	TechStack            map[string]interface{} `json:"tech_stack,omitempty"`
	GeneratedStructure   map[string]interface{} `json:"generated_structure,omitempty"`
}

// MetricHook defines an analytics event type
type MetricHook struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	EventType      string   `json:"event_type"`
	RequiredFields []string `json:"required_fields"`
}

// TemplateService handles template operations
type TemplateService struct {
	templatesDir string
}

// NewTemplateService creates a template service instance
func NewTemplateService() *TemplateService {
	// Determine templates directory relative to binary location
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)
	templatesDir := filepath.Join(execDir, "templates")

	return &TemplateService{
		templatesDir: templatesDir,
	}
}

// ListTemplates returns all available templates
func (ts *TemplateService) ListTemplates() ([]Template, error) {
	entries, err := os.ReadDir(ts.templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %w", err)
	}

	var templates []Template
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		templatePath := filepath.Join(ts.templatesDir, entry.Name())
		template, err := ts.loadTemplate(templatePath)
		if err != nil {
			// Log error but continue processing other templates
			logStructured("template_load_error", map[string]interface{}{
				"file":  entry.Name(),
				"error": err.Error(),
			})
			continue
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// GetTemplate returns a specific template by ID
func (ts *TemplateService) GetTemplate(id string) (*Template, error) {
	templatePath := filepath.Join(ts.templatesDir, id+".json")

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	template, err := ts.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	return &template, nil
}

// loadTemplate reads and parses a template JSON file
func (ts *TemplateService) loadTemplate(path string) (Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Template{}, fmt.Errorf("failed to read file: %w", err)
	}

	var template Template
	if err := json.Unmarshal(data, &template); err != nil {
		return Template{}, fmt.Errorf("failed to parse template JSON: %w", err)
	}

	return template, nil
}

// GenerateScenario creates a new landing page scenario from a template
func (ts *TemplateService) GenerateScenario(templateID, name, slug string, options map[string]interface{}) (map[string]interface{}, error) {
	template, err := ts.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// Validate inputs
	if name == "" || slug == "" {
		return nil, fmt.Errorf("name and slug are required")
	}

	// TODO: Implement actual scenario generation
	// This would:
	// 1. Copy template files to new scenario directory
	// 2. Replace placeholders with user-provided values
	// 3. Initialize database schema
	// 4. Create .vrooli/service.json config
	// 5. Generate Makefile and README

	result := map[string]interface{}{
		"scenario_id": slug,
		"name":        name,
		"template":    template.ID,
		"path":        fmt.Sprintf("/scenarios/%s", slug),
		"status":      "created",
		"next_steps": []string{
			"Review generated scenario files",
			"Customize content via CLI or admin portal",
			"Start scenario: vrooli scenario start " + slug,
			"Deploy to production when ready",
		},
	}

	return result, nil
}
