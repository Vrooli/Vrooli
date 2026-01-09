package content

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateManifest_Valid(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

func TestValidateManifest_MissingFile(t *testing.T) {
	root := t.TempDir()
	// Don't create service.json

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure when service.json missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidateManifest_InvalidJSON(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{invalid json`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure for invalid JSON")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration, got %s", result.FailureClass)
	}
}

func TestValidateManifest_NameMismatch(t *testing.T) {
	root := t.TempDir()
	// Name doesn't match scenario directory
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "different-name"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure when name doesn't match")
	}
}

func TestValidateManifest_NameMismatchDisabled(t *testing.T) {
	root := t.TempDir()
	// Name doesn't match, but validation is disabled
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "different-name"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", false, io.Discard)
	if !result.Success {
		t.Fatalf("expected success when name validation disabled, got error: %v", result.Error)
	}
}

func TestValidateManifest_EmptyName(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": ""},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", false, io.Discard)
	if result.Success {
		t.Fatal("expected failure when service name is empty")
	}
}

func TestValidateManifest_MissingHealthChecks(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": []}}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure when no health checks defined")
	}
}

func TestValidateManifest_NoHealthSection(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure when health section missing")
	}
}

func TestValidateManifest_MultipleHealthChecks(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": [{}, {}, {}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if !result.Success {
		t.Fatalf("expected success with multiple health checks, got error: %v", result.Error)
	}
}

func TestValidateManifest_Observations(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	// Should have success observations
	if len(result.Observations) == 0 {
		t.Error("expected observations to be present")
	}

	hasNameObs := false
	hasHealthObs := false
	for _, obs := range result.Observations {
		if obs.Type == ObservationSuccess {
			if containsStrManifest(obs.Message, "name") {
				hasNameObs = true
			}
			if containsStrManifest(obs.Message, "Health") {
				hasHealthObs = true
			}
		}
	}
	if !hasNameObs {
		t.Error("expected observation about service name")
	}
	if !hasHealthObs {
		t.Error("expected observation about health checks")
	}
}

func TestManifestValidator_Interface(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	v := NewManifestValidator(root, "test-scenario", io.Discard)
	result := v.Validate()

	if !result.Success {
		t.Errorf("Validate() failed: %v", result.Error)
	}
}

func TestManifestValidator_WithNameValidationOption(t *testing.T) {
	root := t.TempDir()
	// Name doesn't match
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "different"},
		"lifecycle": {"health": {"checks": [{}]}}
	}`)

	// With name validation enabled (should fail)
	vEnabled := NewManifestValidator(root, "test-scenario", io.Discard, WithNameValidation(true))
	resultEnabled := vEnabled.Validate()
	if resultEnabled.Success {
		t.Error("expected failure with name validation enabled")
	}

	// With name validation disabled (should succeed)
	vDisabled := NewManifestValidator(root, "test-scenario", io.Discard, WithNameValidation(false))
	resultDisabled := vDisabled.Validate()
	if !resultDisabled.Success {
		t.Errorf("expected success with name validation disabled, got: %v", resultDisabled.Error)
	}
}

func TestValidateManifest_DirectoryAsManifest(t *testing.T) {
	root := t.TempDir()
	vrooliDir := filepath.Join(root, ".vrooli")
	mustMkdirManifest(t, vrooliDir)

	// Create a directory where service.json should be
	mustMkdirManifest(t, filepath.Join(vrooliDir, "service.json"))

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if result.Success {
		t.Fatal("expected failure when service.json is a directory")
	}
}

func TestValidateManifest_WithDependencies(t *testing.T) {
	root := t.TempDir()
	setupManifest(t, root, "test-scenario", `{
		"service": {"name": "test-scenario"},
		"lifecycle": {"health": {"checks": [{}]}},
		"dependencies": {
			"resources": {
				"postgres": {"required": true, "enabled": true},
				"redis": {"required": false, "enabled": true}
			}
		}
	}`)

	result := ValidateManifest(root, "test-scenario", true, io.Discard)
	if !result.Success {
		t.Fatalf("expected success with dependencies, got error: %v", result.Error)
	}
}

// Test helpers

func setupManifest(t *testing.T, root, scenarioName, content string) {
	t.Helper()
	vrooliDir := filepath.Join(root, ".vrooli")
	mustMkdirManifest(t, vrooliDir)
	writeFileManifest(t, filepath.Join(vrooliDir, "service.json"), content)
}

func mustMkdirManifest(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}

func writeFileManifest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func containsStrManifest(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
