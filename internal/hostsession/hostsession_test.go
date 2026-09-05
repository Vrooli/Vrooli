package hostsession

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultProviderReturnsStableHostSession(t *testing.T) {
	home := t.TempDir()
	first, err := DefaultProvider{}.Current(context.Background(), home)
	if err != nil {
		t.Fatalf("Current(first): %v", err)
	}
	second, err := DefaultProvider{}.Current(context.Background(), home)
	if err != nil {
		t.Fatalf("Current(second): %v", err)
	}
	if first.BootID == "" || first.SessionID == "" || first.Source == "" {
		t.Fatalf("first snapshot missing identity: %#v", first)
	}
	if first.BootID != second.BootID || first.SessionID != second.SessionID {
		t.Fatalf("snapshots are not stable: first=%#v second=%#v", first, second)
	}
}

func TestPersistentFallbackSessionReusesExistingToken(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "state", sessionFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("existing-token\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	token, err := persistentFallbackSession(home)
	if err != nil {
		t.Fatalf("persistentFallbackSession: %v", err)
	}
	if token != "existing-token" {
		t.Fatalf("token = %q, want existing-token", token)
	}
}
