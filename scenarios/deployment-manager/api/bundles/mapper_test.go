package bundles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deployment-manager/secrets"
)

func TestApplyBundleSecrets(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	bundleSecrets := []secrets.BundleSecret{
		{
			ID:          "session_secret",
			Class:       "user_prompt",
			Required:    true,
			Description: "Secret for signing cookies and sessions",
			Format:      "^[A-Za-z0-9]{32,}$",
			Target:      secrets.Target{Type: "env", Name: "SESSION_SECRET"},
			Prompt: &secrets.Prompt{
				Label:       "Session secret",
				Description: "Enter a 32+ character random string for signing cookies.",
			},
		},
		{
			ID:          "jwt_secret",
			Class:       "per_install_generated",
			Required:    true,
			Description: "JWT signing key",
			Format:      "^[A-Za-z0-9]{32,}$",
			Target:      secrets.Target{Type: "env", Name: "JWT_SECRET"},
			Generator: map[string]interface{}{
				"type":    "random",
				"length":  32,
				"charset": "alnum",
			},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets returned error: %v", err)
	}

	if len(manifest.Secrets) != len(bundleSecrets) {
		t.Fatalf("expected %d secrets, got %d", len(bundleSecrets), len(manifest.Secrets))
	}
	for i, s := range bundleSecrets {
		if manifest.Secrets[i].ID != s.ID {
			t.Fatalf("secret %d id mismatch: %s vs %s", i, manifest.Secrets[i].ID, s.ID)
		}
	}
}

func TestApplyBundleSecrets_NilManifest(t *testing.T) {
	bundleSecrets := []secrets.BundleSecret{
		{
			ID:     "test_secret",
			Class:  "user_prompt",
			Target: secrets.Target{Type: "env", Name: "TEST"},
		},
	}

	err := ApplyBundleSecrets(nil, bundleSecrets)
	if err == nil {
		t.Error("expected error for nil manifest, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error message to mention nil, got: %v", err)
	}
}

func TestApplyBundleSecrets_EmptySecrets(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Empty secrets list should be valid
	bundleSecrets := []secrets.BundleSecret{}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets should accept empty secrets: %v", err)
	}

	if len(manifest.Secrets) != 0 {
		t.Errorf("expected 0 secrets, got %d", len(manifest.Secrets))
	}
}

