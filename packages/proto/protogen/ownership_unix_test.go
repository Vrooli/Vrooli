//go:build !windows

package protogen

import (
	"os"
	"path/filepath"
	"testing"
)

// An unprivileged caller must leave the tree alone, and a tree already owned
// by the parent's owner must stay valid.
func TestHandBackOwnershipIsNoOpWithoutPrivilege(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("covers the unprivileged path")
	}
	parent := t.TempDir()
	tree := filepath.Join(parent, "gen")
	if err := os.MkdirAll(filepath.Join(tree, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := handBackOwnership(tree, parent); err != nil {
		t.Fatalf("handBackOwnership: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tree, "go")); err != nil {
		t.Fatalf("tree changed: %v", err)
	}
}
