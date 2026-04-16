package bundles

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveScenarioRootCanonicalizesContractDescendant(t *testing.T) {
	root := newBundleContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "deployment-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	got := resolveScenarioRoot("alpha")
	want := filepath.Join(root, "scenarios", "alpha")
	if got != want {
		t.Fatalf("resolveScenarioRoot() = %q, want %q", got, want)
	}
}

func newBundleContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := bundleRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/deployment-manager-bundles-test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func bundleRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
}
