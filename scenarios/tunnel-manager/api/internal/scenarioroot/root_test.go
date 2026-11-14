package scenarioroot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveUsesCanonicalLifecycleRootWhenNestedScenariosExists(t *testing.T) {
	repoRoot := testRepoRoot(t)

	t.Setenv("VROOLI_SCENARIOS_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv("VROOLI_ROOT", "")

	resolver := New()
	got, err := resolver.ScenariosRoot()
	if err != nil {
		t.Fatalf("ScenariosRoot() error = %v", err)
	}
	if want := filepath.Join(repoRoot, "scenarios"); got != want {
		t.Fatalf("ScenariosRoot() = %q, want %q", got, want)
	}
}

func TestNewRejectsInvalidExplicitScenariosRoot(t *testing.T) {
	t.Setenv("VROOLI_SCENARIOS_ROOT", filepath.Join(t.TempDir(), "missing"))

	resolver := New()
	if _, err := resolver.ScenariosRoot(); err == nil {
		t.Fatal("ScenariosRoot() error = nil, want invalid override error")
	}
}

func TestServiceFileUsesContractWellKnownPath(t *testing.T) {
	repoRoot := testRepoRoot(t)
	t.Setenv("VROOLI_SCENARIOS_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", repoRoot)
	t.Setenv("VROOLI_ROOT", "")

	path, err := New().ServiceFile("secrets-manager")
	if err != nil {
		t.Fatalf("ServiceFile() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("ServiceFile() path %q is not readable: %v", path, err)
	}
	if want := filepath.Join(repoRoot, "scenarios", "secrets-manager", ".vrooli", "service.json"); path != want {
		t.Fatalf("ServiceFile() = %q, want %q", path, want)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve test working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("find repository root from test working directory")
		}
		dir = parent
	}
}
