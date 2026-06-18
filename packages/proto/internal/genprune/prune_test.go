package genprune

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneBeforeGenerateRemovesStaleOutputsAndIsIdempotent(t *testing.T) {
	protoRoot := t.TempDir()
	writePruneFixture(t, filepath.Join(protoRoot, "gen", "go", "demo", "stale.pb.go"))
	writePruneFixture(t, filepath.Join(protoRoot, "gen", "python", "demo", "stale_pb2.py"))
	writePruneFixture(t, filepath.Join(protoRoot, "gen", "typescript", "demo", "stale_pb.ts"))
	writePruneFixture(t, filepath.Join(protoRoot, "gen", "typescript", "js", "demo", "stale_pb.js"))
	writePruneFixture(t, filepath.Join(protoRoot, "gen", "typescript", "package.json"))

	if err := PruneBeforeGenerate(protoRoot); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, filepath.Join(protoRoot, "gen", "go"))
	assertMissing(t, filepath.Join(protoRoot, "gen", "python"))
	assertMissing(t, filepath.Join(protoRoot, "gen", "typescript", "demo"))
	assertMissing(t, filepath.Join(protoRoot, "gen", "typescript", "js"))
	assertExists(t, filepath.Join(protoRoot, "gen", "typescript", "package.json"))

	if err := PruneBeforeGenerate(protoRoot); err != nil {
		t.Fatalf("second prune should be idempotent: %v", err)
	}
	assertExists(t, filepath.Join(protoRoot, "gen", "typescript", "package.json"))
}

func writePruneFixture(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat err=%v", path, err)
	}
}
