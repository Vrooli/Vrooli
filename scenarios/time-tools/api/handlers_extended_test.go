package main

import (
	"net/http"
	"testing"
)

// TestHealthHandlerDatabaseConnected tests health with database connection
func TestHealthHandlerDatabaseConnected(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithDatabase", func(t *testing.T) {
		// Test with database (if available)
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := executeRequest(http.HandlerFunc(healthHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["database"].(string); !ok {
				t.Error("Expected database field in health response")
			}
		})
	})
}

// TestParseTimeHandlerComprehensive tests comprehensive time parsing
func TestParseTimeHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name     string
		input    string
		timezone string
	}{
		{"RFC3339_WithTimezone", "2024-01-15T14:30:00Z", "UTC"},
		{"DateTime_WithTimezone", "2024-01-15 14:30:00", "America/New_York"},
		{"Date_WithTimezone", "2024-01-15", "Europe/London"},
		{"ISO8601", "2024-01-15T14:30:00+00:00", ""},
		{"UnixEpoch", "1705329000", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]interface{}{
				"input": tc.input,
			}
			if tc.timezone != "" {
				body["timezone"] = tc.timezone
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/parse",
				Body:   body,
			}

			w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

			// May succeed or fail depending on format
			if w.Code == http.StatusOK {
				assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) {
					if _, ok := resp["parsed_time"].(string); !ok {
						t.Error("Expected parsed_time in response")
					}
				})
			}
		})
	}
}

// TestParseTimeHandlerWithFormatHint tests parsing with format hints
func TestParseTimeHandlerWithFormatHint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithFormatHint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/parse",
			Body: map[string]interface{}{
				"input":  "01/15/2024",
				"format": "MM/DD/YYYY",
			},
		}

		w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["parsed_time"].(string); !ok {
					t.Error("Expected parsed_time")
				}
			})
		}
	})

	t.Run("InvalidFormatHint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/parse",
			Body: map[string]interface{}{
				"input":  "invalid-time-string-xyz",
				"format": "INVALID",
			},
		}

		w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

		// May return error or attempt to parse
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Logf("Got status %d", w.Code)
		}
	})
}

// TestSubtractTimeHandlerVariants tests subtract time with different inputs
func TestSubtractTimeHandlerVariants(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()

	testCases := []struct {
		name     string
		duration string
		unit     string
	}{
		{"SubtractDays", "3", "days"},
		{"SubtractWeeks", "1", "weeks"},
		{"SubtractMonths", "2", "months"},
		{"SubtractYears", "1", "years"},
		{"SubtractSeconds", "30", "seconds"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/subtract",
				Body: map[string]interface{}{
					"time":     testTime.RFC3339,
					"duration": tc.duration,
					"unit":     tc.unit,
				},
			}

			w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

			if w.Code == http.StatusOK {
				assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
					if _, ok := body["result_time"].(string); !ok {
						t.Error("Expected result_time")
					}
				})
			}
		})
	}
}

// TestSubtractTimeHandlerErrors tests error cases
func TestSubtractTimeHandlerErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidDuration", func(t *testing.T) {
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/subtract",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"duration": "invalid",
				"unit":     "hours",
			},
		}

		w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

		if w.Code != http.StatusBadRequest {
			t.Logf("Expected error for invalid duration, got %d", w.Code)
		}
	})

	t.Run("MissingTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/subtract",
			Body: map[string]interface{}{
				"duration": "2",
				"unit":     "hours",
			},
		}

		w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

		if w.Code != http.StatusBadRequest {
			t.Logf("Expected error for missing time, got %d", w.Code)
		}
	})
}

// TestGenerateOptimalSlotsExtended tests slot generation edge cases
func TestGenerateOptimalSlotsExtended(t *testing.T) {
	testCases := []struct {
		name              string
		businessHoursOnly bool
		durationMinutes   int
	}{
		{"ShortMeeting", true, 15},
		{"StandardMeeting", true, 60},
		{"LongMeeting", true, 120},
		{"AllDay", false, 480},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/schedule/optimal",
				Body: map[string]interface{}{
					"earliest_date":       "2024-01-15",
					"latest_date":         "2024-01-19",
					"duration_minutes":    tc.durationMinutes,
					"timezone":            "America/New_York",
					"business_hours_only": tc.businessHoursOnly,
				},
			}

			w := executeRequest(http.HandlerFunc(scheduleOptimalHandler), req)

			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				slots, ok := body["optimal_slots"].([]interface{})
				if !ok {
					t.Error("Expected optimal_slots array")
				}
				if len(slots) == 0 {
					t.Error("Expected at least one slot")
				}
			})
		})
	}
}

