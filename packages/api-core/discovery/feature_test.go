package discovery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFeatureResolutionUsesAuthenticatedRuntimePerCall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("secret-test-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN_FILE", path)
	calls := 0
	runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls++
		if req.URL.Path != "/features" || req.URL.Query().Get("scenario") != "agent-manager" || req.URL.Query().Get("feature") != "usage" || req.Header.Get("Authorization") != "Bearer secret-test-token" {
			t.Errorf("unexpected feature request: %s", req.URL)
		}
		if calls == 3 {
			http.Error(w, "removed", 404)
			return
		}
		fmt.Fprintf(w, `{"url":"http://127.0.0.1:%d","source":"component","component_id":"agent-manager-usage"}`, 3000+calls)
	}))
	defer runtime.Close()
	t.Setenv("VROOLI_RUNTIME_URL", runtime.URL)
	resolver := NewResolver(ResolverConfig{StaticBaseURL: "http://127.0.0.1:9999"})
	for i := 1; i <= 2; i++ {
		got, err := resolver.ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage")
		if err != nil || got != fmt.Sprintf("http://127.0.0.1:%d", 3000+i) {
			t.Fatalf("lookup %d: %s %v", i, got, err)
		}
	}
	if got, err := resolver.ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage"); err == nil || got != "" {
		t.Fatalf("removed feature used bare fallback: %q %v", got, err)
	}
}

func TestFeatureResolutionOutsideDesktopUsesFullScenario(t *testing.T) {
	t.Setenv("VROOLI_RUNTIME_URL", "")
	resolver := NewResolver(ResolverConfig{StaticBaseURL: "http://localhost:9999"})
	got, err := resolver.ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage")
	if err != nil || got != "http://localhost:9999" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := resolver.ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", " "); err == nil {
		t.Fatal("empty feature accepted")
	}
}

func TestFeatureResolutionFailsClosedForInvalidRuntimeAndResponse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("test-token"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN_FILE", path)
	for _, response := range []string{`{`, `{ "url": "https://example.com" }`, `{ "url":"http://user:password@localhost:34" }`, `{ "url":"http://127.0.0.1:3000?token=secret" }`, `{ "url": "" }`} {
		t.Run(response, func(t *testing.T) {
			runtime := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
			defer runtime.Close()
			t.Setenv("VROOLI_RUNTIME_URL", runtime.URL)
			if _, err := ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage"); err == nil {
				t.Fatal("invalid response accepted")
			}
		})
	}
	t.Setenv("VROOLI_RUNTIME_URL", "http://example.com:3000")
	if _, err := ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage"); err == nil {
		t.Fatal("nonlocal runtime accepted")
	}
	t.Setenv("VROOLI_RUNTIME_URL", "http://127.0.0.1:3000")
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))
	_, err := ResolveScenarioFeatureURLDefault(context.Background(), "agent-manager", "usage")
	if err == nil || !strings.Contains(err.Error(), "credential unavailable") {
		t.Fatalf("missing credential: %v", err)
	}
}
