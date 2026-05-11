package projectstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocatorUsesUserHomeProjectState(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "Vrooli")
	locator, err := NewLocator(home, root)
	if err != nil {
		t.Fatalf("NewLocator: %v", err)
	}

	stateDir := locator.SetupStateDir()
	if !strings.HasPrefix(stateDir, filepath.Join(home, ".vrooli", "state", "projects")+string(filepath.Separator)) {
		t.Fatalf("state dir %q is not under user-home project state", stateDir)
	}
	if strings.Contains(stateDir, filepath.Join(root, ".vrooli")) {
		t.Fatalf("state dir %q must not use repo-local .vrooli", stateDir)
	}
	if !strings.HasPrefix(locator.ProjectKey(), "Vrooli-") {
		t.Fatalf("project key = %q, want basename prefix", locator.ProjectKey())
	}
}

func TestLocatorProjectKeysAreDeterministicAndRootScoped(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(t.TempDir(), "Vrooli")
	first, err := NewLocator(home, root)
	if err != nil {
		t.Fatalf("first NewLocator: %v", err)
	}
	second, err := NewLocator(home, root)
	if err != nil {
		t.Fatalf("second NewLocator: %v", err)
	}
	other, err := NewLocator(home, filepath.Join(t.TempDir(), "Vrooli"))
	if err != nil {
		t.Fatalf("other NewLocator: %v", err)
	}

	if first.ProjectKey() != second.ProjectKey() {
		t.Fatalf("project key not deterministic: %q != %q", first.ProjectKey(), second.ProjectKey())
	}
	if first.ProjectKey() == other.ProjectKey() {
		t.Fatalf("different roots produced same project key %q", first.ProjectKey())
	}
}

func TestHasSetupCompleteIgnoresRepoLocalLegacyMarker(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	legacyDir := filepath.Join(root, ".vrooli", "state", "setup")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy state dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, ".setup-complete"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}

	locator, err := NewLocator(home, root)
	if err != nil {
		t.Fatalf("NewLocator: %v", err)
	}
	if locator.HasSetupComplete() {
		t.Fatal("legacy repo-local setup marker must not be detected")
	}
	if err := os.MkdirAll(locator.SetupStateDir(), 0o755); err != nil {
		t.Fatalf("mkdir setup state dir: %v", err)
	}
	if err := os.WriteFile(locator.SetupCompletePath(), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}
	if !locator.HasSetupComplete() {
		t.Fatal("expected user-home setup marker to be detected")
	}
}

func TestHasResourcePopulatedUsesUserHomeMarkers(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	locator, err := NewLocator(home, root)
	if err != nil {
		t.Fatalf("NewLocator: %v", err)
	}
	if locator.HasResourcePopulated("postgres") {
		t.Fatal("expected missing resource marker")
	}
	if err := os.MkdirAll(locator.SetupStateDir(), 0o755); err != nil {
		t.Fatalf("mkdir setup state dir: %v", err)
	}
	if err := os.WriteFile(locator.ResourcePopulatedPath("postgres"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write resource marker: %v", err)
	}
	if !locator.HasResourcePopulated("postgres") {
		t.Fatal("expected user-home resource marker to be detected")
	}
	if strings.Contains(locator.ResourcePopulatedPath("../bad/name"), "..") {
		t.Fatalf("resource marker path should sanitize traversal: %q", locator.ResourcePopulatedPath("../bad/name"))
	}
}
