package projectstate

import (
	"os"
	"testing"
)

func TestHasSetupCompleteUsesCanonicalMarkerOnly(t *testing.T) {
	root := t.TempDir()
	if HasSetupComplete(root) {
		t.Fatal("expected missing setup marker")
	}
	if err := os.MkdirAll(SetupStateDir(root), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(SetupCompletePath(root), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write setup marker: %v", err)
	}
	if !HasSetupComplete(root) {
		t.Fatal("expected canonical setup marker to be detected")
	}
}

func TestHasResourcePopulatedUsesCanonicalMarkerOnly(t *testing.T) {
	root := t.TempDir()
	if HasResourcePopulated(root, "postgres") {
		t.Fatal("expected missing resource marker")
	}
	if err := os.MkdirAll(SetupStateDir(root), 0o755); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(ResourcePopulatedPath(root, "postgres"), []byte("ok\n"), 0o644); err != nil {
		t.Fatalf("write resource marker: %v", err)
	}
	if !HasResourcePopulated(root, "postgres") {
		t.Fatal("expected canonical resource marker to be detected")
	}
}
