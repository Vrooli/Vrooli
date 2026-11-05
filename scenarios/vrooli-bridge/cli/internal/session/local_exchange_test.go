package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExchangeLocalUnavailableReturnsNoTokens(t *testing.T) {
	t.Setenv("VROOLI_AUTH_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	access, refresh, err := ExchangeLocal(context.Background())
	if err == nil || access != "" || refresh != "" {
		t.Fatal("unavailable exchange must return an error and no tokens")
	}
}

func TestDefaultAuthSocketIgnoresAmbientScenarioNamespace(t *testing.T) {
	t.Setenv("VROOLI_AUTH_SOCKET", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "web-console")
	want := filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
	if got := defaultAuthSocket(); got != want {
		t.Fatalf("defaultAuthSocket() = %q, want %q", got, want)
	}
}
