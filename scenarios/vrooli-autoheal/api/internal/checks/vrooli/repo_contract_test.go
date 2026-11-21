package vrooli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

func TestResolveVrooliRootCanonicalizesContractDescendant(t *testing.T) {
	root := newVrooliContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "vrooli-autoheal", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_ROOT", nested)

	if got := resolveVrooliRoot(); got != root {
		t.Fatalf("resolveVrooliRoot() = %q, want %q", got, root)
	}
}

func newVrooliContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := vrooliChecksRepoRoot(t)
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
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/vrooli-autoheal-checks-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func vrooliChecksRepoRoot(t *testing.T) string {
	t.Helper()
	if root, err := repocontract.ResolveRepoRoot(); err == nil {
		return root
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", ".."))
}
