package projectstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasSetupCompletePromotesLegacyMarker(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "data", ".setup-complete")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}

	if !HasSetupComplete(root) {
		t.Fatal("expected setup marker to be detected")
	}
	if _, err := os.Stat(SetupCompletePath(root)); err != nil {
		t.Fatalf("expected promoted marker at new path: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("expected legacy marker to be removed, err=%v", err)
	}
}

func TestHasResourcePopulatedPromotesLegacyMarker(t *testing.T) {
	root := t.TempDir()
	legacy := filepath.Join(root, "data", ".postgres-populated")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("legacy\n"), 0o644); err != nil {
		t.Fatalf("write legacy marker: %v", err)
	}

	if !HasResourcePopulated(root, "postgres") {
		t.Fatal("expected resource marker to be detected")
	}
	if _, err := os.Stat(ResourcePopulatedPath(root, "postgres")); err != nil {
		t.Fatalf("expected promoted resource marker at new path: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("expected legacy marker to be removed, err=%v", err)
	}
}
