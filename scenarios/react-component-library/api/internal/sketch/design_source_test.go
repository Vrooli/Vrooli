package sketch

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestDesignSourceSnapshotExactBoundedAndConfined(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	content := "# Design\r\nPreserve density. 日本語\n"
	write("docs/DESIGN.md", []byte(content))
	store := NewStore(root)
	got, err := store.ReadDesignSource("demo", "path:docs/DESIGN.md")
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != "docs/DESIGN.md" || got.Content != content || got.ContentHash != fmt.Sprintf("%x", sha256.Sum256([]byte(content))) {
		t.Fatalf("source bytes changed: %+v", got)
	}
	write("docs/DESIGN.md", []byte(content+"Changed."))
	updated, err := store.ReadDesignSource("demo", "docs/DESIGN.md")
	if err != nil || updated.ContentHash == got.ContentHash {
		t.Fatal("source change not reflected", err)
	}
	write("large", []byte(strings.Repeat("x", maxDesignSourceBytes+1)))
	write("binary", []byte{0xff})
	write("nul", []byte{'a', 0})
	write("empty", []byte(" \n"))
	if err := os.Symlink(root, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("docs/DESIGN.md", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"../demo/docs/DESIGN.md", "/etc/passwd", "missing", "docs", "large", "binary", "nul", "empty", "escape/scenarios/demo/docs/DESIGN.md", "pipe"} {
		if _, err := store.ReadDesignSource("demo", source); err == nil {
			t.Errorf("accepted invalid source %q", source)
		}
	}
	linked, err := store.ReadDesignSource("demo", "link")
	if err != nil || linked.ContentHash != updated.ContentHash {
		t.Fatal("confined source alias failed", err)
	}
	if source, err := store.ReadDesignSource("demo", ""); err != nil || source != nil {
		t.Fatal("optional source failed", err)
	}
}
