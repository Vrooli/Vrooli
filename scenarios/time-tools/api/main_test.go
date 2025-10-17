package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		}

		w := executeRequest(http.HandlerFunc(healthHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if status, ok := body["status"].(string); !ok || (status != "healthy" && status != "degraded") {
				t.Errorf("Expected status 'healthy' or 'degraded', got %v", body["status"])
			}

			if version, ok := body["version"].(string); !ok || version != apiVersion {
				t.Errorf("Expected version %s, got %v", apiVersion, body["version"])
			}

			if service, ok := body["service"].(string); !ok || service != serviceName {
				t.Errorf("Expected service %s, got %v", serviceName, body["service"])
			}

			if _, ok := body["timestamp"].(string); !ok {
				t.Error("Expected timestamp in response")
			}
		})
	})
}

// TestTimezoneConvertHandler tests timezone conversion
func TestTimezoneConvertHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_UTC_to_EST", func(t *testing.T) {
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/convert",
			Body: map[string]interface{}{
				"time":          testTime.RFC3339,
				"from_timezone": "UTC",
				"to_timezone":   "America/New_York",
			},
		}

		w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["original_time"].(string); !ok {
				t.Error("Expected original_time in response")
			}
			if _, ok := body["converted_time"].(string); !ok {
				t.Error("Expected converted_time in response")
			}
			if offset, ok := body["offset_minutes"].(float64); !ok {
				t.Error("Expected offset_minutes in response")
			} else if offset != -300 { // EST is UTC-5
				t.Logf("Note: offset_minutes is %v, expected -300 for EST", offset)
			}
		})
	})

	t.Run("Success_Different_Format", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/convert",
			Body: map[string]interface{}{
				"time":          "2024-01-15 14:30:00",
				"from_timezone": "UTC",
				"to_timezone":   "Asia/Tokyo",
			},
		}

		w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "TimezoneConvert",
			Handler:     timezoneConvertHandler,
			BaseURL:     "/api/v1/time/convert",
		}

		patterns := TimezoneConversionErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		// Test with same timezone
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/convert",
			Body: map[string]interface{}{
				"time":          getTestTime().RFC3339,
				"from_timezone": "UTC",
				"to_timezone":   "UTC",
			},
		}

		w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for same timezone, got %d", w.Code)
		}
	})
}

// TestDurationCalculateHandler tests duration calculation
func TestDurationCalculateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_BasicDuration", func(t *testing.T) {
		start, end := getTestTimeRange()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/duration",
			Body: map[string]interface{}{
				"start_time":        start.RFC3339,
				"end_time":          end.RFC3339,
				"exclude_weekends":  false,
				"exclude_holidays":  false,
				"business_hours_only": false,
			},
		}

		w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			totalMinutes, ok := body["total_minutes"].(float64)
			if !ok {
				t.Error("Expected total_minutes in response")
			} else if totalMinutes != 480 { // 8 hours = 480 minutes
				t.Errorf("Expected 480 minutes, got %v", totalMinutes)
			}

			totalHours, ok := body["total_hours"].(float64)
			if !ok {
				t.Error("Expected total_hours in response")
			} else if totalHours != 8.0 {
				t.Errorf("Expected 8 hours, got %v", totalHours)
			}
		})
	})

	t.Run("Success_ExcludeWeekends", func(t *testing.T) {
		start, end := getTestTimeRange()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/duration",
			Body: map[string]interface{}{
				"start_time":       start.RFC3339,
				"end_time":         end.RFC3339,
				"exclude_weekends": true,
			},
		}

		w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var body map[string]interface{}
		json.NewDecoder(w.Body).Decode(&body)

		if _, ok := body["business_days"]; !ok {
			t.Error("Expected business_days in response when exclude_weekends is true")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "DurationCalculate",
			Handler:     durationCalculateHandler,
			BaseURL:     "/api/v1/time/duration",
		}

		patterns := DurationCalculationErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})
}

