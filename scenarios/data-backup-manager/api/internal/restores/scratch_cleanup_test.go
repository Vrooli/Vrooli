package restores

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveScratchTreeRejectsPathOutsideDBMContract(t *testing.T) {
	root := t.TempDir()
	keep := filepath.Join(root, "keep.txt")
	if err := os.WriteFile(keep, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeScratchTree(root); err == nil {
		t.Fatal("removeScratchTree accepted an arbitrary directory")
	}
	if data, err := os.ReadFile(keep); err != nil || string(data) != "preserve" {
		t.Fatalf("outside path changed: data=%q err=%v", data, err)
	}
}

func TestRemoveScratchTreeUnlinksButDoesNotFollowSymlinks(t *testing.T) {
	parent := t.TempDir()
	outside := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(outside, []byte("preserve"), 0o600); err != nil {
		t.Fatal(err)
	}
	scratch := filepath.Join(parent, "dbm-verify-test-target-123")
	if err := os.Mkdir(scratch, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(scratch, "outside-link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := removeScratchTree(scratch); err != nil {
		t.Fatalf("removeScratchTree: %v", err)
	}
	if _, err := os.Lstat(scratch); !os.IsNotExist(err) {
		t.Fatalf("scratch tree remains: %v", err)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "preserve" {
		t.Fatalf("symlink target changed: data=%q err=%v", data, err)
	}
}
