package runtimepaths

import (
	"path/filepath"
	"testing"
)

func TestDataPathRejectsForeignLifecycleNamespace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "web-console")
	got, err := DataPath("records")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "data", "vrooli", "swarm-manager", "records")
	if got != want {
		t.Fatalf("DataPath foreign namespace = %q, want %q", got, want)
	}
}

func TestDataPathPreservesOwnVariantNamespace(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_STORAGE_ROOT", root)
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "swarm-manager_shadow")
	got, err := DataPath("records")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "data", "vrooli", "swarm-manager_shadow", "records")
	if got != want {
		t.Fatalf("DataPath own namespace = %q, want %q", got, want)
	}
}
