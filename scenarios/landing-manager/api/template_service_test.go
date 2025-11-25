package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// [REQ:TMPL-AVAILABILITY][REQ:TMPL-METADATA]
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

// [REQ:TMPL-MULTIPLE]
func TestTemplateService_ListMultipleTemplates(t *testing.T) {
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

// [REQ:TMPL-GENERATION][REQ:TMPL-OUTPUT-VALIDATION][REQ:TMPL-PROVENANCE][REQ:TMPL-DRY-RUN]
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

	t.Run("dry-run mode", func(t *testing.T) {
		// Provide a minimal payload
		payload := t.TempDir()
		minimalFiles := []string{
			filepath.Join(payload, "api", "main.go"),
			filepath.Join(payload, "ui", "src", "App.tsx"),
		}
		for _, p := range minimalFiles {
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatalf("failed to create payload dir %s: %v", p, err)
			}
			if err := os.WriteFile(p, []byte("// stub"), 0o644); err != nil {
				t.Fatalf("failed to write stub file %s: %v", p, err)
			}
		}

		t.Setenv("TEMPLATE_PAYLOAD_DIR", payload)
		t.Setenv("GEN_OUTPUT_DIR", filepath.Join(t.TempDir(), "generated"))

		opts := map[string]interface{}{"dry_run": true}
		result, err := ts.GenerateScenario("test-template", "Test Landing", "test-landing", opts)
		if err != nil {
			t.Errorf("GenerateScenario() in dry-run mode returned error: %v", err)
		}
		if result == nil {
			t.Fatal("GenerateScenario() returned nil result")
		}

		status, ok := result["status"].(string)
		if !ok || status != "dry_run" {
			t.Errorf("Expected status 'dry_run', got '%v'", result["status"])
		}

		// Check that plan was generated
		if plan, ok := result["plan"].(map[string]interface{}); !ok || plan == nil {
			t.Error("Expected plan to be generated in dry-run mode")
		}
	})
}

// [REQ:TMPL-GENERATION]
func TestTemplateService_ListGeneratedScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GEN_OUTPUT_DIR", tmpDir)

	ts := &TemplateService{templatesDir: t.TempDir()}

	t.Run("empty directory", func(t *testing.T) {
		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("ListGeneratedScenarios() returned error: %v", err)
		}
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("with generated scenarios", func(t *testing.T) {
		// Create a mock generated scenario
		scenarioPath := filepath.Join(tmpDir, "test-landing")
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write provenance
		provenance := map[string]interface{}{
			"template_id":      "saas-landing-page",
			"template_version": "1.0.0",
			"generated_at":     "2025-11-24T12:00:00Z",
		}
		provData, _ := json.Marshal(provenance)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "template.json"), provData, 0o644); err != nil {
			t.Fatalf("Failed to write template.json: %v", err)
		}

		// Write service config
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"displayName": "Test Landing Page",
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("ListGeneratedScenarios() returned error: %v", err)
		}

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.ScenarioID != "test-landing" {
			t.Errorf("Expected scenario_id 'test-landing', got '%s'", scenario.ScenarioID)
		}

		if scenario.Name != "Test Landing Page" {
			t.Errorf("Expected name 'Test Landing Page', got '%s'", scenario.Name)
		}

		if scenario.TemplateID != "saas-landing-page" {
			t.Errorf("Expected template_id 'saas-landing-page', got '%s'", scenario.TemplateID)
		}

		if scenario.TemplateVersion != "1.0.0" {
			t.Errorf("Expected template_version '1.0.0', got '%s'", scenario.TemplateVersion)
		}

		if scenario.Status != "present" {
			t.Errorf("Expected status 'present', got '%s'", scenario.Status)
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		// Set GEN_OUTPUT_DIR to a non-existent path
		t.Setenv("GEN_OUTPUT_DIR", filepath.Join(tmpDir, "non-existent"))

		scenarios, err := ts.ListGeneratedScenarios()
		if err != nil {
			t.Errorf("ListGeneratedScenarios() should not error on non-existent directory: %v", err)
		}
		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios for non-existent directory, got %d", len(scenarios))
		}
	})
}

