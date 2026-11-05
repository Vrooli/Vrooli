package cliutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopPortCallsRuntimeAndNeverCLIFallback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("APP_DATA_DIR", root)
	t.Setenv("BUNDLE_ROOT", root)
	t.Setenv("VROOLI_DESKTOP_MODE", "true")
	t.Setenv("PATH", t.TempDir())
	resetPortDetectorCache()
	defer resetPortDetectorCache()
	if err := os.MkdirAll(filepath.Join(root, "runtime"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{"bundle.json": `{"ipc":{"auth_token_path":"runtime/auth-token"}}`, "runtime/auth-token": "test-token"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing runtime authentication")
		}
		if r.URL.Path != "/scenario-port" || r.URL.Query().Get("scenario") != "audio-tools" {
			t.Errorf("wrong discovery request: %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"port":12345}`))
	}))
	u, _ := url.Parse(server.URL)
	if err := os.WriteFile(filepath.Join(root, "runtime/ipc_port"), []byte(u.Port()), 0600); err != nil {
		t.Fatal(err)
	}
	out := LookupScenarioPort(context.Background(), "audio-tools", "API_PORT", PortCachePolicy{})
	if out.Err != nil || out.Port != "12345" {
		t.Fatalf("native discovery: %+v", out)
	}
	server.Close()
	out = LookupScenarioPort(context.Background(), "audio-tools", "API_PORT", PortCachePolicy{})
	if out.Err == nil {
		t.Fatal("closed runtime should be unavailable")
	}
	if stats := PortLookupStats(); stats.CLIHits != 0 || stats.RegistryHits != 0 || stats.PeerHits != 0 {
		t.Fatalf("desktop escaped runtime authority: %+v", stats)
	}
}
