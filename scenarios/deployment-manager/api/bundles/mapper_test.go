package bundles

import (
	"encoding/json"
	"os"
	"path/filepath"
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
