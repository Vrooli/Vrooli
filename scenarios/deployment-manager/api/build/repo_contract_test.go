package build

import (
	"os"
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestNewHandlerCanonicalizesContractDescendant(t *testing.T) {
	root := newBuildContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "deployment-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	handler := NewHandler(nil, nil)
	if handler.vrooli != root {
		t.Fatalf("handler.vrooli = %q, want %q", handler.vrooli, root)
	}
}

func TestResolveScenarioDirUsesContractScenarioPath(t *testing.T) {
	root := newBuildContractFixtureRepo(t)
	got := resolveScenarioDir(root, "alpha")
	want := filepath.Join(root, "scenarios", "alpha")
	if got != want {
		t.Fatalf("resolveScenarioDir() = %q, want %q", got, want)
	}
}

func newBuildContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := buildRepoRoot(t)
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/deployment-manager-build-test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func buildRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}