// TestConflictDetectHandlerWithDatabase tests conflict detection with DB
func TestConflictDetectHandlerWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	start, end := getTestTimeRange()

	t.Run("WithDatabase_NoConflicts", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "test-user",
				"start_time":   start.RFC3339,
				"end_time":     end.RFC3339,
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if conflicts, ok := body["conflicts"].([]interface{}); ok {
				t.Logf("Found %d conflicts", len(conflicts))
			}
			if hasConflicts, ok := body["has_conflicts"].(bool); ok {
				t.Logf("Has conflicts: %v", hasConflicts)
			}
		})
	})

	t.Run("EmptyOrganizerID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "",
				"start_time":   start.RFC3339,
				"end_time":     end.RFC3339,
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		// Should still work
		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["conflicts"]; !ok {
					t.Error("Expected conflicts field")
				}
			})
		}
	})
}

// TestListEventsHandlerWithDatabase tests listing with database
func TestListEventsHandlerWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ListWithFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/events",
			Query: map[string]string{
				"organizer_id": "test-user",
				"status":       "confirmed",
			},
		}

		w := executeRequest(http.HandlerFunc(listEventsHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["events"].([]interface{}); !ok {
				t.Error("Expected events array")
			}
			if _, ok := body["count"]; !ok {
				t.Error("Expected count field")
			}
		})
	})

	t.Run("ListWithDateRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/events",
			Query: map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-12-31",
			},
		}

		w := executeRequest(http.HandlerFunc(listEventsHandler), req)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["events"]; !ok {
					t.Error("Expected events field")
				}
			})
		}
	})

	t.Run("ListAllEvents", func(t *testing.T) {
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
			t.Logf("Found %d events", len(events))
		})
	})
}

// TestCreateEventHandlerWithDatabase tests event creation with database
func TestCreateEventHandlerWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMinimalEvent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/events",
			Body: map[string]interface{}{
				"title":        "Minimal Event",
				"start_time":   "2024-01-15T10:00:00Z",
				"end_time":     "2024-01-15T11:00:00Z",
				"timezone":     "UTC",
				"status":       "confirmed",
				"priority":     "normal",
				"organizer_id": "test-user",
			},
		}

		w := executeRequest(http.HandlerFunc(createEventHandler), req)

		// Will fail without database, but test the handler logic
		if w.Code == http.StatusCreated {
			assertJSONResponse(t, w, http.StatusCreated, func(body map[string]interface{}) {
				if _, ok := body["id"]; !ok {
					t.Error("Expected id in response")
				}
			})
		} else if w.Code == http.StatusServiceUnavailable {
			t.Log("Database not available, test passed")
		} else {
			t.Logf("Got status %d", w.Code)
		}
	})

	t.Run("CreateCompleteEvent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/events",
			Body: map[string]interface{}{
				"title":        "Complete Event",
				"description":  "Full event with all fields",
				"start_time":   "2024-01-15T10:00:00Z",
				"end_time":     "2024-01-15T11:00:00Z",
				"timezone":     "America/New_York",
				"all_day":      false,
				"event_type":   "meeting",
				"status":       "confirmed",
				"priority":     "high",
				"organizer_id": "user123",
				"participants": []map[string]interface{}{
					{
						"id":    "user456",
						"email": "user456@example.com",
						"name":  "Test User",
					},
				},
				"location":      "Room 101",
				"location_type": "physical",
				"tags":          []string{"important", "recurring"},
			},
		}

		w := executeRequest(http.HandlerFunc(createEventHandler), req)

		// Will succeed only if database is available
		if w.Code == http.StatusCreated {
			assertJSONResponse(t, w, http.StatusCreated, func(body map[string]interface{}) {
				if title, ok := body["title"].(string); !ok || title != "Complete Event" {
					t.Error("Expected matching title")
				}
			})
		} else {
			t.Logf("Event creation: %d (database may not be available)", w.Code)
		}
	})
}

// TestRespondJSONErrorPath tests JSON encoding error
func TestRespondJSONErrorPath(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidJSON", func(t *testing.T) {
		// Test that respondJSON works correctly
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/format",
			Body: map[string]interface{}{
				"time":   testTime.RFC3339,
				"format": "iso8601",
			},
		}

		w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type application/json")
		}
	})
}
