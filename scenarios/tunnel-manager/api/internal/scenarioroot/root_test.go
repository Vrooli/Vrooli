package scenarioroot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveUsesCanonicalLifecycleRootWhenNestedScenariosExists(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../../../"))

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
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "../../../../../"))
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
