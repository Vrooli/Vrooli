package filesystem

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestOSFilesystemOperations(t *testing.T) {
	root := t.TempDir()
	fsys := OS{}
	file := filepath.Join(root, "nested", "evidence.json")
	want := []byte("{\"ok\":true}\n")

	if fsys.Exists(file) {
		t.Fatal("nonexistent path reported as present")
	}
	if err := fsys.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := fsys.WriteFile(file, want, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !fsys.Exists(file) {
		t.Fatal("written file reported missing")
	}
	got, err := fsys.ReadFile(file)
	if err != nil || string(got) != string(want) {
		t.Fatalf("ReadFile = %q, %v; want %q", got, err, want)
	}
	info, err := fsys.Stat(file)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("Stat mode = %v, %v; want 0640", info, err)
	}
	entries, err := fsys.ReadDir(filepath.Dir(file))
	if err != nil || len(entries) != 1 || entries[0].Name() != "evidence.json" {
		t.Fatalf("ReadDir = %+v, %v", entries, err)
	}

	seenWalk, seenWalkDir := false, false
	if err := fsys.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		seenWalk = seenWalk || path == file
		return nil
	}); err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if err := fsys.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		seenWalkDir = seenWalkDir || path == file
		return nil
	}); err != nil {
		t.Fatalf("WalkDir: %v", err)
	}
	if !seenWalk || !seenWalkDir {
		t.Fatalf("walkers saw file: Walk=%v WalkDir=%v", seenWalk, seenWalkDir)
	}

	if err := fsys.Remove(file); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := fsys.ReadFile(file); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile after Remove error = %v; want not-exist", err)
	}
}
