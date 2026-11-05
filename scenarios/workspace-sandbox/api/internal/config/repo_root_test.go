package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestResolveDefaultProjectRoot(t *testing.T) {
	t.Run("prefers explicit project root", func(t *testing.T) {
		t.Setenv("PROJECT_ROOT", "/tmp/custom")
		if got := ResolveDefaultProjectRoot(); got != "/tmp/custom" {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, "/tmp/custom")
		}
	})

	t.Run("resolves repo root from vrooli source root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("PROJECT_ROOT", "")
		t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))
		t.Setenv("VROOLI_ROOT", "")

		if got := ResolveDefaultProjectRoot(); got != repoRoot {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, repoRoot)
		}
	})

	t.Run("falls back to cwd repo root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("PROJECT_ROOT", "")
		t.Setenv("VROOLI_SOURCE_ROOT", "")
		t.Setenv("VROOLI_ROOT", "")
		t.Chdir(filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))

		if got := ResolveDefaultProjectRoot(); got != repoRoot {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, repoRoot)
		}
	})
}

func writeRepoContractFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	repocontracttest.WriteScenarioStub(t, root, "scenarios", "workspace-sandbox")
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "workspace-sandbox", "api"), 0o755); err != nil {
		t.Fatalf("mkdir workspace-sandbox api: %v", err)
	}
	return root
}
