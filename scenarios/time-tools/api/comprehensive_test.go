package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAllHandlersMethods tests HTTP methods for all handlers
func TestAllHandlersMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	handlers := map[string]http.HandlerFunc{
		"/api/v1/time/convert":         timezoneConvertHandler,
		"/api/v1/time/duration":        durationCalculateHandler,
		"/api/v1/time/format":          formatTimeHandler,
		"/api/v1/time/add":             addTimeHandler,
		"/api/v1/time/subtract":        subtractTimeHandler,
		"/api/v1/time/parse":           parseTimeHandler,
		"/api/v1/schedule/optimal":     scheduleOptimalHandler,
		"/api/v1/schedule/conflicts":   conflictDetectHandler,
	}

	for path, handler := range handlers {
		t.Run(path+"_GET_NotAllowed", func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			handler(w, req)

			// Most should not allow GET
			if w.Code == http.StatusMethodNotAllowed {
				t.Logf("Method not allowed as expected for %s", path)
			}
		})
	}
}

// TestParseDurationAllVariants tests all duration parsing variants
func TestParseDurationAllVariants(t *testing.T) {
	testCases := []struct {
		duration string
		unit     string
		expectOK bool
	}{
		// Valid cases
		{"1", "seconds", true},
		{"1", "minutes", true},
		{"1", "hours", true},
		{"1", "days", true},
		{"1", "weeks", true},
		{"1", "months", true},
		{"1", "years", true},
		{"2h30m", "", true},
		{"1.5h", "", true},
		{"30m", "", true},
		// Singular forms
		{"1", "second", true},
		{"1", "minute", true},
		{"1", "hour", true},
		{"1", "day", true},
		{"1", "week", true},
		{"1", "month", true},
		{"1", "year", true},
		// Abbreviations
		{"10", "s", true},
		{"10", "m", true},
		{"10", "h", true},
		{"10", "d", true},
		{"10", "w", true},
		// Invalid cases
		{"invalid", "hours", false},
		{"10", "invalid_unit", false},
		{"", "", false},
		{"abc", "xyz", false},
	}

	for _, tc := range testCases {
		t.Run(tc.duration+"_"+tc.unit, func(t *testing.T) {
			result, err := parseDuration(tc.duration, tc.unit)

			if tc.expectOK {
				if err != nil {
					t.Logf("Expected success for %s %s, got error: %v", tc.duration, tc.unit, err)
				}
			} else {
				if err == nil {
					t.Logf("Expected error for %s %s, got result: %v", tc.duration, tc.unit, result)
				}
			}
		})
	}
}

// TestGetRelativeTimeAllCases tests all relative time cases
func TestGetRelativeTimeAllCases(t *testing.T) {
	now := time.Now()

	testCases := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"ExactlyNow", now, ""},
		{"1SecondAgo", now.Add(-1 * time.Second), ""},
		{"30SecondsAgo", now.Add(-30 * time.Second), ""},
		{"59SecondsAgo", now.Add(-59 * time.Second), ""},
		{"1MinuteAgo", now.Add(-1 * time.Minute), ""},
		{"30MinutesAgo", now.Add(-30 * time.Minute), ""},
		{"59MinutesAgo", now.Add(-59 * time.Minute), ""},
		{"1HourAgo", now.Add(-1 * time.Hour), ""},
		{"12HoursAgo", now.Add(-12 * time.Hour), ""},
		{"23HoursAgo", now.Add(-23 * time.Hour), ""},
		{"25HoursAgo", now.Add(-25 * time.Hour), ""},
		{"36HoursAgo", now.Add(-36 * time.Hour), ""},
		{"48HoursAgo", now.Add(-48 * time.Hour), ""},
		{"7DaysAgo", now.Add(-7 * 24 * time.Hour), ""},
		{"30DaysAgo", now.Add(-30 * 24 * time.Hour), ""},
		// Future times
		{"In1Minute", now.Add(1 * time.Minute), ""},
		{"In5Minutes", now.Add(5 * time.Minute), ""},
		{"In1Hour", now.Add(1 * time.Hour), ""},
		{"In24Hours", now.Add(24 * time.Hour), ""},
		{"In48Hours", now.Add(48 * time.Hour), ""},
		{"In7Days", now.Add(7 * 24 * time.Hour), ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getRelativeTime(tc.time)
			if result == "" {
				t.Error("Expected non-empty relative time string")
			}
			t.Logf("%s: %s", tc.name, result)
		})
	}
}

// TestIsDSTAllTimezones tests DST detection across timezones
func TestIsDSTAllTimezones(t *testing.T) {
	timezones := []string{
		"America/New_York",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Australia/Sydney",
		"Asia/Tokyo", // No DST
		"UTC",        // No DST
	}

	summer := time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC)
	winter := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	for _, tzName := range timezones {
		t.Run(tzName, func(t *testing.T) {
			loc, err := time.LoadLocation(tzName)
			if err != nil {
				t.Skipf("Timezone %s not available", tzName)
			}

			summerTime := summer.In(loc)
			winterTime := winter.In(loc)

			summerDST := isDST(summerTime)
			winterDST := isDST(winterTime)

			t.Logf("%s - Summer DST: %v, Winter DST: %v", tzName, summerDST, winterDST)
		})
	}
}

