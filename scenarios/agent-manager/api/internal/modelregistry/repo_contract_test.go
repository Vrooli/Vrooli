package modelregistry

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePathCanonicalizesContractDescendant(t *testing.T) {
	root := newModelRegistryContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "agent-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("AGENT_MANAGER_MODEL_REGISTRY_PATH", "")
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	got := ResolvePath()
	want := filepath.Join(root, "scenarios", "agent-manager", "config", "model-registry.json")
	if got != want {
		t.Fatalf("ResolvePath() = %q, want %q", got, want)
	}
}

func newModelRegistryContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := modelRegistryRepoRoot(t)
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/agent-manager-modelregistry-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func modelRegistryRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}
