package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAuthSocketIgnoresAmbientScenarioNamespace(t *testing.T) {
	t.Setenv("VROOLI_AUTH_SOCKET", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "web-console")
	want := filepath.Join(os.TempDir(), "vrooli-scenario-authenticator-scenario-authenticator.sock")
	if got := defaultAuthSocket(); got != want {
		t.Fatalf("defaultAuthSocket() = %q, want %q", got, want)
	}
}
