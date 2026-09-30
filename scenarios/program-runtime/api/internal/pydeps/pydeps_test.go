package pydeps

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeIsStableAndContainsEmptyLock(t *testing.T) {
	dir := t.TempDir()
	path, err := Materialize(dir)
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 || filepath.Base(path) != LockName {
		t.Fatalf("invalid lock materialization: %q", path)
	}
	secondPath, err := Materialize(dir)
	if err != nil || secondPath != path {
		t.Fatalf("second materialization = %q, %v", secondPath, err)
	}
	second, err := os.ReadFile(path)
	if err != nil || string(first) != string(second) {
		t.Fatalf("materialized lock changed: %v", err)
	}
}
