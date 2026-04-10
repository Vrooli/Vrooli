package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomicParentDirError(t *testing.T) {
	baseDir := t.TempDir()
	blocker := filepath.Join(baseDir, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}

	path := filepath.Join(blocker, "payload.json")
	if err := WriteJSONAtomic(path, map[string]string{"ok": "true"}); err == nil {
		t.Fatalf("expected error when parent dir is not a directory")
	}
}

func TestReadJSONBytesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	if _, err := ReadJSONBytes(path); err == nil {
		t.Fatalf("expected error for missing file")
	}
}
