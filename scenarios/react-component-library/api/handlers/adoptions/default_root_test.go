package adoptions

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDefaultScenariosRoot_ResolvesViaContract creates a minimal repo layout
// in a temp dir, points VROOLI_ROOT at it, and asserts defaultScenariosRoot
// resolves to <temp>/scenarios via packages/repo-contract-go. Regression
// guard against the off-by-one bug where defaultScenariosRoot returned
// <repo>/scenarios/scenarios because of a redundant filepath.Join.
func TestDefaultScenariosRoot_ResolvesViaContract(t *testing.T) {
	tempRepo := t.TempDir()

	// Minimal repo-contract requires root markers + the contract file.
	for _, dir := range []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(tempRepo, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(tempRepo, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	// Copy the real repo-contract.json so we exercise the canonical schema.
	src, err := os.ReadFile(filepath.Join(repoRootForTest(t), ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read source contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempRepo, ".vrooli", "repo-contract.json"), src, 0o644); err != nil {
		t.Fatalf("write contract: %v", err)
	}

	t.Setenv("VROOLI_ROOT", tempRepo)
	t.Setenv("VROOLI_SOURCE_ROOT", tempRepo)
	t.Setenv("ADOPTIONS_SCENARIOS_ROOT", "") // make sure no override leaks

	got, err := defaultScenariosRoot()
	if err != nil {
		t.Fatalf("defaultScenariosRoot: %v", err)
	}
	want := filepath.Join(tempRepo, "scenarios")
	if got != want {
		t.Fatalf("defaultScenariosRoot() = %q; want %q", got, want)
	}
}

// TestDefaultScenariosRoot_EnvOverride confirms the ADOPTIONS_SCENARIOS_ROOT
// env override still wins (kept for backwards compat with existing tests).
func TestDefaultScenariosRoot_EnvOverride(t *testing.T) {
	t.Setenv("ADOPTIONS_SCENARIOS_ROOT", "/explicit/path")
	got, err := defaultScenariosRoot()
	if err != nil {
		t.Fatalf("defaultScenariosRoot: %v", err)
	}
	if got != "/explicit/path" {
		t.Fatalf("defaultScenariosRoot() = %q; want /explicit/path", got)
	}
}

// repoRootForTest finds the actual repo root from the test binary's CWD so
// we can copy the canonical repo-contract.json fixture.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	// Tests run with CWD = scenarios/react-component-library/api by default.
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli", "repo-contract.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate repo-contract.json walking up from %s", cwd)
	return ""
}
