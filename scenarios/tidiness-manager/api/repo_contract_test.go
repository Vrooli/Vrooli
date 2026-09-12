package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	dirs := []string{
		".vrooli",
		"scenarios",
		"resources",
		"templates",
		"packages",
		"cmd",
		"internal",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("Failed to create repo dir %s: %v", dir, err)
		}
	}

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test-repo\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	// Copy the live repo's .vrooli/repo-contract.json verbatim rather than
	// hand-typing a literal. This keeps the single source of truth authoritative
	// and prevents the fixture from drifting when the contract schema gains a
	// required field (e.g. runtime_home).
	contract := liveRepoContract(t)
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contract, 0o644); err != nil {
		t.Fatalf("Failed to write repo contract: %v", err)
	}
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
