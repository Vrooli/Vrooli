package smoketest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMonetizationProbeUsesJourneyTargetInsteadOfEnvironmentFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/internal/monetization/journey" || r.URL.Query().Get("operation") != "signin_shared_session" {
			t.Fatalf("unexpected probe request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"observed":"session=shared","route":"private","app_key":"web-console"}`))
	}))
	defer server.Close()
	t.Setenv("VROOLI_VALIDATION_RENDERER_URL", "http://127.0.0.1:1")
	probe := monetizationJourneyAPI{target: JourneyTarget{APIURL: server.URL}}
	got, err := probe.Probe(context.Background(), "signin_shared_session")
	if err != nil || got.Observed != "session=shared" {
		t.Fatalf("probe = %#v, err=%v", got, err)
	}
}

func TestMonetizationProbeRefusesMissingJourneyTarget(t *testing.T) {
	t.Setenv("S2D_MONETIZATION_JOURNEY_URL", "http://127.0.0.1:1")
	_, err := (monetizationJourneyAPI{}).Probe(context.Background(), "signin_shared_session")
	if err == nil || !strings.Contains(err.Error(), "renderer URL is unavailable") {
		t.Fatalf("error = %v, want missing target error", err)
	}
	if _, ok := os.LookupEnv("S2D_MONETIZATION_JOURNEY_URL"); !ok {
		t.Fatal("test setup did not set environment fallback")
	}
}

func TestJourneyTargetRecordsIsolatedInstanceSource(t *testing.T) {
	target := JourneyTarget{RendererURL: "http://127.0.0.1:24100", APIURL: "http://127.0.0.1:19100", Instance: "web-console@desktop-smoke", Source: "isolated_instance"}
	if target.Source != "isolated_instance" || target.Instance == "" {
		t.Fatalf("target = %#v", target)
	}
}

func TestStartIsolatedSmokeInstanceUsesRequestContextForTimeout(t *testing.T) {
	if client := newControlPlaneHTTPClient(); client.Timeout != 0 {
		t.Fatalf("control-plane client timeout = %s, want request-context-owned timeout", client.Timeout)
	}
	if isolatedSmokeStartTimeoutSeconds != 300 {
		t.Fatalf("isolated start timeout = %d seconds, want bounded five-minute setup budget", isolatedSmokeStartTimeoutSeconds)
	}
}

func TestLoopbackJourneyAPIProbesCommunicationRouteAndChecksIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/internal/monetization/journey" || r.URL.Query().Get("operation") != "provider_observation" {
			t.Fatalf("unexpected probe request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"observed":"provider=tier1-local-vrooli;route=scenario-api-proxy","route":"/api/v1/internal/monetization/journey","app_key":"web-console"}`))
	}))
	defer server.Close()

	api := loopbackJourneyAPI{target: JourneyTarget{APIURL: server.URL}, expectedAppKey: "web-console"}
	result, err := api.Probe(context.Background(), "provider_observation")
	if err != nil {
		t.Fatalf("probe error = %v", err)
	}
	if result.AppKey != "web-console" || result.Observed == "" {
		t.Fatalf("probe result = %+v", result)
	}
}

func TestLoopbackJourneyAPIRejectsCommunicationIdentityMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"observed":"ok","app_key":"other-app"}`))
	}))
	defer server.Close()

	api := loopbackJourneyAPI{target: JourneyTarget{APIURL: server.URL}, expectedAppKey: "web-console"}
	if _, err := api.Probe(context.Background(), "communication_operation"); err == nil || !strings.Contains(err.Error(), "probe_app_mismatch") {
		t.Fatalf("error = %v, want probe_app_mismatch", err)
	}
}

func TestLoopbackJourneyAPITerminalFixtureChecksIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "unidentified", body: `{"observed":"terminal_output=desktop-terminal-fixture"}`, want: "probe_app_unidentified"},
		{name: "mismatch", body: `{"observed":"terminal_output=desktop-terminal-fixture","app_key":"other-app"}`, want: "probe_app_mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			api := loopbackJourneyAPI{target: JourneyTarget{APIURL: server.URL}, expectedAppKey: "web-console"}
			if _, err := api.Probe(context.Background(), "terminal_fixture"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestResolveBundledTargetReadsPrivateRuntimeEndpoints(t *testing.T) {
	id := "bundled-target-test"
	path := bundledTargetPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	if err := os.WriteFile(path, []byte(`{"renderer_url":"http://127.0.0.1:24001/","api_url":"http://127.0.0.1:24002"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err := resolveBundledTarget(context.Background(), id)
	if err != nil {
		t.Fatalf("resolve bundled target: %v", err)
	}
	if target.Source != "bundled_private" || target.APIURL != "http://127.0.0.1:24002" {
		t.Fatalf("target = %#v", target)
	}
}
