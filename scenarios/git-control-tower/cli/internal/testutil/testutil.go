// Package testutil contains CLI-only test helpers. Production CLI packages
// must not import this package.
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// TempWorkspace creates an isolated filesystem workspace for CLI tests without
// initializing Git or invoking repository mutation commands.
func TempWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("create CLI workspace: %v", err)
	}
	return root
}
