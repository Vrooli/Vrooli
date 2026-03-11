// [REQ:LD-DIGEST-WEEKLY] Tests for weekly digest handlers.
package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lifestyle-dashboard/domain"
)

// TestGetCurrentDigest_ReturnsDigest tests that the current digest endpoint works.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetCurrentDigest_ReturnsDigest(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register a domain and create some events
	h.registerTestDomain(t, "sleep", "Sleep Tracker")
	h.createTestEvent(t, "sleep", "bedtime")
	h.createTestEvent(t, "sleep", "waketime")

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Digest.WeekStartDate == "" {
		t.Error("Expected non-empty week start date")
	}
	if resp.Digest.WeekEndDate == "" {
		t.Error("Expected non-empty week end date")
	}
	if resp.Digest.Summary == "" {
		t.Error("Expected non-empty summary")
	}
}

// TestGetDigestByWeek_ReturnsDigestForSpecifiedWeek tests digest generation for a specific week.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetDigestByWeek_ReturnsDigestForSpecifiedWeek(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Use last Monday
	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	lastMonday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	// Create a route that includes PathValue
	req := httptest.NewRequest("GET", "/api/v1/digests/"+lastMonday, nil)
	req.SetPathValue("week_start", lastMonday)
	rr := httptest.NewRecorder()

	h.GetDigestByWeek(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Digest.WeekStartDate != lastMonday {
		t.Errorf("Expected week start %s, got %s", lastMonday, resp.Digest.WeekStartDate)
	}
}

// TestGetDigestByWeek_MissingWeekStart returns validation error.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetDigestByWeek_MissingWeekStart(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Request without week_start path value
	req := httptest.NewRequest("GET", "/api/v1/digests/", nil)
	rr := httptest.NewRecorder()

	h.GetDigestByWeek(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

// TestGetCurrentDigest_IncludesDomainChanges tests that domain changes are included.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetCurrentDigest_IncludesDomainChanges(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	// Register multiple domains with events
	h.registerTestDomain(t, "sleep", "Sleep Tracker")
	h.registerTestDomain(t, "exercise", "Exercise Log")
	h.createTestEvent(t, "sleep", "bedtime")
	h.createTestEvent(t, "exercise", "workout")

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have domain changes
	if len(resp.Digest.DomainChanges) == 0 {
		t.Error("Expected domain changes to be populated")
	}
}

// TestGetCurrentDigest_IncludesScoreTrend tests that score trend is included.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetCurrentDigest_IncludesScoreTrend(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	h.registerTestDomain(t, "sleep", "Sleep Tracker")
	h.createTestEvent(t, "sleep", "bedtime")

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Score trend should have a valid direction
	validDirections := map[string]bool{"up": true, "down": true, "stable": true}
	if !validDirections[resp.Digest.ScoreTrend.Direction] {
		t.Errorf("Expected valid direction, got '%s'", resp.Digest.ScoreTrend.Direction)
	}
}

// TestGetCurrentDigest_IncludesHighlights tests that highlights are generated.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetCurrentDigest_IncludesHighlights(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should always have at least one highlight
	if len(resp.Digest.Highlights) == 0 {
		t.Error("Expected at least one highlight")
	}
}

// TestGetCurrentDigest_IncludesNextWeekFocus tests that next week focus is generated.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetCurrentDigest_IncludesNextWeekFocus(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should always have at least one focus item
	if len(resp.Digest.NextWeekFocus) == 0 {
		t.Error("Expected at least one next week focus item")
	}
}

// TestDigest_EmptyDatabase tests digest generation with no data.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_EmptyDatabase(t *testing.T) {
	h, db := setupTestHandler(t)
	defer db.Close()

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	// Should still succeed even with empty database
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp domain.WeeklyDigestResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Summary should indicate no activity
	if resp.Digest.Summary == "" {
		t.Error("Expected non-empty summary even for empty database")
	}
}

// TestGetCurrentDigest_NilRepository verifies error when digest repository is nil.
// [REQ:LD-DIGEST-WEEKLY] Handler gracefully handles missing repository.
func TestGetCurrentDigest_NilRepository(t *testing.T) {
	h := &Handler{Digest: nil}

	req := httptest.NewRequest("GET", "/api/v1/digests/current", nil)
	rr := httptest.NewRecorder()

	h.GetCurrentDigest(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}

// TestGetDigestByWeek_NilRepository verifies error when digest repository is nil.
// [REQ:LD-DIGEST-WEEKLY] Handler gracefully handles missing repository.
func TestGetDigestByWeek_NilRepository(t *testing.T) {
	h := &Handler{Digest: nil}

	req := httptest.NewRequest("GET", "/api/v1/digests/2026-03-10", nil)
	req.SetPathValue("week_start", "2026-03-10")
	rr := httptest.NewRecorder()

	h.GetDigestByWeek(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d: %s", http.StatusServiceUnavailable, rr.Code, rr.Body.String())
	}
}
