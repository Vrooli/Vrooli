package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSONAtomicReadOnlyDir(t *testing.T) {
	baseDir := t.TempDir()
	readonly := filepath.Join(baseDir, "readonly")
	if err := os.MkdirAll(readonly, 0o755); err != nil {
		t.Fatalf("create readonly dir: %v", err)
	}
	if err := os.Chmod(readonly, 0o555); err != nil {
		t.Fatalf("chmod readonly dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(readonly, 0o755)
	}()

	path := filepath.Join(readonly, "data.json")
	if err := WriteJSONAtomic(path, map[string]string{"ok": "true"}); err == nil {
		t.Fatalf("expected error when directory is read-only")
	}
}
