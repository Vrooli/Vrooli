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
