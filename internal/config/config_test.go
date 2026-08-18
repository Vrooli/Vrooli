package config

import (
	osuser "os/user"
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
	if got := TemplateBaseDir(root); got != filepath.Join(root, "templates", "scenarios") {
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

func TestHomeDirFallsBackToCurrentUserHomeDir(t *testing.T) {
	t.Setenv("HOME", "")

	current, currentErr := osuser.Current()
	home, err := HomeDir()
	if currentErr != nil {
		if err == nil {
			t.Fatal("HomeDir unexpectedly succeeded when the current user cannot be resolved")
		}
		return
	}
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}
	if home != current.HomeDir {
		t.Fatalf("home = %q, want current user's home %q", home, current.HomeDir)
	}
}

func TestConfigDirectoryHelpers(t *testing.T) {
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
