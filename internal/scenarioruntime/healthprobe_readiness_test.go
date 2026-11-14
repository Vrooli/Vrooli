package scenarioruntime

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/internal/scenario"
)

func TestHealthProbeAggregatesReadinessAcrossAllEndpointsAndLegacyResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ready":
			_, _ = w.Write([]byte(`{"status":"healthy","service":"alpha","timestamp":"2026-11-06T20:10:35Z","readiness":true}`))
		case "/starting":
			_, _ = w.Write([]byte(`{"status":"healthy","service":"alpha","timestamp":"2026-11-06T20:10:35Z","readiness":false}`))
		default:
			_, _ = w.Write([]byte("legacy endpoint"))
		}
	}))
	defer server.Close()
	for _, order := range [][]string{{"/ready", "/starting"}, {"/starting", "/ready"}, {"/legacy", "/starting"}, {"/starting", "/legacy"}} {
		var checks []scenario.HealthCheck
		for _, target := range order {
			checks = append(checks, scenario.HealthCheck{Type: "http", Target: server.URL + target, Critical: true})
		}
		snapshot := (HealthProbe{AllowLegacyHTTP: true}).Probe(t.Context(), HealthProbeInput{HealthConfig: &scenario.HealthConfig{Checks: checks}})
		if snapshot.Readiness == nil || *snapshot.Readiness {
			t.Fatalf("order %v erased a not-ready endpoint: %+v", order, snapshot)
		}
	}
}

func TestHealthProbeLegacyMetadataCannotEraseExplicitNegativeFields(t *testing.T) {
	for _, body := range []string{
		`{"status":"healthy","readiness":false}`,
		`{"status":"unhealthy"}`,
		`{"readiness":false}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			snapshot := (HealthProbe{AllowLegacyHTTP: true}).Probe(t.Context(), HealthProbeInput{HealthConfig: &scenario.HealthConfig{
				Checks: []scenario.HealthCheck{{Type: "http", Target: server.URL, Critical: true}},
			}})
			if snapshot.Status != HealthStatusUnhealthy && (snapshot.Readiness == nil || *snapshot.Readiness) {
				t.Fatalf("legacy metadata erased negative fields: %+v", snapshot)
			}
		})
	}
}

func TestHealthProbeLegacyAvailabilityCannotProveExpectedBuildIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/standard" {
			_, _ = w.Write([]byte(`{"status":"healthy","service":"alpha","timestamp":"2026-11-06T20:10:35Z","readiness":true,"build_identity":"candidate"}`))
			return
		}
		_, _ = w.Write([]byte(`<html>legacy healthy transport</html>`))
	}))
	defer server.Close()
	legacy := scenario.HealthCheck{Type: "http", Target: server.URL + "/legacy", Critical: true}
	standard := scenario.HealthCheck{Type: "http", Target: server.URL + "/standard", Critical: true}
	for _, tc := range []struct {
		name     string
		checks   []scenario.HealthCheck
		expected string
		want     string
	}{
		{name: "legacy availability", checks: []scenario.HealthCheck{legacy}, want: HealthStatusHealthy},
		{name: "legacy cannot prove build", checks: []scenario.HealthCheck{legacy}, expected: "candidate", want: HealthStatusUnhealthy},
		{name: "standard api proves mixed availability", checks: []scenario.HealthCheck{legacy, standard}, expected: "candidate", want: HealthStatusHealthy},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := (HealthProbe{AllowLegacyHTTP: true}).Probe(t.Context(), HealthProbeInput{HealthConfig: &scenario.HealthConfig{Checks: tc.checks}, ExpectedBuildIdentity: tc.expected})
			if snapshot.Status != tc.want {
				t.Fatalf("status=%s want=%s; snapshot=%+v", snapshot.Status, tc.want, snapshot)
			}
		})
	}
}