func TestGenerationRoot(t *testing.T) {
	t.Run("with GEN_OUTPUT_DIR set", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("GEN_OUTPUT_DIR", tmpDir)

		root, err := generationRoot()
		if err != nil {
			t.Errorf("generationRoot() returned error: %v", err)
		}
		// Root should be an absolute path to tmpDir
		if !filepath.IsAbs(root) {
			t.Errorf("Expected absolute path, got '%s'", root)
		}
	})

	t.Run("default behavior", func(t *testing.T) {
		// Clear GEN_OUTPUT_DIR to use default
		t.Setenv("GEN_OUTPUT_DIR", "")

		root, err := generationRoot()
		if err != nil {
			t.Errorf("generationRoot() returned error: %v", err)
		}
		// Root should be an absolute path
		if !filepath.IsAbs(root) {
			t.Errorf("Expected absolute path, got '%s'", root)
		}
		// Root should end with "generated"
		if !strings.HasSuffix(root, "generated") {
			t.Errorf("Expected root to end with 'generated', got '%s'", root)
		}
	})
}

// [REQ:TMPL-PREVIEW-LINKS]
func TestTemplateService_GetPreviewLinks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GEN_OUTPUT_DIR", tmpDir)

	ts := &TemplateService{templatesDir: t.TempDir()}

	t.Run("success - get preview links for generated scenario", func(t *testing.T) {
		// Create a mock generated scenario with service.json
		scenarioID := "test-preview-landing"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write service.json with UI_PORT
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name":        scenarioID,
				"displayName": "Test Preview Landing",
			},
			"ports": map[string]interface{}{
				"API_PORT": 15000,
				"UI_PORT":  38000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		// Get preview links
		result, err := ts.GetPreviewLinks(scenarioID)
		if err != nil {
			t.Fatalf("GetPreviewLinks() returned error: %v", err)
		}

		// Verify scenario_id
		if sid, ok := result["scenario_id"].(string); !ok || sid != scenarioID {
			t.Errorf("Expected scenario_id '%s', got '%v'", scenarioID, result["scenario_id"])
		}

		// Verify path
		if path, ok := result["path"].(string); !ok || path != scenarioPath {
			t.Errorf("Expected path '%s', got '%v'", scenarioPath, result["path"])
		}

		// Verify base_url
		expectedBaseURL := "http://localhost:38000"
		if baseURL, ok := result["base_url"].(string); !ok || baseURL != expectedBaseURL {
			t.Errorf("Expected base_url '%s', got '%v'", expectedBaseURL, result["base_url"])
		}

		// Verify links object
		links, ok := result["links"].(map[string]string)
		if !ok {
			t.Fatal("Expected links to be a map[string]string")
		}

		expectedLinks := map[string]string{
			"public_landing": "http://localhost:38000/",
			"admin_portal":   "http://localhost:38000/admin",
			"admin_login":    "http://localhost:38000/admin/login",
			"health":         "http://localhost:38000/health",
		}

		for key, expectedURL := range expectedLinks {
			if actualURL, ok := links[key]; !ok || actualURL != expectedURL {
				t.Errorf("Expected links['%s'] = '%s', got '%s'", key, expectedURL, actualURL)
			}
		}

		// Verify instructions exist
		if instructions, ok := result["instructions"].([]string); !ok || len(instructions) == 0 {
			t.Error("Expected non-empty instructions array")
		}

		// Verify notes exist
		if notes, ok := result["notes"].(string); !ok || notes == "" {
			t.Error("Expected non-empty notes string")
		}
	})

	t.Run("scenario not found", func(t *testing.T) {
		_, err := ts.GetPreviewLinks("non-existent-scenario")
		if err == nil {
			t.Error("Expected error for non-existent scenario, got nil")
		}
	})

	t.Run("missing service.json", func(t *testing.T) {
		// Create a scenario directory without service.json
		scenarioID := "missing-service-json"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for missing service.json, got nil")
		}
	})

	t.Run("missing UI_PORT in service.json", func(t *testing.T) {
		// Create a scenario with service.json but no UI_PORT
		scenarioID := "missing-ui-port"
		scenarioPath := filepath.Join(tmpDir, scenarioID)
		if err := os.MkdirAll(filepath.Join(scenarioPath, ".vrooli"), 0o755); err != nil {
			t.Fatalf("Failed to create scenario directory: %v", err)
		}

		// Write service.json without UI_PORT
		serviceConfig := map[string]interface{}{
			"service": map[string]interface{}{
				"name": scenarioID,
			},
			"ports": map[string]interface{}{
				"API_PORT": 15000,
			},
		}
		svcData, _ := json.Marshal(serviceConfig)
		if err := os.WriteFile(filepath.Join(scenarioPath, ".vrooli", "service.json"), svcData, 0o644); err != nil {
			t.Fatalf("Failed to write service.json: %v", err)
		}

		_, err := ts.GetPreviewLinks(scenarioID)
		if err == nil {
			t.Error("Expected error for missing UI_PORT, got nil")
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES]
func TestTemplateService_GetPersonas(t *testing.T) {
	tmpDir := t.TempDir()
	personasDir := filepath.Join(tmpDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		t.Fatalf("Failed to create personas directory: %v", err)
	}

	// Create test persona catalog
	catalog := PersonaCatalog{
		Personas: []Persona{
			{
				ID:          "test-persona",
				Name:        "Test Persona",
				Description: "A test persona for unit testing",
				Prompt:      "This is a test prompt with guidance.",
				UseCases:    []string{"Testing", "Development"},
				Keywords:    []string{"test", "example"},
			},
			{
				ID:          "another-persona",
				Name:        "Another Persona",
				Description: "Another test persona",
				Prompt:      "Another test prompt.",
				UseCases:    []string{"Example"},
				Keywords:    []string{"sample"},
			},
		},
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	catalogData, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("Failed to marshal catalog: %v", err)
	}

	catalogPath := filepath.Join(personasDir, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogData, 0o644); err != nil {
		t.Fatalf("Failed to write catalog: %v", err)
	}

	ts := &TemplateService{
		templatesDir: filepath.Join(tmpDir, "templates"),
	}
	// Create templates dir so persona path resolution works
	if err := os.MkdirAll(ts.templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	t.Run("List all personas", func(t *testing.T) {
		personas, err := ts.GetPersonas()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(personas) != 2 {
			t.Errorf("Expected 2 personas, got %d", len(personas))
		}

		if personas[0].ID != "test-persona" {
			t.Errorf("Expected first persona ID 'test-persona', got %s", personas[0].ID)
		}

		if personas[0].Name != "Test Persona" {
			t.Errorf("Expected first persona name 'Test Persona', got %s", personas[0].Name)
		}

		if len(personas[0].UseCases) != 2 {
			t.Errorf("Expected 2 use cases, got %d", len(personas[0].UseCases))
		}

		if len(personas[0].Keywords) != 2 {
			t.Errorf("Expected 2 keywords, got %d", len(personas[0].Keywords))
		}
	})

	t.Run("Get specific persona by ID", func(t *testing.T) {
		persona, err := ts.GetPersona("test-persona")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if persona.ID != "test-persona" {
			t.Errorf("Expected ID 'test-persona', got %s", persona.ID)
		}

		if persona.Name != "Test Persona" {
			t.Errorf("Expected name 'Test Persona', got %s", persona.Name)
		}

		if persona.Description != "A test persona for unit testing" {
			t.Errorf("Expected description to match, got %s", persona.Description)
		}

		if !strings.Contains(persona.Prompt, "test prompt") {
			t.Errorf("Expected prompt to contain 'test prompt', got %s", persona.Prompt)
		}
	})

	t.Run("Get non-existent persona", func(t *testing.T) {
		_, err := ts.GetPersona("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent persona, got nil")
		}

		if !strings.Contains(err.Error(), "persona not found") {
			t.Errorf("Expected 'persona not found' error, got %v", err)
		}
	})

	t.Run("Get persona with missing catalog", func(t *testing.T) {
		// Remove catalog file
		if err := os.Remove(catalogPath); err != nil {
			t.Fatalf("Failed to remove catalog: %v", err)
		}

		_, err := ts.GetPersonas()
		if err == nil {
			t.Error("Expected error for missing catalog, got nil")
		}
	})
}
