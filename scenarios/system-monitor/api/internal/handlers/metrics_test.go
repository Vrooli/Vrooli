package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"system-monitor-api/internal/config"
	handlermocks "system-monitor-api/internal/handlers/mocks"
	"system-monitor-api/internal/models"
	"system-monitor-api/internal/testutil"
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

	handler.GetCurrentMetrics(w, req)
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

	handler.GetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetCurrentMetrics_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("collection failed"))
	handler := NewMetricsHandler(&config.Config{}, mock, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)
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
	handler.GetCurrentMetrics(w, req)
}

var _ MonitorQuerier = (*handlermocks.MonitorQuerier)(nil)
