package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveAllRobust_RemovesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := removeAllRobust(root); err != nil {
		t.Fatalf("removeAllRobust: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected root removed; stat err=%v", err)
	}
}

func TestRemoveAllRobust_RefusesRoot(t *testing.T) {
	if err := removeAllRobust("/"); err == nil {
		t.Fatalf("expected error")
	}
}
