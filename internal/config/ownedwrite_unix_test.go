//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReconcileHomeOwnershipSkipsNonRoot validates the CD-7 invariant that
// reconcile never reassigns entries owned by a non-root user. The test files are
// created by the (non-root) test process, so a reconcile targeting the current
// uid must touch nothing. This exercises the walk + skip logic without needing
// real root.
func TestReconcileHomeOwnershipSkipsNonRoot(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "state"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "state", "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := reconcileHomeOwnership(home, os.Getuid(), os.Getgid())
	if err != nil {
		t.Fatalf("reconcileHomeOwnership: %v", err)
	}
	if changed != 0 {
		t.Fatalf("expected 0 reowned (no root-owned entries present), got %d", changed)
	}
}
