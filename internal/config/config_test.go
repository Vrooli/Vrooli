package config

import (
	"os"
	"path/filepath"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-10

func TestHomeDirPrefersHOME(t *testing.T) {
	t.Setenv("HOME", "/tmp/vrooli-home")

	home, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}
	if home != "/tmp/vrooli-home" {
		t.Fatalf("home = %q", home)
	}
}

func TestTemplateBaseDirDefaultsToRepoTemplates(t *testing.T) {
	root := "/repo"
	if got := TemplateBaseDir(root); got != filepath.Join(root, "scripts", "scenarios", "templates") {
		t.Fatalf("TemplateBaseDir = %q", got)
	}
}

func TestTemplateBaseDirResolvesRelativeOverride(t *testing.T) {
	t.Setenv(TemplateBaseDirEnvVar, "custom/templates")
	root := "/repo"
	if got := TemplateBaseDir(root); got != filepath.Join(root, "custom", "templates") {
		t.Fatalf("TemplateBaseDir = %q", got)
	}
}

func TestHomeDirFallsBackToUserHomeDir(t *testing.T) {
	t.Setenv("HOME", "")

	want, wantErr := os.UserHomeDir()
	home, err := HomeDir()
	if wantErr != nil {
		if err == nil || err.Error() != wantErr.Error() {
			t.Fatalf("HomeDir error = %v, want %v", err, wantErr)
		}
		return
	}
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}
	if home != want {
		t.Fatalf("home = %q, want %q", home, want)
	}
}

func TestConfigDirectoryHelpers(t *testing.T) {
	if got := VrooliDir("/tmp/home"); got != filepath.Join("/tmp/home", ".vrooli") {
		t.Fatalf("VrooliDir = %q", got)
	}
	if got := RepoConfigDir("/repo"); got != filepath.Join("/repo", ".vrooli") {
		t.Fatalf("RepoConfigDir = %q", got)
	}
}

func TestTemplateBaseDirResolvesAbsoluteOverride(t *testing.T) {
	t.Setenv(TemplateBaseDirEnvVar, "/opt/templates")
	if got := TemplateBaseDir("/repo"); got != "/opt/templates" {
		t.Fatalf("TemplateBaseDir = %q", got)
	}
}
