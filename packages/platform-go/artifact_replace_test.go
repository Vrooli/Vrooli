package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicReplaceFilePreservesTargetOnSuccess(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "app")
	staged := filepath.Join(root, "staged")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := AtomicReplace(staged, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("target = %q, want new", got)
	}
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("staged artifact still exists: %v", err)
	}
}

func TestAtomicReplaceDirectoryReplacesCompleteTree(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "dist")
	staged := filepath.Join(root, "stage")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(staged, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staged, "index.html"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staged, "assets.js"), []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := AtomicReplace(staged, target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("index.html = %q, want new", got)
	}
	if _, err := os.Stat(filepath.Join(target, "assets.js")); err != nil {
		t.Fatalf("new tree missing assets.js: %v", err)
	}
}
