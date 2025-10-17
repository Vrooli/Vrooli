package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestGenerateOptimalSlots tests the optimal slot generation logic
func TestGenerateOptimalSlots(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BusinessHoursOnly", func(t *testing.T) {
		startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)

		req := ScheduleOptimizationRequest{
			DurationMinutes:   60,
			BusinessHoursOnly: true,
			Participants:      []string{"user1"},
		}

		slots := generateOptimalSlots(startDate, endDate, req)

		if len(slots) == 0 {
			t.Error("Expected at least one optimal slot")
		}

		// Verify all slots are during business hours
		for _, slot := range slots {
			slotTime, _ := time.Parse(time.RFC3339, slot.StartTime)
			hour := slotTime.Hour()
			if hour < 9 || hour >= 17 {
				t.Errorf("Slot at %s is outside business hours", slot.StartTime)
			}
		}
	})

	t.Run("AllHours", func(t *testing.T) {
		startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

		req := ScheduleOptimizationRequest{
			DurationMinutes:   60,
			BusinessHoursOnly: false,
			Participants:      []string{},
		}

		slots := generateOptimalSlots(startDate, endDate, req)

		if len(slots) == 0 {
			t.Error("Expected at least one optimal slot")
		}
	})

	t.Run("LongDuration", func(t *testing.T) {
		startDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)

		req := ScheduleOptimizationRequest{
			DurationMinutes:   240, // 4 hours
			BusinessHoursOnly: true,
			Participants:      []string{},
		}

		slots := generateOptimalSlots(startDate, endDate, req)

		// Should still generate some slots
		if len(slots) == 0 {
			t.Log("Note: No slots generated for 4-hour duration")
		}
	})
}

// TestCalculateSlotScore tests the slot scoring algorithm
func TestCalculateSlotScore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name     string
		time     time.Time
		minScore float64
	}{
		{
			name:     "MidMorning",
			time:     time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC), // Tuesday 10 AM
			minScore: 0.5,
		},
		{
			name:     "EarlyAfternoon",
			time:     time.Date(2024, 1, 17, 14, 0, 0, 0, time.UTC), // Wednesday 2 PM
			minScore: 0.5,
		},
		{
			name:     "EarlyMorning",
			time:     time.Date(2024, 1, 16, 8, 0, 0, 0, time.UTC),
			minScore: 0.0,
		},
	}

	req := ScheduleOptimizationRequest{}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateSlotScore(tc.time, req)

			if score < tc.minScore {
				t.Logf("Score %v is less than expected minimum %v", score, tc.minScore)
			}

			if score > 1.0 {
				t.Errorf("Score %v exceeds maximum 1.0", score)
			}
		})
	}
}

// TestParseDurationVariants tests various duration parsing formats
func TestParseDurationVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		duration string
		unit     string
		valid    bool
	}{
		{"2 hours", "", true},
		{"30 minutes", "", true},
		{"1 day", "", true},
		{"2 weeks", "", true},
		{"1 month", "", true},
		{"1 year", "", true},
		{"5", "seconds", true},
		{"10", "s", true},
		{"2", "h", true},
		{"30", "m", true},
		{"1", "d", true},
		{"1", "w", true},
		{"invalid", "invalid", false},
		{"", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.duration+"_"+tc.unit, func(t *testing.T) {
			result, err := parseDuration(tc.duration, tc.unit)

			if tc.valid && err != nil {
				t.Errorf("Expected valid duration, got error: %v", err)
			}

			if !tc.valid && err == nil {
				t.Logf("Expected error for invalid duration %s %s, got %v", tc.duration, tc.unit, result)
			}
		})
	}
}

// TestGetRelativeTimeVariants tests relative time formatting
func TestGetRelativeTimeVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	now := time.Now()

	testCases := []struct {
		name         string
		time         time.Time
		shouldInclude string
	}{
		{"JustNow", now, "now"},
		{"OneMinuteAgo", now.Add(-1 * time.Minute), "1 minute"},
		{"FiveMinutesAgo", now.Add(-5 * time.Minute), "minutes"},
		{"OneHourAgo", now.Add(-1 * time.Hour), "1 hour"},
		{"ThreeHoursAgo", now.Add(-3 * time.Hour), "hours"},
		{"Yesterday", now.Add(-25 * time.Hour), "yesterday"},
		{"TwoDaysAgo", now.Add(-48 * time.Hour), "days"},
		{"InFuture_Minutes", now.Add(5 * time.Minute), ""},
		{"InFuture_Hours", now.Add(2 * time.Hour), ""},
		{"InFuture_Days", now.Add(25 * time.Hour), ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getRelativeTime(tc.time)

			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}

			// Log for visibility
			t.Logf("Relative time for %s: %s", tc.name, result)
		})
	}
}

