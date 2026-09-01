package scenarioruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestHealthProbeNoChecksReportsNotConfigured(t *testing.T) {
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	snapshot := HealthProbe{Clock: clk}.Probe(context.Background(), HealthProbeInput{
		InstanceID:   "inst-alpha",
		Scenario:     "alpha",
		HealthConfig: &scenario.HealthConfig{},
	})

	if snapshot.Status != HealthStatusNotConfigured {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusNotConfigured)
	}
	if snapshot.CheckedAt == nil || !snapshot.CheckedAt.Equal(clk.Now()) {
		t.Fatalf("snapshot.CheckedAt = %#v, want %s", snapshot.CheckedAt, clk.Now())
	}
	if snapshot.Readiness != nil {
		t.Fatalf("snapshot.Readiness = %#v, want nil", snapshot.Readiness)
	}
}

func TestHealthProbeRecognizesStandardHealthResponse(t *testing.T) {
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q, want /health", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(apihealth.Response{
			Status:    apihealth.StatusDegraded,
			Service:   "alpha-api",
			Timestamp: clk.Now().Format(time.RFC3339),
			Readiness: true,
		})
	}))
	defer server.Close()

	snapshot := HealthProbe{Clock: clk}.Probe(context.Background(), HealthProbeInput{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
		HealthConfig: &scenario.HealthConfig{Checks: []scenario.HealthCheck{{
			Name:     "api",
			Type:     "http",
			Target:   server.URL + "/health",
			Critical: true,
		}}},
	})

	if snapshot.Status != HealthStatusDegraded {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusDegraded)
	}
	if snapshot.Readiness == nil || *snapshot.Readiness != true {
		t.Fatalf("snapshot.Readiness = %#v, want true", snapshot.Readiness)
	}
	if snapshot.SchemaValid == nil || *snapshot.SchemaValid != true {
		t.Fatalf("snapshot.SchemaValid = %#v, want true", snapshot.SchemaValid)
	}
	if !strings.Contains(snapshot.ResponseJSON, `"status":"degraded"`) {
		t.Fatalf("snapshot.ResponseJSON = %q, want standard health JSON", snapshot.ResponseJSON)
	}
	if !strings.Contains(snapshot.Error, `health check "api" reported degraded without a reason`) {
		t.Fatalf("snapshot.Error = %q, want a non-empty degraded reason", snapshot.Error)
	}
}

func TestHealthProbePreservesDegradedStatusWithProviderDetail(t *testing.T) {
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","service":"alpha-api","timestamp":"2026-05-08T12:00:00Z","readiness":true,"dependencies":{"providers":{"error":"speaker-verification"}}}`))
	}))
	defer server.Close()

	snapshot := HealthProbe{Clock: clk}.Probe(context.Background(), HealthProbeInput{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
		HealthConfig: &scenario.HealthConfig{Checks: []scenario.HealthCheck{{
			Name: "api", Type: "http", Target: server.URL, Critical: true,
		}}},
	})

	if snapshot.Status != HealthStatusDegraded {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusDegraded)
	}
	if snapshot.Error != "providers: speaker-verification" {
		t.Fatalf("snapshot.Error = %q, want provider detail", snapshot.Error)
	}
}

func TestHealthProbeInvalidSchemaIsDiagnosticMetadata(t *testing.T) {
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	snapshot := HealthProbe{Clock: clk}.Probe(context.Background(), HealthProbeInput{
		InstanceID: "inst-alpha",
		Scenario:   "alpha",
		HealthConfig: &scenario.HealthConfig{Checks: []scenario.HealthCheck{{
			Name:   "api",
			Type:   "http",
			Target: server.URL,
		}}},
	})

	if snapshot.Status != HealthStatusDegraded {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusDegraded)
	}
	if snapshot.SchemaValid == nil || *snapshot.SchemaValid != false {
		t.Fatalf("snapshot.SchemaValid = %#v, want false", snapshot.SchemaValid)
	}
	if !strings.Contains(snapshot.Error, "invalid health response schema") {
		t.Fatalf("snapshot.Error = %q, want invalid schema diagnostic", snapshot.Error)
	}
	if snapshot.ResponseJSON != `{"ok":true}` {
		t.Fatalf("snapshot.ResponseJSON = %q, want bounded response body", snapshot.ResponseJSON)
	}
}

func TestHealthProbeUnhealthySnapshotDoesNotDeleteInstance(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	instance, err := store.CreateLease(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"}, time.Minute)
	if err != nil {
		t.Fatalf("CreateLease() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"unhealthy"}`))
	}))
	defer server.Close()

	snapshot := HealthProbe{Clock: clk}.Probe(ctx, HealthProbeInput{
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		HealthConfig: &scenario.HealthConfig{Checks: []scenario.HealthCheck{{
			Name:     "api",
			Type:     "http",
			Target:   server.URL,
			Critical: true,
		}}},
	})
	if snapshot.Status != HealthStatusUnhealthy {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusUnhealthy)
	}
	if _, err := store.UpsertHealthSnapshot(ctx, snapshot); err != nil {
		t.Fatalf("UpsertHealthSnapshot() error = %v", err)
	}

	got, err := store.GetInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetInstance() error = %v", err)
	}
	if got.Status != StatusStarting {
		t.Fatalf("got.Status = %q, want lease status unchanged", got.Status)
	}
}
