package executionwriter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryRefreshRejectsUnknownTopLevelFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "result.json"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated.txt"), []byte("not BAS owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryDirectoryFiles(root); err == nil {
		t.Fatal("inventory refresh blessed an unknown top-level file")
	}
}

func TestInventoryRefreshRejectsUnknownArtifactSubtree(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "artifacts", "private", "payload.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not BAS owned"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := inventoryDirectoryFiles(root); err == nil {
		t.Fatal("inventory refresh blessed an unknown artifact subtree")
	}
}
