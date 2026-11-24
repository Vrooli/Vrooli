package agenttranscript

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestWalkNativeFilesRespectsSymlinkPolicyAndContext(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "session.jsonl")
	if err := os.WriteFile(regular, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.jsonl")
	if err := os.Symlink(regular, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	collect := func(options NativeWalkOptions) []string {
		t.Helper()
		var paths []string
		if err := WalkNativeFiles(context.Background(), root, options, func(path string, _ fs.DirEntry, _ os.FileInfo) error {
			paths = append(paths, filepath.Base(path))
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return paths
	}
	if got := collect(NativeWalkOptions{}); len(got) != 1 || got[0] != "session.jsonl" {
		t.Fatalf("default walk paths = %v, want only regular file", got)
	}
	if got := collect(NativeWalkOptions{VisitNonRegularEntries: true}); len(got) != 2 {
		t.Fatalf("symlink-enabled walk paths = %v, want regular and symlink files", got)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	err := WalkNativeFiles(canceled, root, NativeWalkOptions{}, func(string, fs.DirEntry, os.FileInfo) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled walk error = %v, want context.Canceled", err)
	}
}

func TestWalkNativeFilesAllowsCallerSpecificRootErrorPolicy(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	err := WalkNativeFiles(context.Background(), root, NativeWalkOptions{
		Errors: NativeWalkSkipMissingRoot,
	}, func(string, fs.DirEntry, os.FileInfo) error {
		t.Fatal("missing root invoked file visitor")
		return nil
	})
	if err != nil {
		t.Fatalf("walk with caller's missing-root policy: %v", err)
	}
}
