package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExchangeLocalUnavailableReturnsNoSession(t *testing.T) {
	t.Setenv("VROOLI_AUTH_SOCKET", filepath.Join(t.TempDir(), "absent.sock"))
	response, err := exchangeLocal(context.Background())
	if err == nil || response != nil {
		t.Fatal("unavailable exchange must return an error and no session")
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
