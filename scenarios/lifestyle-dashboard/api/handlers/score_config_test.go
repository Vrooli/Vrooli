package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifestyle-dashboard/domain"
)

// [REQ:LD-SCORE-CALC] Tests for score configuration API endpoints.

func TestGetScoreConfig_Empty(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/score/config", nil)
	rr := httptest.NewRecorder()
	h.GetScoreConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp domain.ScoreConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.DefaultWeight != "medium" {
		t.Errorf("Expected default weight 'medium', got '%s'", resp.DefaultWeight)
	}
}

func TestGetScoreConfig_WithDomains(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register domains
	h.registerTestDomain(t, "sleep", "Sleep Tracker")
	h.registerTestDomain(t, "exercise", "Exercise Tracker")

	req := httptest.NewRequest("GET", "/api/v1/score/config", nil)
	rr := httptest.NewRecorder()
	h.GetScoreConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp domain.ScoreConfigResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Weights) != 2 {
		t.Errorf("Expected 2 weights, got %d", len(resp.Weights))
	}

	// Verify preset weights applied
	for _, w := range resp.Weights {
		if w.Domain == "sleep" && w.Weight != "high" {
			t.Errorf("Expected sleep preset 'high', got '%s'", w.Weight)
		}
		if w.Domain == "exercise" && w.Weight != "high" {
			t.Errorf("Expected exercise preset 'high', got '%s'", w.Weight)
		}
	}
}

func TestGetDomainWeight_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "sleep", "Sleep Tracker")

	req := httptest.NewRequest("GET", "/api/v1/score/config/sleep", nil)
	rr := httptest.NewRecorder()
	h.GetDomainWeight(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var weight domain.DomainWeightConfig
	if err := json.NewDecoder(rr.Body).Decode(&weight); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if weight.Weight != "high" {
		t.Errorf("Expected preset weight 'high', got '%s'", weight.Weight)
	}
	if weight.Multiplier != 3.0 {
		t.Errorf("Expected multiplier 3.0, got %f", weight.Multiplier)
	}
}

func TestGetDomainWeight_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/score/config/nonexistent", nil)
	rr := httptest.NewRecorder()
	h.GetDomainWeight(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestUpdateDomainWeight_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "sleep", "Sleep Tracker")

	body := `{"weight": "low"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/sleep", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var weight domain.DomainWeightConfig
	if err := json.NewDecoder(rr.Body).Decode(&weight); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if weight.Weight != "low" {
		t.Errorf("Expected updated weight 'low', got '%s'", weight.Weight)
	}
	if weight.Multiplier != 1.0 {
		t.Errorf("Expected multiplier 1.0, got %f", weight.Multiplier)
	}
}

func TestUpdateDomainWeight_InvalidWeight(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "sleep", "Sleep Tracker")

	body := `{"weight": "invalid"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/sleep", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestUpdateDomainWeight_NotFound(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"weight": "high"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rr.Code)
	}
}

func TestUpdateDomainWeight_InvalidJSON(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "sleep", "Sleep Tracker")

	req := httptest.NewRequest("PUT", "/api/v1/score/config/sleep", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestUpdateDomainWeight_SetNone(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "socialization", "Social Tracker")

	body := `{"weight": "none"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/socialization", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var weight domain.DomainWeightConfig
	if err := json.NewDecoder(rr.Body).Decode(&weight); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if weight.Multiplier != 0.0 {
		t.Errorf("Expected multiplier 0.0 for 'none', got %f", weight.Multiplier)
	}
}

// TestGetScoreConfig_NilRepository verifies error when score config repository is nil.
// [REQ:LD-SCORE-CALC] Handler gracefully handles missing repository.
func TestGetScoreConfig_NilRepository(t *testing.T) {
	h := &Handler{ScoreConfig: nil}

	req := httptest.NewRequest("GET", "/api/v1/score/config", nil)
	rr := httptest.NewRecorder()

	h.GetScoreConfig(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestGetDomainWeight_NilRepository verifies error when score config repository is nil.
// [REQ:LD-SCORE-CALC] Handler gracefully handles missing repository.
func TestGetDomainWeight_NilRepository(t *testing.T) {
	h := &Handler{ScoreConfig: nil}

	req := httptest.NewRequest("GET", "/api/v1/score/config/sleep", nil)
	rr := httptest.NewRecorder()

	h.GetDomainWeight(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestUpdateDomainWeight_NilRepository verifies error when score config repository is nil.
// [REQ:LD-SCORE-CALC] Handler gracefully handles missing repository.
func TestUpdateDomainWeight_NilRepository(t *testing.T) {
	h := &Handler{ScoreConfig: nil}

	body := `{"weight": "high"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/sleep", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestGetDomainWeight_MissingDomainName verifies validation when domain name is empty.
// [REQ:LD-SCORE-CALC] Handler validates required domain parameter.
func TestGetDomainWeight_MissingDomainName(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Request with empty domain
	req := httptest.NewRequest("GET", "/api/v1/score/config/", nil)
	rr := httptest.NewRecorder()

	h.GetDomainWeight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestUpdateDomainWeight_MissingDomainName verifies validation when domain name is empty.
// [REQ:LD-SCORE-CALC] Handler validates required domain parameter.
func TestUpdateDomainWeight_MissingDomainName(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	body := `{"weight": "high"}`
	req := httptest.NewRequest("PUT", "/api/v1/score/config/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	h.UpdateDomainWeight(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}
