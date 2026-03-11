package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lifestyle-dashboard/domain"
)

// TestQueryEvents_WithFilters verifies event querying with domain filter.
// [REQ:LD-QUERY-FILTER] Handler-level domain filtering.
func TestQueryEvents_WithFilters(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "domain-a", "event.a")
	h.createTestEvent(t, "domain-b", "event.b")
	h.createTestEvent(t, "domain-a", "event.c")

	// Query with filter
	req := httptest.NewRequest("GET", "/api/v1/events?domain=domain-a", nil)
	rr := httptest.NewRecorder()

	h.QueryEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.EventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("Expected 2 events for domain-a, got %d", resp.Count)
	}
}

// TestQueryEvents_TypeFilter verifies event querying with event_type filter.
// [REQ:LD-QUERY-FILTER] Handler-level event type filtering.
func TestQueryEvents_TypeFilter(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events with different types
	h.createTestEvent(t, "domain-a", "session.start")
	h.createTestEvent(t, "domain-a", "session.end")
	h.createTestEvent(t, "domain-a", "session.start")

	// Query with type filter
	req := httptest.NewRequest("GET", "/api/v1/events?event_type=session.start", nil)
	rr := httptest.NewRecorder()

	h.QueryEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.EventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count != 2 {
		t.Errorf("Expected 2 events of type session.start, got %d", resp.Count)
	}
}