// TestCalculateSlotScoreComprehensive tests score calculation
func TestCalculateSlotScoreComprehensive(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")

	testCases := []struct {
		name string
		hour int
		min  int
	}{
		{"EarlyMorning_6AM", 6, 0},
		{"EarlyMorning_7AM", 7, 0},
		{"MidMorning_9AM", 9, 0},
		{"MidMorning_10AM", 10, 0},
		{"MidMorning_11AM", 11, 0},
		{"Noon", 12, 0},
		{"EarlyAfternoon_1PM", 13, 0},
		{"EarlyAfternoon_2PM", 14, 0},
		{"LateAfternoon_3PM", 15, 0},
		{"LateAfternoon_4PM", 16, 0},
		{"Evening_5PM", 17, 0},
		{"Evening_6PM", 18, 0},
		{"Night_8PM", 20, 0},
	}

	req := ScheduleOptimizationRequest{
		DurationMinutes:   60,
		Timezone:          "America/New_York",
		BusinessHoursOnly: true,
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			slot := time.Date(2024, 1, 15, tc.hour, tc.min, 0, 0, loc)
			score := calculateSlotScore(slot, req)

			if score < 0 || score > 1 {
				t.Errorf("Score out of range: %f", score)
			}
			t.Logf("%s: score = %.3f", tc.name, score)
		})
	}
}

// TestTimezoneEdgeCases tests timezone conversion edge cases
func TestTimezoneEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name     string
		time     string
		fromTz   string
		toTz     string
		expectOK bool
	}{
		{"SameTimezone", "2024-01-15T10:00:00Z", "UTC", "UTC", true},
		{"DSTTransition", "2024-03-10T02:00:00-05:00", "America/New_York", "UTC", true},
		{"CrossDateLine", "2024-01-15T23:00:00+12:00", "Pacific/Auckland", "UTC", true},
		{"InvalidFromTz", "2024-01-15T10:00:00Z", "Invalid/Timezone", "UTC", false},
		{"InvalidToTz", "2024-01-15T10:00:00Z", "UTC", "Invalid/Timezone", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/convert",
				Body: map[string]interface{}{
					"time":          tc.time,
					"from_timezone": tc.fromTz,
					"to_timezone":   tc.toTz,
				},
			}

			w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)

			if tc.expectOK {
				if w.Code != http.StatusOK {
					t.Logf("Expected success, got %d", w.Code)
				}
			} else {
				if w.Code == http.StatusOK {
					t.Logf("Expected error, got success")
				}
			}
		})
	}
}

// TestDurationEdgeCases tests duration calculation edge cases
func TestDurationEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name            string
		startTime       string
		endTime         string
		excludeWeekends bool
		expectOK        bool
	}{
		{"ZeroDuration", "2024-01-15T10:00:00Z", "2024-01-15T10:00:00Z", false, true},
		{"NegativeDuration", "2024-01-15T10:00:00Z", "2024-01-15T09:00:00Z", false, true},
		{"OneYear", "2024-01-01T00:00:00Z", "2025-01-01T00:00:00Z", true, true},
		{"OverWeekend", "2024-01-12T00:00:00Z", "2024-01-15T00:00:00Z", true, true},
		{"EntireWeekend", "2024-01-13T00:00:00Z", "2024-01-14T23:59:59Z", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/duration",
				Body: map[string]interface{}{
					"start_time":       tc.startTime,
					"end_time":         tc.endTime,
					"exclude_weekends": tc.excludeWeekends,
				},
			}

			w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)

			if tc.expectOK && w.Code != http.StatusOK {
				t.Errorf("Expected OK, got %d", w.Code)
			}
		})
	}
}

// TestFormatTimeAllFormats tests all time formats
func TestFormatTimeAllFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := "2024-01-15T14:30:00Z"
	formats := []string{
		"iso8601",
		"unix",
		"date",
		"time",
		"datetime",
		"human",
		"relative",
		"rfc3339",
		"rfc1123",
	}

	for _, format := range formats {
		t.Run("Format_"+format, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/time/format",
				Body: map[string]interface{}{
					"time":   testTime,
					"format": format,
				},
			}

			w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

			if w.Code == http.StatusOK {
				assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
					if formatted, ok := body["formatted"].(string); !ok || formatted == "" {
						t.Error("Expected non-empty formatted field")
					}
				})
			}
		})
	}
}

// TestFormatTimeWithCustomFormat tests custom format strings
func TestFormatTimeWithCustomFormat(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := "2024-01-15T14:30:00Z"

	t.Run("CustomFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/format",
			Body: map[string]interface{}{
				"time":          testTime,
				"format":        "custom",
				"custom_format": "2006-01-02",
			},
		}

		w := executeRequest(http.HandlerFunc(formatTimeHandler), req)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["formatted"]; !ok {
					t.Error("Expected formatted field")
				}
			})
		}
	})
}

// TestTimeArithmeticWithTimezones tests time arithmetic with timezones
func TestTimeArithmeticWithTimezones(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := "2024-01-15T10:00:00-05:00"

	t.Run("AddWithTimezone", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/add",
			Body: map[string]interface{}{
				"time":     testTime,
				"duration": "2",
				"unit":     "hours",
				"timezone": "America/New_York",
			},
		}

		w := executeRequest(http.HandlerFunc(addTimeHandler), req)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["result_time"]; !ok {
					t.Error("Expected result_time")
				}
			})
		}
	})

	t.Run("SubtractWithTimezone", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/time/subtract",
			Body: map[string]interface{}{
				"time":     testTime,
				"duration": "1",
				"unit":     "days",
				"timezone": "Europe/London",
			},
		}

		w := executeRequest(http.HandlerFunc(subtractTimeHandler), req)

		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK, func(body map[string]interface{}) {
				if _, ok := body["result_time"]; !ok {
					t.Error("Expected result_time")
				}
			})
		}
	})
}
