package content

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaValidator_Validate_AllValid(t *testing.T) {
	// Create temp directories
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	// Create .vrooli directory
	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a minimal valid service.json
	serviceJSON := `{
		"version": "1.0.0",
		"service": {
			"name": "test-scenario"
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a minimal schema that accepts the service.json
	serviceSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["version", "service"],
		"properties": {
			"version": {"type": "string"},
			"service": {
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "service.schema.json"), []byte(serviceSchema), 0644); err != nil {
		t.Fatal(err)
	}

	// Configure validator to only check service.json
	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
	})

	result := validator.Validate()

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.ItemsChecked != 1 {
		t.Errorf("expected 1 item checked, got %d", result.ItemsChecked)
	}
}

func TestSchemaValidator_Validate_InvalidSchema(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create an invalid service.json (version should be string, not number)
	serviceJSON := `{
		"version": 123,
		"service": {
			"name": "test-scenario"
		}
	}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Schema expects string version
	serviceSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["version"],
		"properties": {
			"version": {"type": "string"}
		}
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "service.schema.json"), []byte(serviceSchema), 0644); err != nil {
		t.Fatal(err)
	}

	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
	})

	result := validator.Validate()

	if result.Success {
		t.Fatal("expected failure for invalid schema")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
}

func TestSchemaValidator_Validate_OptionalFileMissing(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create schema for the optional file
	testingSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "testing.schema.json"), []byte(testingSchema), 0644); err != nil {
		t.Fatal(err)
	}

	// No testing.json file created - it's optional
	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "testing.json", Required: false, SchemaFile: "testing.schema.json"},
	})

	result := validator.Validate()

	if !result.Success {
		t.Fatalf("expected success for missing optional file, got error: %v", result.Error)
	}
}

func TestSchemaValidator_Validate_RequiredFileMissing(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create schema
	serviceSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "service.schema.json"), []byte(serviceSchema), 0644); err != nil {
		t.Fatal(err)
	}

	// No service.json file created - it's required
	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
	})

	result := validator.Validate()

	if result.Success {
		t.Fatal("expected failure for missing required file")
	}
}

func TestSchemaValidator_Validate_InvalidJSON(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create invalid JSON
	invalidJSON := `{ invalid json }`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(invalidJSON), 0644); err != nil {
		t.Fatal(err)
	}

	serviceSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object"
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "service.schema.json"), []byte(serviceSchema), 0644); err != nil {
		t.Fatal(err)
	}

	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
	})

	result := validator.Validate()

	if result.Success {
		t.Fatal("expected failure for invalid JSON syntax")
	}
}

func TestSchemaValidator_Validate_MissingSchema(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	serviceJSON := `{"version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// No schema file created
	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
	})

	result := validator.Validate()

	if result.Success {
		t.Fatal("expected failure for missing schema file")
	}
	if result.FailureClass != FailureClassSystem {
		t.Errorf("expected system failure class for missing schema, got %s", result.FailureClass)
	}
}

func TestSchemaError_Format(t *testing.T) {
	err := &SchemaError{
		File:       "/path/to/.vrooli/service.json",
		SchemaPath: "/schemas/service.schema.json",
		DocPath:    "docs/service.md",
		Errors:     []string{"version: expected string, got number"},
	}

	formatted := err.Error()

	// Check that key parts are present
	if !contains(formatted, "service.json") {
		t.Error("expected file name in error message")
	}
	if !contains(formatted, "version: expected string") {
		t.Error("expected validation error in message")
	}
	if !contains(formatted, "Schema:") {
		t.Error("expected schema path reference")
	}
	if !contains(formatted, "Docs:") {
		t.Error("expected docs path reference")
	}
}

func TestDefaultConfigFiles(t *testing.T) {
	files := DefaultConfigFiles()

	// Should include the 4 core config files
	expected := map[string]bool{
		"service.json":    true,
		"testing.json":    true,
		"endpoints.json":  true,
		"lighthouse.json": true,
	}

	for _, f := range files {
		if _, ok := expected[f.Name]; !ok {
			t.Errorf("unexpected config file: %s", f.Name)
		}
		delete(expected, f.Name)
	}

	for name := range expected {
		t.Errorf("missing config file: %s", name)
	}

	// Verify service.json is required
	for _, f := range files {
		if f.Name == "service.json" && !f.Required {
			t.Error("service.json should be required")
		}
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		results  []SchemaValidateResult
		expected bool
	}{
		{
			name: "no errors",
			results: []SchemaValidateResult{
				{Valid: true},
				{Valid: true, Skipped: true},
			},
			expected: false,
		},
		{
			name: "has error",
			results: []SchemaValidateResult{
				{Valid: true},
				{Valid: false, Error: &SchemaError{Errors: []string{"test error"}}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasErrors(tt.results); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSchemaValidator_ValidateDetailed(t *testing.T) {
	scenarioDir := t.TempDir()
	schemasDir := t.TempDir()

	vrooliDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.Mkdir(vrooliDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create valid service.json
	serviceJSON := `{"version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(vrooliDir, "service.json"), []byte(serviceJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Create schema
	serviceSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"version": {"type": "string"}
		}
	}`
	if err := os.WriteFile(filepath.Join(schemasDir, "service.schema.json"), []byte(serviceSchema), 0644); err != nil {
		t.Fatal(err)
	}

	validator := NewSchemaValidator(scenarioDir, schemasDir, io.Discard)
	validator.WithConfigFiles([]ConfigFile{
		{Name: "service.json", Required: true, SchemaFile: "service.schema.json"},
		{Name: "testing.json", Required: false, SchemaFile: "testing.schema.json"},
	})

	// Create testing schema for the optional file
	testingSchema := `{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`
	if err := os.WriteFile(filepath.Join(schemasDir, "testing.schema.json"), []byte(testingSchema), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := validator.ValidateDetailed()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// First result should be valid (service.json)
	if !results[0].Valid {
		t.Error("expected service.json to be valid")
	}

	// Second result should be skipped (testing.json not present)
	if !results[1].Skipped {
		t.Error("expected testing.json to be skipped")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
