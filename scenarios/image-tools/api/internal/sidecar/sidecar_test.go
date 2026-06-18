package sidecar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialize_WritesPackage(t *testing.T) {
	root := t.TempDir()
	got, err := Materialize(root)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if got != root {
		t.Fatalf("expected PYTHONPATH root %q, got %q", root, got)
	}
	for _, f := range []string{"__init__.py", "_common.py", "bg_removal.py", "denoise.py"} {
		p := filepath.Join(root, PackageName, f)
		info, statErr := os.Stat(p)
		if statErr != nil {
			t.Fatalf("expected %s materialized: %v", f, statErr)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", f)
		}
	}
}

func TestMaterialize_Idempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if _, err := Materialize(root); err != nil {
		t.Fatalf("second materialize should be idempotent: %v", err)
	}
}

func TestMaterialize_EmptyRootRejected(t *testing.T) {
	if _, err := Materialize(""); err == nil {
		t.Fatalf("expected error for empty root")
	}
}

func TestEnsureOnPath_PrependsAndDedupes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PYTHONPATH", "/some/existing")
	path, err := EnsureOnPath(root)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	got := os.Getenv("PYTHONPATH")
	if !strings.HasPrefix(got, path+string(os.PathListSeparator)) {
		t.Fatalf("expected sidecar prepended, got %q", got)
	}
	if !strings.Contains(got, "/some/existing") {
		t.Fatalf("expected existing PYTHONPATH preserved, got %q", got)
	}
	// A second call must not duplicate the entry.
	if _, err := EnsureOnPath(root); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if c := strings.Count(os.Getenv("PYTHONPATH"), path); c != 1 {
		t.Fatalf("sidecar path should appear exactly once, got %d in %q", c, os.Getenv("PYTHONPATH"))
	}
}
