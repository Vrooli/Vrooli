package retention

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryWalkCursorResumesWithoutFollowingSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "payload"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "private"), []byte("protected"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	walk, err := OpenDirectoryWalk(root)
	if err != nil {
		t.Fatal(err)
	}
	defer walk.Close()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := walk.Next(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled observation = %v", err)
	}
	seen := map[string]bool{}
	for {
		path, _, err := walk.Next(context.Background())
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if seen[path] {
			t.Fatalf("visited %s twice", path)
		}
		seen[path] = true
	}
	if len(seen) != 4 || !seen[filepath.Join(root, "nested", "payload")] || !seen[filepath.Join(root, "link")] {
		t.Fatalf("cursor lost entries or traversed external content: %v", seen)
	}
}

func TestDirectoryWalkCursorRefusesReplacementDuringPause(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	walk, err := OpenDirectoryWalk(root)
	if err != nil {
		t.Fatal(err)
	}
	defer walk.Close()
	for {
		path, _, err := walk.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if path == nested {
			break
		}
	}
	if err := os.Rename(nested, nested+"-old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := walk.Next(context.Background()); err == nil || errors.Is(err, io.EOF) {
		t.Fatalf("replacement was treated as complete: %v", err)
	}
}
