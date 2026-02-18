package handlers

import (
	"fmt"
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

	handler := NewMetricsHandler(&config.Config{}, mock)

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

	handler := NewMetricsHandler(&config.Config{}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current?fresh=true", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusOK)
}

func TestGetCurrentMetrics_Error(t *testing.T) {
	mock := handlermocks.NewMonitorQuerier().WithError(fmt.Errorf("collection failed"))
	handler := NewMetricsHandler(&config.Config{}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)
	testutil.AssertStatusCode(t, w.Code, http.StatusInternalServerError)
}

var _ MonitorQuerier = (*handlermocks.MonitorQuerier)(nil)
