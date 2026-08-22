package manifest_test

import (
	"os"
	"path/filepath"
	"testing"
)

// findRepoRoot walks up from the test's working directory to the repository
// root, so a test that reads a real manifest does not depend on where it runs.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "resources")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found above the test working directory")
		}
		dir = parent
	}
}
