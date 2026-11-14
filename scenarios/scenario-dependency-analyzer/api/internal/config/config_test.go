package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadNormalizesScenariosDir(t *testing.T) {
	tempDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})

	t.Setenv("API_PORT", "15413")
	t.Setenv("VROOLI_STORAGE_ROOT", tempDir)
	t.Setenv("VROOLI_SCENARIOS_DIR", filepath.Join("workspace", "scenarios"))

	cfg := Load()
	if !filepath.IsAbs(cfg.ScenariosDir) {
		t.Fatalf("ScenariosDir = %q, want absolute path", cfg.ScenariosDir)
	}
	if strings.Contains(cfg.ScenariosDir, "..") {
		t.Fatalf("ScenariosDir = %q, want cleaned path", cfg.ScenariosDir)
	}
	want := filepath.Join(tempDir, "workspace", "scenarios")
	if cfg.ScenariosDir != want {
		t.Fatalf("ScenariosDir = %q, want %q", cfg.ScenariosDir, want)
	}
}