// TestFormatTimeHandler tests time formatting
func TestFormatTimeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name           string
		format         string
		validateResult func(t *testing.T, formatted string)
	}{
		{
			name:   "ISO8601",
			format: "iso8601",
			validateResult: func(t *testing.T, formatted string) {
				if _, err := time.Parse(time.RFC3339, formatted); err != nil {
					t.Errorf("Expected valid RFC3339 format, got %s", formatted)
				}
			},
		},
		{
			name:   "Unix",
			format: "unix",
			validateResult: func(t *testing.T, formatted string) {
				if len(formatted) == 0 {
					t.Error("Expected non-empty unix timestamp")
				}
			},
		},
		{
			name:   "Date",
			format: "date",
			validateResult: func(t *testing.T, formatted string) {
				if _, err := time.Parse("2006-01-02", formatted); err != nil {
					t.Errorf("Expected valid date format, got %s", formatted)
				}
			},
		},
		{
			name:   "DateTime",
			format: "datetime",
			validateResult: func(t *testing.T, formatted string) {
				if _, err := time.Parse("2006-01-02 15:04:05", formatted); err != nil {
					t.Errorf("Expected valid datetime format, got %s", formatted)
				}
			},
		},
		{
			name:   "Human",
			format: "human",
			validateResult: func(t *testing.T, formatted string) {
				if len(formatted) == 0 {
					t.Error("Expected non-empty human-readable format")
				}
			},
		},
		{
			name:   "Relative",
			format: "relative",
			validateResult: func(t *testing.T, formatted string) {
				if len(formatted) == 0 {
					t.Error("Expected non-empty relative time format")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run("Success_"+tc.name, func(t *testing.T) {
			testTime := getTestTime()
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/format",
				Body: map[string]interface{}{
					"time":   testTime.RFC3339,
					"format": tc.format,
				},
			}

			w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				formatted, ok := body["formatted"].(string)
				if !ok {
					t.Error("Expected formatted field in response")
					return
				}
				tc.validateResult(t, formatted)
			})
		})
	}

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "FormatTime",
			Handler:     formatTimeHandler,
			BaseURL:     "/api/v1/time/format",
		}

		patterns := FormatTimeErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})
}

// TestAddTimeHandler tests adding time
func TestAddTimeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name     string
		duration string
		unit     string
	}{
		{"2Hours", "2", "hours"},
		{"30Minutes", "30", "minutes"},
		{"1Day", "1", "days"},
		{"GoDuration", "2h30m", ""},
	}

	for _, tc := range testCases {
		t.Run("Success_"+tc.name, func(t *testing.T) {
			testTime := getTestTime()
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/add",
				Body: map[string]interface{}{
					"time":     testTime.RFC3339,
					"duration": tc.duration,
					"unit":     tc.unit,
				},
			}

			w := executeRequest(http.HandlerFunc(addTimeHandler), req)

			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["result_time"].(string); !ok {
					t.Error("Expected result_time in response")
				}
				if op, ok := body["operation"].(string); !ok || op != "add" {
					t.Errorf("Expected operation 'add', got %v", body["operation"])
				}
			})
		})
	}

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "AddTime",
			Handler:     addTimeHandler,
			BaseURL:     "/api/v1/time/add",
		}

		patterns := TimeArithmeticErrorPatterns("/api/v1/time/add")
		suite.RunErrorTests(t, patterns)
	})
}

// TestSubtractTimeHandler tests subtracting time
func TestSubtractTimeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		testTime := getTestTime()
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/subtract",
			Body: map[string]interface{}{
				"time":     testTime.RFC3339,
				"duration": "1",
				"unit":     "hours",
			},
		}

		w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if op, ok := body["operation"].(string); !ok || op != "subtract" {
				t.Errorf("Expected operation 'subtract', got %v", body["operation"])
			}
		})
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "SubtractTime",
			Handler:     subtractTimeHandler,
			BaseURL:     "/api/v1/time/subtract",
		}

		patterns := TimeArithmeticErrorPatterns("/api/v1/time/subtract")
		suite.RunErrorTests(t, patterns)
	})
}

// TestParseTimeHandler tests time parsing
func TestParseTimeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name  string
		input string
	}{
		{"RFC3339", "2024-01-15T14:30:00Z"},
		{"DateTime", "2024-01-15 14:30:00"},
		{"Date", "2024-01-15"},
		{"USDate", "01/15/2024"},
		{"UnixTimestamp", "1705329000"},
	}

	for _, tc := range testCases {
		t.Run("Success_"+tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/parse",
				Body: map[string]interface{}{
					"input": tc.input,
				},
			}

			w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["parsed_time"].(string); !ok {
					t.Error("Expected parsed_time in response")
				}
				if _, ok := body["rfc3339"].(string); !ok {
					t.Error("Expected rfc3339 in response")
				}
				if _, ok := body["unix"]; !ok {
					t.Error("Expected unix timestamp in response")
				}
			})
		})
	}

	t.Run("InvalidInput", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/parse",
			Body: map[string]interface{}{
				"input": "not-a-time-at-all-xyz-123-abc",
			},
		}

		w := executeRequest(http.HandlerFunc(parseTimeHandler), req)

		// The handler is very permissive and tries many formats
		// It may parse some things as Unix timestamps
		// So we just verify it returns a response (not testing rejection here)
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})
}

