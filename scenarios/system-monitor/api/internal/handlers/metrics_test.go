package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"system-monitor-api/internal/config"
	"system-monitor-api/internal/models"
)

// mockMonitorQuerier is a test double for MonitorQuerier.
type mockMonitorQuerier struct {
	metrics         *models.MetricsResponse
	freshMetrics    *models.MetricsResponse
	detailedMetrics *models.DetailedMetrics
	processData     *models.ProcessMonitorData
	infraData       *models.InfrastructureMonitorData
	active          bool
	err             error
}

func (m *mockMonitorQuerier) GetCurrentMetrics(_ context.Context) (*models.MetricsResponse, error) {
	return m.metrics, m.err
}

func (m *mockMonitorQuerier) GetCurrentMetricsFresh(_ context.Context) (*models.MetricsResponse, error) {
	if m.freshMetrics != nil {
		return m.freshMetrics, m.err
	}
	return m.metrics, m.err
}

func (m *mockMonitorQuerier) GetDetailedMetrics(_ context.Context) (*models.DetailedMetrics, error) {
	return m.detailedMetrics, m.err
}

func (m *mockMonitorQuerier) GetProcessMonitorData(_ context.Context) (*models.ProcessMonitorData, error) {
	return m.processData, m.err
}

func (m *mockMonitorQuerier) GetInfrastructureMonitorData(_ context.Context) (*models.InfrastructureMonitorData, error) {
	return m.infraData, m.err
}

func (m *mockMonitorQuerier) IsActive() bool { return m.active }

func TestGetCurrentMetrics_Success(t *testing.T) {
	mock := &mockMonitorQuerier{
		metrics: &models.MetricsResponse{
			CPUUsage:       42.5,
			MemoryUsage:    65.3,
			TCPConnections: 120,
			Timestamp:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		active: true,
	}

	handler := NewMetricsHandler(&config.Config{}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify JSON contains expected fields
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if body["cpuUsage"] == nil && body["cpu_usage"] == nil {
		t.Error("expected cpuUsage or cpu_usage in response")
	}
}

func TestGetCurrentMetrics_Fresh(t *testing.T) {
	mock := &mockMonitorQuerier{
		metrics: &models.MetricsResponse{
			CPUUsage:    10.0,
			MemoryUsage: 20.0,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		freshMetrics: &models.MetricsResponse{
			CPUUsage:    55.5,
			MemoryUsage: 77.7,
			Timestamp:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		active: true,
	}

	handler := NewMetricsHandler(&config.Config{}, mock)

	// Request with fresh=true
	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current?fresh=true", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestGetCurrentMetrics_Error(t *testing.T) {
	mock := &mockMonitorQuerier{
		err: fmt.Errorf("collection failed"),
	}

	handler := NewMetricsHandler(&config.Config{}, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/current", nil)
	w := httptest.NewRecorder()

	handler.GetCurrentMetrics(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
