package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
		chdirForTest(t, filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))

		if got := ResolveDefaultProjectRoot(); got != repoRoot {
			t.Fatalf("ResolveDefaultProjectRoot() = %q, want %q", got, repoRoot)
		}
	})
}

func writeRepoContractFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "workspace-sandbox", "api"), 0o755); err != nil {
		t.Fatalf("mkdir workspace-sandbox api: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "workspace-sandbox", ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir workspace-sandbox config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "workspace-sandbox", ".vrooli", "service.json"), []byte(`{"service":{"name":"workspace-sandbox"}}`), 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	// Copy the live repo's .vrooli/repo-contract.json verbatim rather than
	// hand-typing a literal. This keeps the single source of truth authoritative
	// and prevents the fixture from drifting when the contract schema gains a
	// required field (e.g. runtime_home).
	contract := liveRepoContract(t)
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
	return root
}

// liveRepoContract reads the repository's authoritative
// .vrooli/repo-contract.json by walking up from this source file until the
// contract is found, returning the raw bytes for verbatim copy into a fixture.
func liveRepoContract(t *testing.T) []byte {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed; cannot locate live repo contract")
	}
	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, ".vrooli", "repo-contract.json")
		if data, err := os.ReadFile(candidate); err == nil {
			return data
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate .vrooli/repo-contract.json above test package")
		}
		dir = parent
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
}
