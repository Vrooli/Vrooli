package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
	handlermocks "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/handlers/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

func TestGetCurrentMetrics_Success(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:       42.5,
			MemoryUsage:    65.3,
			TCPConnections: 120,
			Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)

	// Verify JSON contains expected fields.
	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	if body["cpuUsage"] == nil && body["cpu_usage"] == nil {
		t.Error("expected cpuUsage or cpu_usage in response")
	}
}

func TestGetCurrentMetrics_Fresh(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:    10.0,
			MemoryUsage: 20.0,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithFreshMetrics(&models.MetricsResponse{
			CPUUsage:    55.5,
			MemoryUsage: 77.7,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current?fresh=true", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetCurrentMetrics_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("collection failed"))
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.HandleGetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusInternalServerError)
}

// brokenResponseWriter simulates a ResponseWriter whose Write method fails.
type brokenResponseWriter struct {
	header http.Header
}

func newBrokenResponseWriter() *brokenResponseWriter {
	return &brokenResponseWriter{header: make(http.Header)}
}

func (w *brokenResponseWriter) Header() http.Header         { return w.header }
func (w *brokenResponseWriter) WriteHeader(_ int)           {}
func (w *brokenResponseWriter) Write(_ []byte) (int, error) { return 0, fmt.Errorf("broken pipe") }

func TestGetCurrentMetrics_WriteError_NoPanic(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithCurrentMetrics(&models.MetricsResponse{
			CPUUsage:    42.5,
			MemoryUsage: 65.3,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}).
		WithActive(true)

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := newBrokenResponseWriter()

	// This should not panic even though Write returns an error.
	handler.HandleGetCurrentMetrics(w, req)
}

func TestGetMetricsTimeline_Success(t *testing.T) {
	gpu := 45.0
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         120,
			SampleIntervalSeconds: 5,
			Samples: []models.MetricTimelineSample{
				{
					Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
					CPUUsage:       42.5,
					MemoryUsage:    65.3,
					TCPConnections: 120,
					GPUUsage:       &gpu,
				},
				{
					Timestamp:      time.Date(2026, 1, 1, 0, 0, 5, 0, time.UTC),
					CPUUsage:       44.1,
					MemoryUsage:    66.0,
					TCPConnections: 118,
				},
			},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=120", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)

	body := testutil.DecodeJSONBody[map[string]interface{}](t, w.Body.Bytes())
	samples, ok := body["samples"].([]interface{})
	if !ok {
		t.Fatal("expected samples array in response")
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(samples))
	}
}

func TestGetMetricsTimeline_EmptySamples(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         120,
			SampleIntervalSeconds: 5,
			Samples:               []models.MetricTimelineSample{},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetMetricsTimeline_CustomWindow(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().
		WithTimelineResponse(&models.MetricsTimelineResponse{
			WindowSeconds:         300,
			SampleIntervalSeconds: 10,
			Samples:               []models.MetricTimelineSample{},
		})

	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=300&interval=10", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetMetricsTimeline_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("timeline retrieval failed"))
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/timeline?window=120", nil)
	w := httptest.NewRecorder()

	handler.HandleGetMetricsTimeline(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusInternalServerError)
}

var _ MonitorQuerier = (*handlermocks.MonitorQuerier)(nil)
