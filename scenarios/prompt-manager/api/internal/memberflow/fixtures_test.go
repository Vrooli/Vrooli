package memberflow

import (
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/internal/testutil/fixtures"
)

// Typed fixture builders live in this package rather than in
// internal/testutil/fixtures because that package imports memberflow, and
// every test file here is `package memberflow`. Domain-free helpers
// (WriteJSON, RepositoryRoot) still come from the shared package; only the
// memberflow-typed builders are local. Fixtures marshal the real structs so a
// schema change breaks compilation instead of leaving stale JSON behind.

// writeRepoFile writes an arbitrary fixture file under root, creating parents.
// It replaces the per-file os.MkdirAll+os.WriteFile pairs that each test used
// to hand-roll.
func writeRepoFile(t testing.TB, root, relativePath, content string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", relativePath, err)
	}
	return path
}

// requireRepositoryRoot resolves the real checkout for tests that read
// committed repository content, skipping when no repository is available.
// It replaces five separate locators that used three different strategies and
// three different failure modes, so a missing checkout now always skips and a
// malformed one always fails.
func requireRepositoryRoot(t testing.TB) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}
	root, err := fixtures.RepositoryRoot(wd)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("no Vrooli repository checkout available for fixture")
		}
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// requirePromptManagerStoreDir resolves the committed prompt-manager store
// through the same repository resolver, replacing a runtime.Caller walk that
// hard-coded the number of parent directories between a test file and the
// scenario root.
func requirePromptManagerStoreDir(t testing.TB) string {
	t.Helper()
	return filepath.Join(requireRepositoryRoot(t), "scenarios", "prompt-manager", "store")
}
