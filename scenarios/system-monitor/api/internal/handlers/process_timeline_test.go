package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

func TestGetProcessTimeline_RankedWindow(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	mock := handlermocks.NewMonitorQuerier().WithProcessTimeline([]repository.ProcessTimelineEntry{
		{Owner: "security-health", Comm: "osv-scanner", PID: 42, CPUPct: 130.5, RSSKB: 2048, SampleCount: 7, FirstSeen: now, LastSeen: now.Add(time.Minute)},
		{Owner: "web-console", Comm: "node", Aggregated: true, CPUPct: 12.0, RSSKB: 9000, SampleCount: 20},
	})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/processes/timeline?window=5m&top=10", nil)
	w := httptest.NewRecorder()
	handler.HandleGetProcessTimeline(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)

	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["window_seconds"].(float64) != 300 {
		t.Errorf("window_seconds = %v, want 300", body["window_seconds"])
	}
	if body["count"].(float64) != 2 {
		t.Errorf("count = %v, want 2", body["count"])
	}
	entries, ok := body["entries"].([]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %v", body["entries"])
	}
	first := entries[0].(map[string]interface{})
	if first["owner"] != "security-health" {
		t.Errorf("first owner = %v, want security-health", first["owner"])
	}
	if first["cpu_pct"].(float64) < 130 {
		t.Errorf("first cpu_pct = %v, want >=130", first["cpu_pct"])
	}
	if first["first_seen"] == nil {
		t.Error("expected first_seen on raw entry")
	}
	second := entries[1].(map[string]interface{})
	if second["aggregated"] != true {
		t.Errorf("second entry should be aggregated, got %v", second["aggregated"])
	}
}

func TestGetProcessTimeline_BareSecondsWindow(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithProcessTimeline(nil)
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/processes/timeline?window=120", nil)
	w := httptest.NewRecorder()
	handler.HandleGetProcessTimeline(w, req)

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["window_seconds"].(float64) != 120 {
		t.Errorf("bare-seconds window_seconds = %v, want 120", body["window_seconds"])
	}
}

func TestGetProcessTimelineConnect_RankedWindow(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	mock := handlermocks.NewMonitorQuerier().WithProcessTimeline([]repository.ProcessTimelineEntry{
		{Owner: "security-health", Comm: "osv-scanner", PID: 42, CPUPct: 130.5, RSSKB: 2048, SampleCount: 7, FirstSeen: now, LastSeen: now.Add(time.Minute)},
	})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	resp, err := handler.GetProcessTimeline(context.Background(), connect.NewRequest(&metricspb.GetProcessTimelineRequest{
		WindowSeconds: ptrInt32(300),
		Owner:         "security-health",
		Top:           ptrInt32(10),
	}))
	if err != nil {
		t.Fatalf("GetProcessTimeline returned error: %v", err)
	}

	timeline := resp.Msg.GetTimeline()
	if timeline.GetWindowSeconds() != 300 {
		t.Fatalf("window_seconds = %d, want 300", timeline.GetWindowSeconds())
	}
	if timeline.GetOwner() != "security-health" {
		t.Fatalf("owner = %q, want security-health", timeline.GetOwner())
	}
	if timeline.GetCount() != 1 {
		t.Fatalf("count = %d, want 1", timeline.GetCount())
	}
	if got := timeline.GetEntries()[0].GetCpuPct(); got != 130.5 {
		t.Fatalf("cpu_pct = %v, want 130.5", got)
	}
	if timeline.GetEntries()[0].GetFirstSeen() == nil {
		t.Fatalf("first_seen missing")
	}
}

func ptrInt32(v int32) *int32 {
	return &v
}