// TestQueryEvents_LimitFilter verifies event querying with limit parameter.
// [REQ:LD-QUERY-FILTER] Handler-level limit enforcement.
func TestQueryEvents_LimitFilter(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create more test events than the limit
	for i := 0; i < 5; i++ {
		h.createTestEvent(t, "limit-test", "event.test")
	}

	// Query with limit
	req := httptest.NewRequest("GET", "/api/v1/events?limit=3", nil)
	rr := httptest.NewRecorder()

	h.QueryEvents(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.EventsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Count != 3 {
		t.Errorf("Expected 3 events (limit), got %d", resp.Count)
	}
}

// TestGetTimeline_AggregatesCorrectly verifies timeline statistics.
// [REQ:LD-QUERY-AGGREGATE] Timeline aggregation through handler layer.
func TestGetTimeline_AggregatesCorrectly(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test events
	h.createTestEvent(t, "timeline-test", "event.a")
	h.createTestEvent(t, "timeline-test", "event.b")

	req := httptest.NewRequest("GET", "/api/v1/stats/timeline?days=7", nil)
	rr := httptest.NewRecorder()

	h.GetTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.TimelineResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Days != "7" {
		t.Errorf("Expected days '7', got '%s'", resp.Days)
	}
}

// TestGetTimeline_DefaultDays verifies default timeline days when not specified.
// [REQ:LD-QUERY-AGGREGATE] Timeline default parameter handling.
func TestGetTimeline_DefaultDays(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/timeline", nil)
	rr := httptest.NewRecorder()

	h.GetTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.TimelineResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// Default is 7 days per config
	if resp.Days != "7" {
		t.Errorf("Expected default days '7', got '%s'", resp.Days)
	}
}

// TestGetSummary_ReturnsTotals verifies summary statistics.
// [REQ:LD-QUERY-AGGREGATE] Summary aggregation through handler layer.
func TestGetSummary_ReturnsTotals(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test data
	h.registerTestDomain(t, "summary-domain", "Summary Test")
	h.createTestEvent(t, "summary-domain", "event.a")
	h.createTestEvent(t, "summary-domain", "event.b")

	req := httptest.NewRequest("GET", "/api/v1/stats/summary", nil)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.SummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.TotalEvents < 2 {
		t.Errorf("Expected at least 2 events, got %d", resp.TotalEvents)
	}
	if resp.ActiveDomains < 1 {
		t.Errorf("Expected at least 1 active domain, got %d", resp.ActiveDomains)
	}
}

// TestGetSummary_EmptyDatabase verifies summary with no data.
// [REQ:LD-QUERY-AGGREGATE] Summary aggregation with empty database.
func TestGetSummary_EmptyDatabase(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/summary", nil)
	rr := httptest.NewRecorder()

	h.GetSummary(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.SummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.TotalEvents != 0 {
		t.Errorf("Expected 0 events, got %d", resp.TotalEvents)
	}
	if resp.ActiveDomains != 0 {
		t.Errorf("Expected 0 active domains, got %d", resp.ActiveDomains)
	}
}

// TestGetScore_Success verifies lifestyle score retrieval.
// [REQ:LD-UI-SCORE] Handler returns lifestyle score.
func TestGetScore_Success(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	rr := httptest.NewRecorder()

	h.GetScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.ScoreResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// Empty database should return score of 0
	if resp.Current.Score < 0 || resp.Current.Score > 100 {
		t.Errorf("Expected score 0-100, got %d", resp.Current.Score)
	}
}

// TestGetScore_WithActivity verifies score calculation with domain activity.
// [REQ:LD-UI-SCORE] Handler calculates score based on domain activity.
func TestGetScore_WithActivity(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Create test data across multiple domains
	h.registerTestDomain(t, "score-domain-1", "Score Domain 1")
	h.registerTestDomain(t, "score-domain-2", "Score Domain 2")
	h.createTestEvent(t, "score-domain-1", "test.event")
	h.createTestEvent(t, "score-domain-1", "test.event")
	h.createTestEvent(t, "score-domain-2", "test.event")

	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	rr := httptest.NewRecorder()

	h.GetScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.ScoreResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// With activity, score should be greater than 0
	if resp.Current.Score <= 0 {
		t.Errorf("Expected score > 0 with activity, got %d", resp.Current.Score)
	}
	// Should have domain scores breakdown
	if len(resp.Current.DomainScores) == 0 {
		t.Error("Expected domain scores with activity")
	}
}

// TestGetScore_WithHistoryDays verifies history_days parameter handling.
// [REQ:LD-UI-SCORE] Handler respects history_days parameter.
func TestGetScore_WithHistoryDays(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/score?history_days=14", nil)
	rr := httptest.NewRecorder()

	h.GetScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.ScoreResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// History should contain 14 entries
	if len(resp.History) != 14 {
		t.Errorf("Expected 14 history entries, got %d", len(resp.History))
	}
}

// TestGetScore_DefaultHistoryDays verifies default history_days.
// [REQ:LD-UI-SCORE] Handler uses default history_days.
func TestGetScore_DefaultHistoryDays(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/score", nil)
	rr := httptest.NewRecorder()

	h.GetScore(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.ScoreResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	// Should have history with default 7 days
	if len(resp.History) != 7 {
		t.Errorf("Expected 7 history entries (default), got %d", len(resp.History))
	}
}

// TestGetTimeline_InvalidDays verifies invalid days parameter handling.
// [REQ:LD-QUERY-AGGREGATE] Timeline parameter validation.
func TestGetTimeline_InvalidDays(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/stats/timeline?days=invalid", nil)
	rr := httptest.NewRecorder()

	h.GetTimeline(rr, req)

	// Should still succeed with default days
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.TimelineResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Days != "7" {
		t.Errorf("Expected default days '7', got '%s'", resp.Days)
	}
}

// TestGetTimeline_MaxDays verifies max days limit enforcement.
// [REQ:LD-QUERY-AGGREGATE] Timeline max days enforcement.
func TestGetTimeline_MaxDays(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Request more than max allowed
	req := httptest.NewRequest("GET", "/api/v1/stats/timeline?days=999", nil)
	rr := httptest.NewRecorder()

	h.GetTimeline(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp domain.TimelineResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	// Should be capped at max (365)
	if resp.Days != "365" {
		t.Errorf("Expected capped days '365', got '%s'", resp.Days)
	}
}
