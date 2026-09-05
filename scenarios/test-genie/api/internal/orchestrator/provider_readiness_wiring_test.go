package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

// The staleness gate fails open: a repo root that resolves wrong leaves the gate
// silently inert, and nothing downstream looks broken. These tests cover that
// wiring directly, because every other test in the tree constructs a Manager
// with RepoRoot and Ledger already set by hand.

func TestRepoRootResolvesFromScenariosRoot(t *testing.T) {
	repo := t.TempDir()
	scenarios := filepath.Join(repo, "scenarios")
	if err := os.MkdirAll(scenarios, 0o755); err != nil {
		t.Fatal(err)
	}

	o := &SuiteOrchestrator{scenariosRoot: scenarios}
	got := o.repoRoot()

	wantAbs, err := filepath.Abs(repo)
	if err != nil {
		t.Fatal(err)
	}
	if got != wantAbs {
		t.Fatalf("repoRoot() = %q, want %q — the staleness gate would look for providers in the wrong place", got, wantAbs)
	}
}

func TestRepoRootHandlesRelativeScenariosRoot(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}

	o := &SuiteOrchestrator{scenariosRoot: "scenarios"}
	got := o.repoRoot()

	// macOS /var -> /private/var symlinking makes a literal compare brittle;
	// what matters is that the result is absolute and contains scenarios/.
	if !filepath.IsAbs(got) {
		t.Fatalf("repoRoot() = %q, want an absolute path", got)
	}
	if _, err := os.Stat(filepath.Join(got, "scenarios")); err != nil {
		t.Fatalf("repoRoot() = %q does not contain a scenarios dir: %v", got, err)
	}
}

// An unset or unusable scenarios root must disable the gate rather than point it
// somewhere arbitrary.
func TestRepoRootEmptyWhenUnset(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    *SuiteOrchestrator
	}{
		{"nil orchestrator", nil},
		{"empty scenarios root", &SuiteOrchestrator{}},
		{"whitespace scenarios root", &SuiteOrchestrator{scenariosRoot: "   "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.o.repoRoot(); got != "" {
				t.Errorf("repoRoot() = %q, want \"\" so the gate stays inert", got)
			}
		})
	}
}

// The ledger has to land somewhere real and writable, otherwise the cooldown
// silently never fires across runs.
func TestCooldownLedgerPathIsWritable(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "scenarios"), 0o755); err != nil {
		t.Fatal(err)
	}
	o := &SuiteOrchestrator{scenariosRoot: filepath.Join(repo, "scenarios")}

	root := o.repoRoot()
	if root == "" {
		t.Fatal("repoRoot() was empty; the ledger would never be created")
	}
	// Mirror the path checkProviderReadiness builds.
	ledgerPath := filepath.Join(root, "scenarios", "test-genie", "coverage", "runtime", "provider-restarts.json")

	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o755); err != nil {
		t.Fatalf("ledger directory is not creatable: %v", err)
	}
	if err := os.WriteFile(ledgerPath, []byte(`{"providers":{}}`), 0o644); err != nil {
		t.Fatalf("ledger path is not writable: %v", err)
	}
	if _, err := os.Stat(ledgerPath); err != nil {
		t.Fatalf("ledger did not persist: %v", err)
	}
}