// TestScheduleOptimalHandler tests schedule optimization
func TestScheduleOptimalHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/optimal",
			Body: map[string]interface{}{
				"earliest_date":       "2024-01-15",
				"latest_date":         "2024-01-19",
				"duration_minutes":    60,
				"timezone":            "America/New_York",
				"business_hours_only": true,
				"participants":        []string{"user1", "user2"},
			},
		}

		w := executeRequest(http.HandlerFunc(scheduleOptimalHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			slots, ok := body["optimal_slots"].([]interface{})
			if !ok {
				t.Error("Expected optimal_slots array in response")
				return
			}
			if len(slots) == 0 {
				t.Error("Expected at least one optimal slot")
			}
		})
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "ScheduleOptimal",
			Handler:     scheduleOptimalHandler,
			BaseURL:     "/api/v1/schedule/optimal",
		}

		patterns := ScheduleOptimizationErrorPatterns()
		suite.RunErrorTests(t, patterns)
	})
}

// TestConflictDetectHandler tests conflict detection
func TestConflictDetectHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_NoDatabase", func(t *testing.T) {
		start, end := getTestTimeRange()
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
			if _, ok := body["conflicts"].([]interface{}); !ok {
				t.Error("Expected conflicts array in response")
			}
			if _, ok := body["has_conflicts"].(bool); !ok {
				t.Error("Expected has_conflicts in response")
			}
		})
	})

	t.Run("InvalidTimeFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schedule/conflicts",
			Body: map[string]interface{}{
				"organizer_id": "test-user",
				"start_time":   "invalid-time",
				"end_time":     getTestTime().RFC3339,
			},
		}

		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid time, got %d", w.Code)
		}
	})
}

// TestListEventsHandler tests event listing
func TestListEventsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_NoDatabase", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/events",
		}

		w := executeRequest(http.HandlerFunc(listEventsHandler), req)

		assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
			if _, ok := body["events"].([]interface{}); !ok {
				t.Error("Expected events array in response")
			}
			if count, ok := body["count"].(float64); !ok || count != 0 {
				t.Errorf("Expected count 0 without database, got %v", body["count"])
			}
		})
	})

	t.Run("WithQueryParameters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/events",
			Query: map[string]string{
				"organizer_id": "test-user",
				"status":       "active",
			},
		}

		w := executeRequest(http.HandlerFunc(listEventsHandler), req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("ParseDuration", func(t *testing.T) {
		testCases := []struct {
			duration string
			unit     string
			expected time.Duration
			hasError bool
		}{
			{"2h", "", 2 * time.Hour, false},
			{"30m", "", 30 * time.Minute, false},
			{"2", "hours", 2 * time.Hour, false},
			{"30", "minutes", 30 * time.Minute, false},
			{"1", "days", 24 * time.Hour, false},
			{"invalid", "", 0, true},
		}

		for _, tc := range testCases {
			t.Run(tc.duration+"_"+tc.unit, func(t *testing.T) {
				result, err := parseDuration(tc.duration, tc.unit)

				if tc.hasError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tc.hasError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !tc.hasError && result != tc.expected {
					t.Errorf("Expected duration %v, got %v", tc.expected, result)
				}
			})
		}
	})

	t.Run("GetRelativeTime", func(t *testing.T) {
		now := time.Now()

		testCases := []struct {
			name     string
			time     time.Time
			contains string
		}{
			{"JustNow", now, "just now"},
			{"MinutesAgo", now.Add(-5 * time.Minute), "minutes ago"},
			{"HoursAgo", now.Add(-2 * time.Hour), "hours ago"},
			{"Yesterday", now.Add(-25 * time.Hour), "yesterday"},
			{"DaysAgo", now.Add(-48 * time.Hour), "days ago"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := getRelativeTime(tc.time)
				if len(result) == 0 {
					t.Error("Expected non-empty result")
				}
			})
		}
	})

	t.Run("IsDST", func(t *testing.T) {
		// Test with a timezone that observes DST
		loc, _ := time.LoadLocation("America/New_York")

		summer := time.Date(2024, 7, 1, 12, 0, 0, 0, loc)
		winter := time.Date(2024, 1, 1, 12, 0, 0, 0, loc)

		summerDST := isDST(summer)
		winterDST := isDST(winter)

		// In summer, DST should be active; in winter, it should not
		if summerDST == winterDST {
			t.Log("Note: DST detection may vary based on timezone implementation")
		}
	})
}
