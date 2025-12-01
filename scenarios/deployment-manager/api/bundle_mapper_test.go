package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyBundleSecrets(t *testing.T) {
	path := filepath.Join("..", "..", "..", "docs", "deployment", "examples", "manifests", "desktop-happy.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read sample manifest: %v", err)
	}

	var manifest desktopBundleManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}

	secrets := []secretsManagerBundleSecret{
		{
			ID:          "session_secret",
			Class:       "user_prompt",
			Required:    true,
			Description: "Secret for signing cookies and sessions",
			Format:      "^[A-Za-z0-9]{32,}$",
			Target:      secretTarget{Type: "env", Name: "SESSION_SECRET"},
			Prompt: &secretPrompt{
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
			Target:      secretTarget{Type: "env", Name: "JWT_SECRET"},
			Generator: map[string]interface{}{
				"type":    "random",
				"length":  32,
				"charset": "alnum",
			},
		},
	}

	if err := applyBundleSecrets(&manifest, secrets); err != nil {
		t.Fatalf("applyBundleSecrets returned error: %v", err)
	}

	if len(manifest.Secrets) != len(secrets) {
		t.Fatalf("expected %d secrets, got %d", len(secrets), len(manifest.Secrets))
	}
	for i, s := range secrets {
		if manifest.Secrets[i].ID != s.ID {
			t.Fatalf("secret %d id mismatch: %s vs %s", i, manifest.Secrets[i].ID, s.ID)
		}
	}
}
