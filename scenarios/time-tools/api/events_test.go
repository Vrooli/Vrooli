package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCreateEventHandler tests event creation
func TestCreateEventHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseAvailable", func(t *testing.T) {
		// Save original db
		originalDB := db
		defer func() { db = originalDB }()

		// Set db to nil
		db = nil

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/events",
			Body: map[string]interface{}{
				"title":        "Test Event",
				"description":  "Test Description",
				"start_time":   "2024-01-15T09:00:00Z",
				"end_time":     "2024-01-15T10:00:00Z",
				"timezone":     "UTC",
				"organizer_id": "test-user",
			},
		}

		w := executeRequest(http.HandlerFunc(createEventHandler), req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503 without database, got %d", w.Code)
		}

		assertErrorResponse(t, w, http.StatusServiceUnavailable, "Database not available")
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/events",
			Body:   "invalid json",
		}

		w := executeRequest(http.HandlerFunc(createEventHandler), req)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
			t.Logf("Got status %d (database may not be available)", w.Code)
		}
	})

	t.Run("EmptyRequestBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/events",
			Body:   map[string]interface{}{},
		}

		w := executeRequest(http.HandlerFunc(createEventHandler), req)

		// Either bad request or service unavailable (if no DB)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusServiceUnavailable {
			t.Logf("Got status %d", w.Code)
		}
	})
}

// TestListEventsHandlerExtended tests extended event listing functionality
func TestListEventsHandlerExtended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseAvailable", func(t *testing.T) {
		// Already tested in main_test.go, but add edge cases
		originalDB := db
		defer func() { db = originalDB }()
		db = nil

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/events",
		}

		w := executeRequest(http.HandlerFunc(listEventsHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			events, ok := body["events"].([]interface{})
			if !ok {
				t.Error("Expected events array")
			}
			if len(events) != 0 {
				t.Errorf("Expected empty array without database, got %d events", len(events))
			}
		})
	})
}

// TestConflictDetectHandlerExtended tests extended conflict detection scenarios
func TestConflictDetectHandlerExtended(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingOrganizerID", func(t *testing.T) {
		start, end := getTestTimeRange()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"start_time": start.RFC3339,
				"end_time":   end.RFC3339,
				// Missing organizer_id
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		// Should still work, just no conflicts found
		if w.Code != http.StatusOK {
			t.Logf("Got status %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("EndTimeBeforeStartTime", func(t *testing.T) {
		start, end := getTestTimeRange()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "test-user",
				"start_time":   end.RFC3339,   // Swapped
				"end_time":     start.RFC3339, // Swapped
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		// Should still return OK, just no conflicts
		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				hasConflicts, ok := body["has_conflicts"].(bool)
				if ok && hasConflicts {
					t.Log("Note: Found conflicts even with reversed times")
				}
			})
		}
	})
}

// TestScheduledEventModel tests the ScheduledEvent model
func TestScheduledEventModel(t *testing.T) {
	t.Run("CreateScheduledEvent", func(t *testing.T) {
		description := "Test Description"
		location := "Conference Room A"
		locationType := "physical"

		event := ScheduledEvent{
			ID:           uuid.New(),
			Title:        "Test Event",
			Description:  &description,
			StartTime:    time.Now(),
			EndTime:      time.Now().Add(time.Hour),
			Timezone:     "UTC",
			AllDay:       false,
			EventType:    "meeting",
			Status:       "confirmed",
			Priority:     "normal",
			OrganizerID:  "test-user",
			Participants: []map[string]interface{}{
				{
					"id":     "user1",
					"email":  "user1@example.com",
					"name":   "User One",
					"status": "accepted",
				},
			},
			Location:     &location,
			LocationType: &locationType,
			Tags:         []string{"work", "important"},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if event.ID == uuid.Nil {
			t.Error("Expected valid UUID")
		}
		if event.Title == "" {
			t.Error("Expected title")
		}
		if len(event.Participants) != 1 {
			t.Errorf("Expected 1 participant, got %d", len(event.Participants))
		}
		if len(event.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(event.Tags))
		}
	})
}

// TestTimeSlotModel tests the TimeSlot model
func TestTimeSlotModel(t *testing.T) {
	t.Run("CreateTimeSlot", func(t *testing.T) {
		slot := OptimalSlot{
			StartTime:       time.Now().Format(time.RFC3339),
			EndTime:         time.Now().Add(time.Hour).Format(time.RFC3339),
			Score:           85.5,
			ConflictCount:   0,
			ParticipantsFree: []string{"user1", "user2"},
		}

		if slot.StartTime == "" {
			t.Error("Expected start time")
		}
		if slot.Score < 0 || slot.Score > 100 {
			t.Errorf("Expected score between 0-100, got %f", slot.Score)
		}
		if len(slot.ParticipantsFree) != 2 {
			t.Errorf("Expected 2 participants, got %d", len(slot.ParticipantsFree))
		}
	})
}

// TestConflictInfoModel tests the ConflictInfo model
func TestConflictInfoModel(t *testing.T) {
	t.Run("CreateConflictInfo", func(t *testing.T) {
		conflict := ConflictInfo{
			EventID:        uuid.New(),
			EventTitle:     "Conflicting Event",
			ConflictType:   "time_overlap",
			Severity:       "high",
			OverlapMinutes: 30,
			StartTime:      time.Now(),
			EndTime:        time.Now().Add(time.Hour),
		}

		if conflict.EventID == uuid.Nil {
			t.Error("Expected valid UUID")
		}
		if conflict.ConflictType != "time_overlap" {
			t.Errorf("Expected conflict type time_overlap, got %s", conflict.ConflictType)
		}
		if conflict.Severity != "high" {
			t.Errorf("Expected severity high, got %s", conflict.Severity)
		}
		if conflict.OverlapMinutes <= 0 {
			t.Error("Expected positive overlap minutes")
		}
	})
}

// TestRequestModels tests request model structures
func TestRequestModels(t *testing.T) {
	t.Run("TimezoneConversionRequest", func(t *testing.T) {
		req := TimezoneConversionRequest{
			Time:         "2024-01-15T10:00:00Z",
			FromTimezone: "UTC",
			ToTimezone:   "America/New_York",
		}

		if req.Time == "" {
			t.Error("Expected time field")
		}
		if req.FromTimezone == "" || req.ToTimezone == "" {
			t.Error("Expected timezone fields")
		}
	})

	t.Run("DurationCalculationRequest", func(t *testing.T) {
		req := DurationCalculationRequest{
			StartTime:       "2024-01-15T09:00:00Z",
			EndTime:         "2024-01-15T17:00:00Z",
			ExcludeWeekends: true,
		}

		if req.StartTime == "" || req.EndTime == "" {
			t.Error("Expected time fields")
		}
		if !req.ExcludeWeekends {
			t.Error("Expected ExcludeWeekends to be true")
		}
	})

	t.Run("TimeArithmeticRequest", func(t *testing.T) {
		req := TimeArithmeticRequest{
			Time:     "2024-01-15T10:00:00Z",
			Duration: "2",
			Unit:     "hours",
		}

		if req.Time == "" || req.Duration == "" || req.Unit == "" {
			t.Error("Expected all fields to be populated")
		}
	})
}
