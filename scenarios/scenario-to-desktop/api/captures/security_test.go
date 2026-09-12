package captures

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCapturePathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := capturePath(root, "../outside.mp4"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
}

func TestCapturePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "capture.mp4")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := capturePath(root, "capture.mp4"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
}
