package fswatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherSignalsNestedWrite(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "packages", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := newWithInterval([]string{filepath.Join(root, "packages")}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := os.WriteFile(filepath.Join(dir, "source.go"), []byte("package demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.Events():
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not signal nested write")
	}
}
