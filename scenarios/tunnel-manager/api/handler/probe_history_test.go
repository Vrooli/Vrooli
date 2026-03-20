package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

// [REQ:OBS-002] Probe history handler tests

func TestHandlerProbeHistory_DefaultLimit(t *testing.T) {
	reader := &mockProbeHistoryReader{
		queryRecentFn: func(limit int) ([]domain.StoredProbeResult, error) {
			results := make([]domain.StoredProbeResult, 3)
			for i := range results {
				results[i] = domain.StoredProbeResult{
					ID: i + 1, RouteID: 1, ProbeType: "internal", Status: "up",
				}
			}
			return results, nil
		},
	}

	h := HandleProbeHistory(reader)
	req := httptest.NewRequest("GET", "/api/v1/probes/history", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []domain.StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestHandlerProbeHistory_CustomLimit(t *testing.T) {
	reader := &mockProbeHistoryReader{
		queryRecentFn: func(limit int) ([]domain.StoredProbeResult, error) {
			// Respect limit
			results := make([]domain.StoredProbeResult, limit)
			for i := range results {
				results[i] = domain.StoredProbeResult{
					ID: i + 1, RouteID: 1, ProbeType: "internal", Status: "up",
				}
			}
			return results, nil
		},
	}

	h := HandleProbeHistory(reader)
	req := httptest.NewRequest("GET", "/api/v1/probes/history?limit=2", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []domain.StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestHandlerProbeHistory_Empty(t *testing.T) {
	reader := &mockProbeHistoryReader{
		queryRecentFn: func(limit int) ([]domain.StoredProbeResult, error) {
			return []domain.StoredProbeResult{}, nil
		},
	}

	h := HandleProbeHistory(reader)
	req := httptest.NewRequest("GET", "/api/v1/probes/history", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var results []domain.StoredProbeResult
	if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}
