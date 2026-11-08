package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func startupHealthItem(target string) scenario.Scenario {
	return scenario.Scenario{Slug: "alpha", Manifest: scenario.ServiceManifest{Lifecycle: scenario.Lifecycle{Health: &scenario.HealthConfig{
		Timeout: 25, Interval: 1,
		Checks: []scenario.HealthCheck{{Name: "api", Type: "http", Target: target, Critical: true}},
	}}}}
}

func TestStartupHealthWaitsForReadinessAndPersistsExactAcceptedSample(t *testing.T) {
	var probes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ready := probes.Add(1) >= 3
		_ = json.NewEncoder(w).Encode(apihealth.Response{Status: apihealth.StatusHealthy, Service: "alpha", Timestamp: time.Now().Format(time.RFC3339), Readiness: ready})
	}))
	defer server.Close()
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}
	item := startupHealthItem(server.URL)
	status, snapshot, err := runner.waitForHealthSnapshot(t.Context(), item, nil)
	if err != nil || status != scenarioruntime.HealthStatusHealthy || snapshot.Readiness == nil || !*snapshot.Readiness || probes.Load() != 3 {
		t.Fatalf("startup accepted status=%s snapshot=%+v err=%v probes=%d", status, snapshot, err, probes.Load())
	}
	store := openWaitTestStore(t, t.TempDir())
	instance, err := store.CreateInstance(t.Context(), scenarioruntime.Instance{Scenario: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	session := runtimeRegistrySession{enabled: true, store: store, instance: instance}
	if err := session.recordHealth(t.Context(), item, snapshot); err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetHealthSnapshot(t.Context(), instance.InstanceID)
	if err != nil || stored.Readiness == nil || !*stored.Readiness || stored.ResponseJSON != snapshot.ResponseJSON || probes.Load() != 3 {
		t.Fatalf("recording changed accepted observation: %+v, %v, probes=%d", stored, err, probes.Load())
	}
}

func TestStartupHealthCannotAcceptDegradedButNotReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","service":"alpha","timestamp":"2026-11-06T20:10:35Z","readiness":false,"functional":{"healthy":false,"reason":"startup initializing: in_flight_runs"}}`))
	}))
	defer server.Close()
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}
	_, snapshot, err := runner.waitForHealthSnapshot(t.Context(), startupHealthItem(server.URL), nil)
	if err == nil || snapshot.Readiness == nil || *snapshot.Readiness || snapshot.Error == "" {
		t.Fatalf("not-ready startup accepted: %+v, %v", snapshot, err)
	}
}

func TestStartupHealthCancellationCannotBecomeDegradedSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(apihealth.Response{Status: apihealth.StatusDegraded, Service: "alpha", Timestamp: time.Now().Format(time.RFC3339), Readiness: true})
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: func(d time.Duration) { clock.Sleep(d); cancel() }}}
	if _, _, err := runner.waitForHealthSnapshot(ctx, startupHealthItem(server.URL), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled startup = %v", err)
	}
}

// Reuse can preserve availability without inventing an image-identity witness.
func TestServingReuseSeparatesLegacyAvailabilityFromStartupImageProof(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		serving    bool
	}{
		{"legacy HTML", "<html>ready</html>", true},
		{"partial healthy", `{"status":"healthy","readiness":true}`, true},
		{"partial not ready", `{"status":"healthy","readiness":false}`, false},
		{"partial unhealthy", `{"status":"unhealthy"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			clock := newFakeAwaitClock()
			runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}
			item := startupHealthItem(server.URL)
			view := registryRuntimeView{Authoritative: true, Instance: scenarioruntime.Instance{BuildIdentity: "sha256:prior-serving-image"}}
			if got := runner.isRegistryRuntimeServingHealthy(item, view); got != tc.serving {
				t.Fatalf("serving=%t want=%t", got, tc.serving)
			}
			// The exact same body cannot qualify a freshly launched expected image.
			_, snapshot, err := runner.waitForHealthSnapshot(t.Context(), item, map[string]string{buildIdentityEnv: view.Instance.BuildIdentity})
			if err == nil || snapshot.BuildIdentity != "" {
				t.Fatalf("legacy endpoint proved image: snapshot=%+v err=%v", snapshot, err)
			}
		})
	}
}

func TestColdLegacyAvailabilityCannotQualifyRegistryArtifactIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>ready</html>")) }))
	defer server.Close()
	clock := newFakeAwaitClock()
	runner := &Runner{deps: lifecycleDeps{now: clock.Now, sleep: clock.Sleep}}
	item := startupHealthItem(server.URL)
	// The compatibility API is availability-only when no image was requested.
	status, observation, err := runner.waitForHealthSnapshot(t.Context(), item, nil)
	if err != nil || status != scenarioruntime.HealthStatusHealthy || observation.BuildIdentity != "" {
		t.Fatalf("legacy availability=%s %+v %v", status, observation, err)
	}
	store := openWaitTestStore(t, t.TempDir())
	instance, err := store.CreateInstance(t.Context(), scenarioruntime.Instance{Scenario: item.Slug, BuildIdentity: "sha256:cold-artifact"})
	if err != nil {
		t.Fatal(err)
	}
	session := runtimeRegistrySession{enabled: true, store: store, instance: instance}
	if err := session.recordHealth(t.Context(), item, observation); err == nil {
		t.Fatal("unproved cold image accepted")
	}
	stored, err := store.GetHealthSnapshot(t.Context(), instance.InstanceID)
	if err != nil || stored.BuildIdentity != "" || stored.Status != scenarioruntime.HealthStatusHealthy {
		t.Fatalf("availability/identity observation fabricated: %+v %v", stored, err)
	}
}
