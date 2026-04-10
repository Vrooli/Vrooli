package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tunnel-manager/domain"
)

// [REQ:OBS-001] Metrics handler tests

func TestHandlerMetricsHistory_DefaultHours(t *testing.T) {
	querier := &mockMetricsQuerier{
		queryFn: func(from, to time.Time) ([]domain.MetricsRecord, error) {
			return []domain.MetricsRecord{
				{HAConnections: 4, ActiveStreams: 10},
			}, nil
		},
	}

	h := HandleMetricsHistory(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []domain.MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_CustomHours(t *testing.T) {
	querier := &mockMetricsQuerier{
		queryFn: func(from, to time.Time) ([]domain.MetricsRecord, error) {
			return []domain.MetricsRecord{
				{HAConnections: 2},
			}, nil
		},
	}

	h := HandleMetricsHistory(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history?hours=1", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []domain.MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_EmptyResult(t *testing.T) {
	querier := &mockMetricsQuerier{
		queryFn: func(from, to time.Time) ([]domain.MetricsRecord, error) {
			return []domain.MetricsRecord{}, nil
		},
	}

	h := HandleMetricsHistory(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []domain.MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestHandlerMetricsHistory_InvalidHoursParam(t *testing.T) {
	querier := &mockMetricsQuerier{
		queryFn: func(from, to time.Time) ([]domain.MetricsRecord, error) {
			return []domain.MetricsRecord{
				{HAConnections: 1},
			}, nil
		},
	}

	h := HandleMetricsHistory(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/history?hours=abc", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var records []domain.MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &records); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record (default 24h window), got %d", len(records))
	}
}

// [REQ:OBS-001] Metrics latest handler tests

func TestHandlerMetricsLatest_WithData(t *testing.T) {
	querier := &mockMetricsQuerier{
		latestFn: func() (*domain.MetricsRecord, error) {
			return &domain.MetricsRecord{HAConnections: 4, ActiveStreams: 8}, nil
		},
	}

	h := HandleMetricsLatest(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/latest", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var record domain.MetricsRecord
	if err := json.Unmarshal(w.Body.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.HAConnections != 4 {
		t.Errorf("HAConnections = %d, want 4", record.HAConnections)
	}
}

func TestHandlerMetricsLatest_NoData(t *testing.T) {
	querier := &mockMetricsQuerier{
		latestFn: func() (*domain.MetricsRecord, error) {
			return nil, nil
		},
	}

	h := HandleMetricsLatest(querier)
	req := httptest.NewRequest("GET", "/api/v1/metrics/latest", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["status"] != "no_data" {
		t.Errorf("expected status=no_data, got %q", resp["status"])
	}
}
