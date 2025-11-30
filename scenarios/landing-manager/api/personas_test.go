package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"landing-manager/services"
)

// [REQ:TMPL-AGENT-PROFILES]
func TestGetPersonas_ListAll(t *testing.T) {
	tmpDir := t.TempDir()
	personasDir := filepath.Join(tmpDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		t.Fatalf("Failed to create personas directory: %v", err)
	}

	// Create test persona catalog
	catalog := services.PersonaCatalog{
		Personas: []services.Persona{
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

	// Create templates dir so persona path resolution works
	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	ps := services.NewPersonaService(templatesDir)

	t.Run("REQ:TMPL-AGENT-PROFILES/list-all-personas", func(t *testing.T) {
		personas, err := ps.GetPersonas()
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

	t.Run("REQ:TMPL-AGENT-PROFILES/verify-all-fields", func(t *testing.T) {
		personas, err := ps.GetPersonas()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		for _, p := range personas {
			if p.ID == "" {
				t.Error("Persona ID should not be empty")
			}
			if p.Name == "" {
				t.Error("Persona Name should not be empty")
			}
			if p.Description == "" {
				t.Error("Persona Description should not be empty")
			}
			if p.Prompt == "" {
				t.Error("Persona Prompt should not be empty")
			}
			if len(p.UseCases) == 0 {
				t.Errorf("Persona %s should have at least one use case", p.ID)
			}
			if len(p.Keywords) == 0 {
				t.Errorf("Persona %s should have at least one keyword", p.ID)
			}
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES]
func TestGetPersona_SpecificPersona(t *testing.T) {
	tmpDir := t.TempDir()
	personasDir := filepath.Join(tmpDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		t.Fatalf("Failed to create personas directory: %v", err)
	}

	catalog := services.PersonaCatalog{
		Personas: []services.Persona{
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

	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	ps := services.NewPersonaService(templatesDir)

	t.Run("REQ:TMPL-AGENT-PROFILES/get-specific-persona", func(t *testing.T) {
		persona, err := ps.GetPersona("test-persona")
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

	t.Run("REQ:TMPL-AGENT-PROFILES/get-second-persona", func(t *testing.T) {
		persona, err := ps.GetPersona("another-persona")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if persona.ID != "another-persona" {
			t.Errorf("Expected ID 'another-persona', got %s", persona.ID)
		}

		if persona.Name != "Another Persona" {
			t.Errorf("Expected name 'Another Persona', got %s", persona.Name)
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES]
func TestGetPersona_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()
	personasDir := filepath.Join(tmpDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		t.Fatalf("Failed to create personas directory: %v", err)
	}

	catalog := services.PersonaCatalog{
		Personas: []services.Persona{
			{
				ID:          "exists",
				Name:        "Exists",
				Description: "This persona exists",
				Prompt:      "Test",
				UseCases:    []string{"Test"},
				Keywords:    []string{"test"},
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

	templatesDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(templatesDir, 0o755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	ps := services.NewPersonaService(templatesDir)

	t.Run("non-existent persona", func(t *testing.T) {
		_, err := ps.GetPersona("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent persona, got nil")
		}

		if !strings.Contains(err.Error(), "persona not found") {
			t.Errorf("Expected 'persona not found' error, got %v", err)
		}
	})

	t.Run("empty persona ID", func(t *testing.T) {
		_, err := ps.GetPersona("")
		if err == nil {
			t.Error("Expected error for empty persona ID, got nil")
		}
	})

	t.Run("missing catalog file", func(t *testing.T) {
		// Remove catalog file
		if err := os.Remove(catalogPath); err != nil {
			t.Fatalf("Failed to remove catalog: %v", err)
		}

		_, err := ps.GetPersonas()
		if err == nil {
			t.Error("Expected error for missing catalog, got nil")
		}
	})

	t.Run("invalid catalog JSON", func(t *testing.T) {
		// Write invalid JSON
		if err := os.WriteFile(catalogPath, []byte("not valid json"), 0o644); err != nil {
			t.Fatalf("Failed to write invalid catalog: %v", err)
		}

		_, err := ps.GetPersonas()
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})
}

// [REQ:TMPL-AGENT-PROFILES]
func TestPersonaCatalog_Structure(t *testing.T) {
	tmpDir := t.TempDir()
	personasDir := filepath.Join(tmpDir, "personas")
	if err := os.MkdirAll(personasDir, 0o755); err != nil {
		t.Fatalf("Failed to create personas directory: %v", err)
	}

	t.Run("catalog with metadata", func(t *testing.T) {
		catalog := services.PersonaCatalog{
			Personas: []services.Persona{
				{
					ID:          "test",
					Name:        "Test",
					Description: "Test persona",
					Prompt:      "Test prompt",
					UseCases:    []string{"Testing"},
					Keywords:    []string{"test"},
				},
			},
			Metadata: map[string]interface{}{
				"version":     "1.0.0",
				"author":      "Test Author",
				"created_at":  "2025-11-26",
				"description": "Test catalog",
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

		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates directory: %v", err)
		}

		ps := services.NewPersonaService(templatesDir)

		personas, err := ps.GetPersonas()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(personas) != 1 {
			t.Errorf("Expected 1 persona, got %d", len(personas))
		}
	})

	t.Run("empty persona list", func(t *testing.T) {
		catalog := services.PersonaCatalog{
			Personas: []services.Persona{},
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

		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates directory: %v", err)
		}

		ps := services.NewPersonaService(templatesDir)

		personas, err := ps.GetPersonas()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(personas) != 0 {
			t.Errorf("Expected 0 personas, got %d", len(personas))
		}
	})

	t.Run("large persona catalog", func(t *testing.T) {
		// Create a catalog with many personas
		personas := make([]services.Persona, 50)
		for i := 0; i < 50; i++ {
			personas[i] = services.Persona{
				ID:          string(rune('a' + i)),
				Name:        string(rune('A' + i)),
				Description: "Test persona " + string(rune('0'+i)),
				Prompt:      "Prompt " + string(rune('0'+i)),
				UseCases:    []string{"Use case"},
				Keywords:    []string{"keyword"},
			}
		}

		catalog := services.PersonaCatalog{
			Personas: personas,
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

		templatesDir := filepath.Join(tmpDir, "templates")
		if err := os.MkdirAll(templatesDir, 0o755); err != nil {
			t.Fatalf("Failed to create templates directory: %v", err)
		}

		ps := services.NewPersonaService(templatesDir)

		loadedPersonas, err := ps.GetPersonas()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(loadedPersonas) != 50 {
			t.Errorf("Expected 50 personas, got %d", len(loadedPersonas))
		}
	})
}
