package env

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSecretFromEnv(t *testing.T) {
	t.Setenv("LPBS_SERVICE_SECRET", "env-secret")
	if got := ResolveSecret("LPBS_SERVICE_SECRET"); got != "env-secret" {
		t.Fatalf("expected env-secret, got %q", got)
	}
}

func TestResolveSecretFromVrooliRootFile(t *testing.T) {
	root := t.TempDir()
	secretsDir := filepath.Join(root, ".vrooli")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "secrets.json"), []byte(`{"LPBS_SERVICE_SECRET":"file-secret"}`), 0o600); err != nil {
		t.Fatalf("write secrets: %v", err)
	}

	t.Setenv("VROOLI_ROOT", root)
	t.Setenv("LPBS_SERVICE_SECRET", "")

	if got := ResolveSecret("LPBS_SERVICE_SECRET"); got != "file-secret" {
		t.Fatalf("expected file-secret, got %q", got)
	}
}
