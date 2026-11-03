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
	if body["rank"] != "cpu" {
		t.Errorf("rank = %v, want cpu", body["rank"])
	}
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

func TestGetProcessTimelineAcceptsRSSRank(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithProcessTimeline(nil)
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())
	w := httptest.NewRecorder()
	handler.HandleGetProcessTimeline(w, httptest.NewRequest(http.MethodGet, "/api/v1/forensics/processes?rank=rss", nil))
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["rank"] != "rss" {
		t.Fatalf("rank = %v, want rss", body["rank"])
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

func TestGetProcessTimeline_ExplicitRange(t *testing.T) {
	start := time.Date(2026, 11, 3, 4, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Second)
	mock := handlermocks.NewMonitorQuerier().WithProcessTimeline([]repository.ProcessTimelineEntry{
		{Owner: "device-control", Comm: "device-control-api", SampleCount: 3, FirstSeen: start, LastSeen: end.Add(-5 * time.Second)},
	})
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())
	path := "/api/v1/forensics/processes?start_utc=" + start.Format(time.RFC3339) + "&end_utc=" + end.Format(time.RFC3339) + "&owner=device-control&rank=rss"
	w := httptest.NewRecorder()
	handler.HandleGetProcessTimeline(w, httptest.NewRequest(http.MethodGet, path, nil))

	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
	gotStart, gotEnd, called := mock.ProcessTimelineRange()
	if !called || !gotStart.Equal(start) || !gotEnd.Equal(end) {
		t.Fatalf("explicit range not forwarded exactly: called=%v start=%s end=%s", called, gotStart, gotEnd)
	}
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["start_utc"] != start.Format(time.RFC3339Nano) || body["end_utc"] != end.Format(time.RFC3339Nano) {
		t.Fatalf("response range = %v to %v, want %s to %s", body["start_utc"], body["end_utc"], start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano))
	}
	if body["window_seconds"].(float64) != 30 || body["rank"] != "rss" {
		t.Fatalf("range metadata = window %v rank %v, want 30 rss", body["window_seconds"], body["rank"])
	}
}

func TestGetProcessTimeline_RejectsInvalidExplicitRanges(t *testing.T) {
	for _, tc := range []struct {
		name  string
		query string
	}{
		{name: "missing end", query: "?start_utc=2026-11-03T04:00:00Z"},
		{name: "invalid timestamp", query: "?start_utc=bad&end_utc=2026-11-03T04:00:30Z"},
		{name: "empty interval", query: "?start_utc=2026-11-03T04:00:00Z&end_utc=2026-11-03T04:00:00Z"},
		{name: "over 24 hours", query: "?start_utc=2026-11-02T04:00:00Z&end_utc=2026-11-03T04:00:01Z"},
		{name: "mixed window", query: "?start_utc=2026-11-03T04:00:00Z&end_utc=2026-11-03T04:00:30Z&window=1m"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := handlermocks.NewMonitorQuerier()
			handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())
			w := httptest.NewRecorder()
			handler.HandleGetProcessTimeline(w, httptest.NewRequest(http.MethodGet, "/api/v1/forensics/processes"+tc.query, nil))
			_, _, called := mock.ProcessTimelineRange()
			if w.Code != http.StatusBadRequest || called {
				t.Fatalf("status=%d range_called=%v, want 400 and no query", w.Code, called)
			}
		})
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
