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
	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Source != "env" {
		t.Fatalf("expected source env, got %q", resolved.Source)
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
	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Source != "file" {
		t.Fatalf("expected source file, got %q", resolved.Source)
	}
	if resolved.SourcePath != filepath.Join(root, ".vrooli", "secrets.json") {
		t.Fatalf("unexpected source path: %q", resolved.SourcePath)
	}
}

func TestResolveSecretMissing(t *testing.T) {
	t.Setenv("VROOLI_ROOT", t.TempDir())
	t.Setenv("LPBS_SERVICE_SECRET", "")
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmpWD := t.TempDir()
	if err := os.Chdir(tmpWD); err != nil {
		t.Fatalf("chdir tempdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	resolved := ResolveSecretWithSource("LPBS_SERVICE_SECRET")
	if resolved.Value != "" {
		t.Fatalf("expected empty value, got %q", resolved.Value)
	}
	if resolved.Source != "missing" {
		t.Fatalf("expected source missing, got %q", resolved.Source)
	}
}
