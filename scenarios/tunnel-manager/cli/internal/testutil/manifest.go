// Package testutil contains helpers shared by CLI package tests.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ReadManifest loads the scenario CLI manifest from the package containing the
// caller, walking upward until the cli/ surface is found. Tests in nested
// domain packages can therefore share one fixture loader without duplicating
// fragile relative paths.
func ReadManifest(t testing.TB) []byte {
	t.Helper()
	_, caller, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("resolve test caller")
	}
	dir := filepath.Dir(caller)
	for {
		path := filepath.Join(dir, "manifest.json")
		if raw, err := os.ReadFile(path); err == nil {
			return raw
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("read CLI manifest from %s", caller)
	return nil
}
