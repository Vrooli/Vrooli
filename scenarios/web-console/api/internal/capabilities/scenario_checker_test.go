package capabilities

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func scenarioHealthChecker(t *testing.T, code int, body string) ScenarioChecker {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("health path = %s", r.URL.Path)
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return ScenarioChecker{Slug: "audio-tools", ResolveURL: func(_ context.Context, slug string) (string, error) {
		if slug != "audio-tools" {
			t.Errorf("selected peer = %s", slug)
		}
		return server.URL, nil
	}}
}

func TestScenarioChecker_HealthWithoutVrooli(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, tc := range []struct {
		name   string
		code   int
		body   string
		status Status
		reason string
	}{
		{"healthy", 200, `{"status":"healthy"}`, StatusAvailable, ""},
		{"ok", 200, `{"status":"ok"}`, StatusAvailable, ""},
		{"degraded", 503, `{"status":"degraded","message":"tts provider down"}`, StatusUnavailable, "scenario_degraded"},
		{"degraded without detail", 200, `{"status":"degraded"}`, StatusUnavailable, "scenario_degraded"},
		{"unhealthy", 503, `{"status":"unhealthy"}`, StatusUnavailable, "scenario_health_not_ready"},
		{"unknown", 200, `{"status":"unexpected"}`, StatusUnavailable, "scenario_health_not_ready"},
		{"missing status", 200, `{}`, StatusUnavailable, "scenario_health_not_ready"},
		{"malformed", 200, `not json`, StatusUnknown, "scenario_health_malformed_json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checker := scenarioHealthChecker(t, tc.code, tc.body)
			got := checker.CheckResult(t.Context())
			if got.Status != tc.status || got.ReasonCode != tc.reason {
				t.Fatalf("result = %+v", got)
			}
			if got.ActionKind != "" || got.OperatorCommand != "" {
				t.Fatalf("health probe invented a lifecycle action: %+v", got)
			}
		})
	}
}

func TestScenarioChecker_UnavailableHasNoDeadAction(t *testing.T) {
	checker := ScenarioChecker{Slug: "audio-tools", ResolveURL: func(context.Context, string) (string, error) { return "", errors.New("peer absent") }}
	got := checker.CheckResult(t.Context())
	if got.Status != StatusUnavailable || got.ReasonCode != "scenario_not_reachable" || got.ActionKind != "" {
		t.Fatalf("result = %+v", got)
	}
}

func TestScenarioChecker_NoSlug(t *testing.T) {
	got := (&ScenarioChecker{}).CheckResult(t.Context())
	if got.Status != StatusUnavailable || got.ReasonCode != "scenario_slug_missing" {
		t.Fatalf("result = %+v", got)
	}
}

func TestScenarioChecker_BoundsDiscovery(t *testing.T) {
	checker := ScenarioChecker{Slug: "audio-tools", Timeout: time.Millisecond, ResolveURL: func(ctx context.Context, _ string) (string, error) { <-ctx.Done(); return "", ctx.Err() }}
	if got := checker.CheckResult(t.Context()); got.Status != StatusUnavailable {
		t.Fatalf("result = %+v", got)
	}
}
