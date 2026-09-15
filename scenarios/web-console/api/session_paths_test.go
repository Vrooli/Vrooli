package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSessionStateRootScopesNonLiveVariant(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	t.Setenv("VROOLI_VARIANT", "desktop-smoke")

	want := filepath.Join(root, "variants", "desktop-smoke")
	if got := resolveSessionStateRoot(); got != want {
		t.Fatalf("variant session root = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("variant session root was not created: %v", err)
	}
}

func TestResolveSessionStateRootKeepsLiveRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sessions")
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	t.Setenv("VROOLI_VARIANT", "live")

	if got := resolveSessionStateRoot(); got != root {
		t.Fatalf("live session root = %q, want %q", got, root)
	}
}
