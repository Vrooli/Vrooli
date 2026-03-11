package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifestyle-dashboard/domain"
)

// TestGetCurrentBrief_Success verifies current brief retrieval.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Handler returns appropriate brief.
func TestGetCurrentBrief_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Brief.Type != "morning" && resp.Brief.Type != "evening" {
		t.Errorf("Expected type 'morning' or 'evening', got '%s'", resp.Brief.Type)
	}
	if resp.Brief.GeneratedAt == "" {
		t.Error("Expected GeneratedAt to be set")
	}
}

// TestGetCurrentBrief_WithDomainActivity verifies brief includes domain sections.
// [REQ:LD-BRIEF-CONSOLIDATE] Handler consolidates cross-domain data.
func TestGetCurrentBrief_WithDomainActivity(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register a domain and create events
	h.registerTestDomain(t, "test-domain", "Test Domain")
	h.createTestEvent(t, "test-domain", "test.event")

	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have at least the test domain section
	if len(resp.Brief.Sections) < 1 {
		t.Error("Expected at least 1 section with domain activity")
	}
}

// TestGetMorningBrief_Success verifies morning brief retrieval.
// [REQ:LD-BRIEF-MORNING] Handler generates morning brief.
func TestGetMorningBrief_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/morning", nil)
	rr := httptest.NewRecorder()

	h.GetMorningBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Brief.Type != "morning" {
		t.Errorf("Expected type 'morning', got '%s'", resp.Brief.Type)
	}
}

// TestGetMorningBrief_WithDate verifies date parameter handling.
// [REQ:LD-BRIEF-MORNING] Handler accepts date parameter.
func TestGetMorningBrief_WithDate(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/morning?date=2026-03-10", nil)
	rr := httptest.NewRecorder()

	h.GetMorningBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Brief.Date != "2026-03-10" {
		t.Errorf("Expected date '2026-03-10', got '%s'", resp.Brief.Date)
	}
}

// TestGetMorningBrief_InvalidDate verifies date validation.
// [REQ:LD-BRIEF-MORNING] Handler validates date format.
func TestGetMorningBrief_InvalidDate(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/morning?date=invalid", nil)
	rr := httptest.NewRecorder()

	h.GetMorningBrief(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestGetEveningBrief_Success verifies evening brief retrieval.
// [REQ:LD-BRIEF-EVENING] Handler generates evening brief.
func TestGetEveningBrief_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/evening", nil)
	rr := httptest.NewRecorder()

	h.GetEveningBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Brief.Type != "evening" {
		t.Errorf("Expected type 'evening', got '%s'", resp.Brief.Type)
	}
}

// TestGetEveningBrief_WithDate verifies date parameter handling.
// [REQ:LD-BRIEF-EVENING] Handler accepts date parameter.
func TestGetEveningBrief_WithDate(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/evening?date=2026-03-09", nil)
	rr := httptest.NewRecorder()

	h.GetEveningBrief(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Brief.Date != "2026-03-09" {
		t.Errorf("Expected date '2026-03-09', got '%s'", resp.Brief.Date)
	}
}

// TestGetEveningBrief_InvalidDate verifies date validation.
// [REQ:LD-BRIEF-EVENING] Handler validates date format.
func TestGetEveningBrief_InvalidDate(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/evening?date=not-a-date", nil)
	rr := httptest.NewRecorder()

	h.GetEveningBrief(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}
}

// TestBriefConfig_Included verifies config is included in response.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Brief response includes config.
func TestBriefConfig_Included(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentBrief(rr, req)

	var resp domain.BriefResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Config.MorningHour != 7 {
		t.Errorf("Expected MorningHour 7, got %d", resp.Config.MorningHour)
	}
	if resp.Config.EveningHour != 21 {
		t.Errorf("Expected EveningHour 21, got %d", resp.Config.EveningHour)
	}
}

// TestGetCurrentBrief_NilRepository verifies error when briefs repository is nil.
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING] Handler gracefully handles missing repository.
func TestGetCurrentBrief_NilRepository(t *testing.T) {
	h := &Handler{Briefs: nil}

	req := httptest.NewRequest("GET", "/api/v1/briefs/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentBrief(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestGetMorningBrief_NilRepository verifies error when briefs repository is nil.
// [REQ:LD-BRIEF-MORNING] Handler gracefully handles missing repository.
func TestGetMorningBrief_NilRepository(t *testing.T) {
	h := &Handler{Briefs: nil}

	req := httptest.NewRequest("GET", "/api/v1/briefs/morning", nil)
	rr := httptest.NewRecorder()

	h.GetMorningBrief(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestGetEveningBrief_NilRepository verifies error when briefs repository is nil.
// [REQ:LD-BRIEF-EVENING] Handler gracefully handles missing repository.
func TestGetEveningBrief_NilRepository(t *testing.T) {
	h := &Handler{Briefs: nil}

	req := httptest.NewRequest("GET", "/api/v1/briefs/evening", nil)
	rr := httptest.NewRecorder()

	h.GetEveningBrief(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}
