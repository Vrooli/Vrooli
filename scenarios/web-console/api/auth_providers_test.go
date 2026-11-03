package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// Web Console must verify scenario-authenticator tokens for its own resource
// audience so Agent Manager's service principal can pin stable-create owners.
func TestManagedAuthenticationIncludesScenarioAuthenticator(t *testing.T) {
	cfg := pointerToManagedAuthentication(authn.Config{})
	found := false
	for _, provider := range cfg.Providers {
		if provider != nil && provider.Source() == identity.SourceScenarioAuthenticator {
			found = true
		}
	}
	if !found {
		t.Fatal("web-console authentication chain has no scenario-authenticator provider")
	}
}

func TestDesktopAuthenticationRequiresDeclaredModeAndRuntimeToken(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	t.Setenv("BUNDLE_ROOT", root)
	t.Setenv("APP_DATA_DIR", root)
	if err := os.WriteFile(filepath.Join(root, "token"), []byte("private-fixture-session"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"personal_local", "remote_vrooli"} {
		if err := os.WriteFile(filepath.Join(root, "bundle.json"), []byte(`{"ipc":{"auth_token_path":"token"},"authentication":{"mode":"`+mode+`"}}`), 0600); err != nil {
			t.Fatal(err)
		}
		config := pointerToManagedAuthentication(authn.Config{})
		found := false
		for _, provider := range config.Providers {
			if provider.Source() != identity.SourcePersonalLocal {
				continue
			}
			found = true
			request := httptest.NewRequest("POST", "http://127.0.0.1:20000/", nil)
			request.RemoteAddr = "127.0.0.1:12345"
			if _, err := provider.VerifyRequest(context.Background(), request); err == nil {
				t.Fatal("loopback alone authenticated")
			}
			request.Header.Set("Authorization", "Bearer independent-account-credential")
			request.Header.Set("X-Vrooli-Local-Session", "private-fixture-session")
			principal, err := provider.VerifyRequest(context.Background(), request)
			if err != nil || !principal.IsHuman() {
				t.Fatalf("runtime session identity: %+v %v", principal, err)
			}
		}
		if found != (mode == "personal_local") {
			t.Fatalf("mode %s personal-local provider present=%t", mode, found)
		}
	}
}
