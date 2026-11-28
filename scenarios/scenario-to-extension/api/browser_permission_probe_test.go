package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureBrowserPermissionProbeIsDeterministicAndDoesNotReplaceForeignAssets(t *testing.T) {
	root := t.TempDir()
	if err := ensureBrowserPermissionProbe(root); err != nil {
		t.Fatal(err)
	}
	if err := ensureBrowserPermissionProbe(root); err != nil {
		t.Fatalf("repeat materialization failed: %v", err)
	}
	path := filepath.Join(root, "__vrooli_bas", "permission-report.js")
	if err := os.WriteFile(path, []byte("foreign artifact code"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureBrowserPermissionProbe(root); err == nil {
		t.Fatal("reserved report path replaced a pre-existing artifact-owned script")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "foreign artifact code" {
		t.Fatalf("foreign asset was modified: got=%q err=%v", got, err)
	}
}