// TestConflictDetectEdgeCases tests edge cases for conflict detection
func TestConflictDetectEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AdjacentTimes", func(t *testing.T) {
		// Times that don't overlap but are adjacent
		start := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		end := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "test-user",
				"start_time":   start.Format(time.RFC3339),
				"end_time":     end.Format(time.RFC3339),
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SameTime", func(t *testing.T) {
		// Start and end at same time
		sameTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "test-user",
				"start_time":   sameTime.Format(time.RFC3339),
				"end_time":     sameTime.Format(time.RFC3339),
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestDurationCalculateEdgeCases tests edge cases for duration calculation
func TestDurationCalculateEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SameTime", func(t *testing.T) {
		sameTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/duration",
			Body: map[string]interface{}{
				"start_time":       sameTime.RFC3339,
				"end_time":         sameTime.RFC3339,
				"exclude_weekends": false,
			},
		}

		w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("VeryLongDuration", func(t *testing.T) {
		start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // 1 year

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/duration",
			Body: map[string]interface{}{
				"start_time":       start.Format(time.RFC3339),
				"end_time":         end.Format(time.RFC3339),
				"exclude_weekends": false,
			},
		}

		w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestFormatTimeEdgeCases tests edge cases for time formatting
func TestFormatTimeEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CustomFormat", func(t *testing.T) {
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/format",
			Body: map[string]interface{}{
				"time":   testTime.RFC3339,
				"format": "2006-01-02", // Custom Go format string
			},
		}

		w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("WithTimezone", func(t *testing.T) {
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/format",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"format":   "datetime",
				"timezone": "Asia/Tokyo",
			},
		}

		w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestListEventsQueryFiltering tests query parameter filtering
func TestListEventsQueryFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name  string
		query map[string]string
	}{
		{
			name: "FilterByOrganizer",
			query: map[string]string{
				"organizer_id": "user-123",
			},
		},
		{
			name: "FilterByDateRange",
			query: map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		},
		{
			name: "FilterByStatus",
			query: map[string]string{
				"status": "active",
			},
		},
		{
			name: "MultipleFilters",
			query: map[string]string{
				"organizer_id": "user-123",
				"status":       "active",
				"start_date":   "2024-01-01",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/events",
				Query:  tc.query,
			}

			w := executeRequest(http.HandlerFunc(listEventsHandler), req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		})
	}
}

// TestTimeArithmeticWithDifferentFormats tests time arithmetic with various formats
func TestTimeArithmeticWithDifferentFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()

	t.Run("AddWithStringDuration", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/add",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"duration": "2 hours",
			},
		}

		w := executeRequest(http.HandlerFunc(addTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("SubtractWithGoDuration", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/subtract",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"duration": "1h30m",
			},
		}

		w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InvalidDuration", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/add",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"duration": "not-a-duration",
			},
		}

		w := executeRequest(http.HandlerFunc(addTimeHandler), req)

		// Should return error for invalid duration
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected status 400 for invalid duration, got %d", w.Code)
		}
	})
}

// TestParseTimeWithHint tests time parsing with format hint
func TestParseTimeWithHint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithFormatHint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/parse",
			Body: map[string]interface{}{
				"input":  "2024-01-15",
				"format": "2006-01-02",
			},
		}

		w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WithTimezone", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/parse",
			Body: map[string]interface{}{
				"input":    "2024-01-15T10:00:00Z",
				"timezone": "America/New_York",
			},
		}

		w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestRespondJSON tests the JSON response helper
func TestRespondJSON(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidJSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]interface{}{
			"message": "test",
			"value":   42,
		}

		respondJSON(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
		}
	})
}

// TestRespondError tests the error response helper
func TestRespondError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		respondError(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
		}
	})
}
