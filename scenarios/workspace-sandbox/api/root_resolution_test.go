package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestResolveWorkspaceSandboxScenarioDir(t *testing.T) {
	t.Run("resolves from contract-aware env root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("VROOLI_SOURCE_ROOT", filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))
		t.Setenv("VROOLI_ROOT", "")

		got, err := resolveWorkspaceSandboxScenarioDir()
		if err != nil {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir: %v", err)
		}
		want := filepath.Join(repoRoot, "scenarios", "workspace-sandbox")
		if got != want {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir = %q, want %q", got, want)
		}
	})

	t.Run("falls back to cwd repo root", func(t *testing.T) {
		repoRoot := writeRepoContractFixture(t)
		t.Setenv("VROOLI_SOURCE_ROOT", "")
		t.Setenv("VROOLI_ROOT", "")
		t.Chdir(filepath.Join(repoRoot, "scenarios", "workspace-sandbox", "api"))

		got, err := resolveWorkspaceSandboxScenarioDir()
		if err != nil {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir: %v", err)
		}
		want := filepath.Join(repoRoot, "scenarios", "workspace-sandbox")
		if got != want {
			t.Fatalf("resolveWorkspaceSandboxScenarioDir = %q, want %q", got, want)
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

func TestResolveSQLiteDSNUsesAuthoritativeStorage(t *testing.T) {
	tmp := t.TempDir()
	// Storage is isolated by redirecting the class-root tree. This lever is
	// scenario-agnostic — every scenario beneath the root still resolves to its
	// own path — which is why it is safe where a database path variable was not.
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)

	dsn, err := resolveSQLiteDSN()
	if err != nil {
		t.Fatalf("resolveSQLiteDSN: %v", err)
	}
	if !strContains(dsn, tmp) {
		t.Fatalf("DSN did not resolve under the authoritative storage root %q: %q", tmp, dsn)
	}
	if !strContains(dsn, "workspace-sandbox") {
		t.Fatalf("DSN is not scoped to this scenario: %q", dsn)
	}
	for _, want := range []string{"_pragma=journal_mode(WAL)", "_txlock=immediate", "_pragma=foreign_keys(ON)"} {
		if !strContains(dsn, want) {
			t.Errorf("DSN missing %s: %q", want, dsn)
		}
	}
}

// TestResolveSQLiteDSNIgnoresInheritedDatabasePath is the regression test for
// the cross-scenario database hijack: a database path arriving in the
// environment must not be able to redirect this scenario's storage.
func TestResolveSQLiteDSNIgnoresInheritedDatabasePath(t *testing.T) {
	tmp := t.TempDir()
	prohibited := filepath.Join(tmp, "inherited", "autoheal.sqlite")
	t.Setenv("VROOLI_STORAGE_ROOT", tmp)
	t.Setenv("SQLITE_PATH", prohibited)
	t.Setenv("SQLITE_DB", prohibited)

	dsn, err := resolveSQLiteDSN()
	if err != nil {
		t.Fatalf("resolveSQLiteDSN: %v", err)
	}
	if strContains(dsn, "autoheal") || strContains(dsn, "inherited") {
		t.Fatalf("an inherited database path redirected this scenario: %q", dsn)
	}
	if _, err := os.Stat(filepath.Dir(prohibited)); err == nil {
		t.Errorf("the prohibited override's parent directory was created")
	}
}

func pathHasSuffix(dsn, suffix string) bool {
	// dsn is "<path>?<params>"; the path is everything before the first '?'
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == '?' {
			path := dsn[:i]
			return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
		}
	}
	return len(dsn) >= len(suffix) && dsn[len(dsn)-len(suffix):] == suffix
}

func strContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
