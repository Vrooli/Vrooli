package discovery

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestResolveScenarioURLDoesNotReusePriorVariantAfterEnvironmentLoss(t *testing.T) {
	const savedDraft = "the operator's saved draft"
	makeOwner := func(instance string) *httptest.Server {
		t.Helper()
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/draft" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("X-Instance", instance)
			_, _ = io.WriteString(w, savedDraft)
		}))
	}
	shadowOwner := makeOwner("shadow")
	defer shadowOwner.Close()
	liveOwner := makeOwner("live")
	defer liveOwner.Close()
	portOf := func(server *httptest.Server) string {
		t.Helper()
		u, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		return u.Port()
	}
	shadowPort, livePort := portOf(shadowOwner), portOf(liveOwner)
	calls := 0
	runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls++
		if len(args) != 4 || args[0] != "scenario" || args[1] != "port" || args[3] != "API_PORT" {
			return nil, errors.New("unexpected lookup arguments")
		}
		switch {
		case args[2] == "agent-manager@shadow" && calls == 1:
			return []byte(shadowPort), nil
		case args[2] == "agent-manager":
			if calls == 2 {
				return []byte(`{"success":false,"error_code":"scenario_not_running","error":"provider stopped"}`), errors.New("provider stopped")
			}
			if calls == 3 {
				return []byte(livePort), nil
			}
		}
		return nil, errors.New("unexpected target or lookup count")
	}
	now := time.Date(2026, 11, 21, 16, 0, 0, 0, time.UTC)
	resolver := NewResolver(ResolverConfig{CommandRunner: runner, Host: "127.0.0.1", CacheTTL: time.Minute, Now: func() time.Time { return now }})
	t.Setenv(cliutil.EnvShadowScenarios, "agent-manager")
	shadowURL, err := resolver.ResolveScenarioURL(context.Background(), "agent-manager", "API_PORT")
	if err != nil {
		t.Fatal(err)
	}
	assertSavedDraft := func(baseURL, instance string) {
		t.Helper()
		resp, err := http.Get(baseURL + "/draft")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if got := resp.Header.Get("X-Instance"); got != instance || string(body) != savedDraft {
			t.Fatalf("route instance=%q body=%q, want %q and retained draft", got, body, instance)
		}
	}
	assertSavedDraft(shadowURL, "shadow")

	// Change the ambient runtime configuration. A transient loss for the new
	// target must not return the cached address of the previous environment.
	t.Setenv(cliutil.EnvShadowScenarios, "")
	if got, err := resolver.ResolveScenarioURL(context.Background(), "agent-manager", "API_PORT"); err == nil || got != "" || !IsScenarioNotRunning(err) {
		t.Fatalf("live target loss returned URL %q, error %v; want stopped-target failure", got, err)
	}
	now = now.Add(time.Second) // Let the bounded confirmed-stopped cache expire before the recovery attempt.
	liveURL, err := resolver.ResolveScenarioURL(context.Background(), "agent-manager", "API_PORT")
	if err != nil {
		t.Fatal(err)
	}
	assertSavedDraft(liveURL, "live")
	if calls != 3 {
		t.Fatalf("port lookups=%d, want shadow, lost live, recovered live", calls)
	}
	if liveURL != "http://127.0.0.1:"+livePort {
		t.Fatalf("recovered URL = %q, want live endpoint on port %s", liveURL, livePort)
	}
}