func TestApplyBundleSecrets_InvalidTargetType(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	bundleSecrets := []secrets.BundleSecret{
		{
			ID:     "test_secret",
			Class:  "user_prompt",
			Target: secrets.Target{Type: "invalid_type", Name: "TEST"},
		},
	}

	err = ApplyBundleSecrets(&manifest, bundleSecrets)
	if err == nil {
		t.Error("expected error for invalid target type, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "unsupported target type") {
		t.Errorf("expected error message about unsupported target type, got: %v", err)
	}
}

func TestApplyBundleSecrets_FileTargetType(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	bundleSecrets := []secrets.BundleSecret{
		{
			ID:          "file_secret",
			Class:       "per_install_generated",
			Required:    true,
			Description: "Secret stored in a file",
			Target:      secrets.Target{Type: "file", Name: "/path/to/secret.key"},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets should accept file target type: %v", err)
	}

	if len(manifest.Secrets) != 1 {
		t.Fatalf("expected 1 secret, got %d", len(manifest.Secrets))
	}

	if manifest.Secrets[0].Target.Type != "file" {
		t.Errorf("expected target type 'file', got '%s'", manifest.Secrets[0].Target.Type)
	}
}

func TestApplyBundleSecrets_WithoutPrompt(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Secret without prompt (generated secrets don't need prompts)
	bundleSecrets := []secrets.BundleSecret{
		{
			ID:          "generated_secret",
			Class:       "per_install_generated",
			Required:    true,
			Description: "Auto-generated secret",
			Target:      secrets.Target{Type: "env", Name: "GEN_SECRET"},
			Generator: map[string]interface{}{
				"type":   "random",
				"length": 64,
			},
			// No Prompt field
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets should accept secrets without prompts: %v", err)
	}

	if manifest.Secrets[0].Prompt != nil {
		t.Error("expected Prompt to be nil for generated secret")
	}
}

func TestApplyBundleSecrets_PreservesExistingFields(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Store original values
	originalSchemaVersion := manifest.SchemaVersion
	originalAppName := manifest.App.Name

	bundleSecrets := []secrets.BundleSecret{
		{
			ID:     "test",
			Class:  "user_prompt",
			Target: secrets.Target{Type: "env", Name: "TEST"},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets returned error: %v", err)
	}

	// Verify other fields were not modified
	if manifest.SchemaVersion != originalSchemaVersion {
		t.Errorf("schema_version changed from %s to %s", originalSchemaVersion, manifest.SchemaVersion)
	}
	if manifest.App.Name != originalAppName {
		t.Errorf("app.name changed from %s to %s", originalAppName, manifest.App.Name)
	}
}

func TestApplyBundleSecrets_FiltersInfrastructureSecrets(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Include infrastructure secret (should be filtered out) and bundle-safe secrets
	bundleSecrets := []secrets.BundleSecret{
		{
			ID:          "database_admin_password",
			Class:       "infrastructure", // Should be FILTERED OUT
			Required:    true,
			Description: "Admin database password - should never be bundled",
			Target:      secrets.Target{Type: "env", Name: "DB_ADMIN_PASSWORD"},
		},
		{
			ID:          "session_secret",
			Class:       "per_install_generated", // Should be included
			Required:    true,
			Description: "Session signing key",
			Target:      secrets.Target{Type: "env", Name: "SESSION_SECRET"},
		},
		{
			ID:          "api_key",
			Class:       "user_prompt", // Should be included
			Required:    true,
			Description: "User API key",
			Target:      secrets.Target{Type: "env", Name: "API_KEY"},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets returned error: %v", err)
	}

	// Should only have 2 secrets (infrastructure filtered out)
	if len(manifest.Secrets) != 2 {
		t.Fatalf("expected 2 secrets (infrastructure filtered), got %d", len(manifest.Secrets))
	}

	// Verify infrastructure secret was filtered
	for _, s := range manifest.Secrets {
		if s.ID == "database_admin_password" {
			t.Error("infrastructure secret should have been filtered out")
		}
		if s.Class == "infrastructure" {
			t.Error("no infrastructure class secrets should be in the manifest")
		}
	}

	// Verify bundle-safe secrets were included
	foundSession := false
	foundAPIKey := false
	for _, s := range manifest.Secrets {
		if s.ID == "session_secret" {
			foundSession = true
		}
		if s.ID == "api_key" {
			foundAPIKey = true
		}
	}
	if !foundSession {
		t.Error("expected session_secret to be in manifest")
	}
	if !foundAPIKey {
		t.Error("expected api_key to be in manifest")
	}
}

func TestApplyBundleSecrets_AllInfrastructureSecretsFiltered(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	// Only infrastructure secrets - all should be filtered
	bundleSecrets := []secrets.BundleSecret{
		{
			ID:     "db_password",
			Class:  "infrastructure",
			Target: secrets.Target{Type: "env", Name: "DB_PASSWORD"},
		},
		{
			ID:     "cloud_api_key",
			Class:  "infrastructure",
			Target: secrets.Target{Type: "env", Name: "CLOUD_API_KEY"},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets returned error: %v", err)
	}

	// Should have 0 secrets (all infrastructure filtered out)
	if len(manifest.Secrets) != 0 {
		t.Errorf("expected 0 secrets (all infrastructure filtered), got %d", len(manifest.Secrets))
	}
}

func TestApplyBundleSecrets_RequiredFlag(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	bundleSecrets := []secrets.BundleSecret{
		{
			ID:       "required_secret",
			Class:    "user_prompt",
			Required: true,
			Target:   secrets.Target{Type: "env", Name: "REQUIRED"},
		},
		{
			ID:       "optional_secret",
			Class:    "user_prompt",
			Required: false,
			Target:   secrets.Target{Type: "env", Name: "OPTIONAL"},
		},
	}

	if err := ApplyBundleSecrets(&manifest, bundleSecrets); err != nil {
		t.Fatalf("ApplyBundleSecrets returned error: %v", err)
	}

	if len(manifest.Secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(manifest.Secrets))
	}

	// Verify secret IDs are preserved
	if manifest.Secrets[0].ID != "required_secret" {
		t.Errorf("expected first secret ID 'required_secret', got %s", manifest.Secrets[0].ID)
	}
	if manifest.Secrets[1].ID != "optional_secret" {
		t.Errorf("expected second secret ID 'optional_secret', got %s", manifest.Secrets[1].ID)
	}

	// Verify Required pointer values are correctly set
	// This test catches the Go loop variable pointer aliasing bug (Go < 1.22)
	if manifest.Secrets[0].Required == nil {
		t.Fatal("first secret Required should not be nil")
	}
	if *manifest.Secrets[0].Required != true {
		t.Errorf("first secret Required should be true, got %v", *manifest.Secrets[0].Required)
	}

	if manifest.Secrets[1].Required == nil {
		t.Fatal("second secret Required should not be nil")
	}
	if *manifest.Secrets[1].Required != false {
		t.Errorf("second secret Required should be false, got %v", *manifest.Secrets[1].Required)
	}
}
