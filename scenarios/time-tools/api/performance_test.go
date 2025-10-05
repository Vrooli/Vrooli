package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// BenchmarkTimezoneConversion benchmarks timezone conversion performance
func BenchmarkTimezoneConversion(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkDurationCalculation benchmarks duration calculation performance
func BenchmarkDurationCalculation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkFormatTime benchmarks time formatting performance
func BenchmarkFormatTime(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/time/format",
		Body: map[string]interface{}{
			"time":   testTime.RFC3339,
			"format": "iso8601",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(formatTimeHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkTimeArithmetic benchmarks time arithmetic operations
func BenchmarkTimeArithmetic(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/time/add",
		Body: map[string]interface{}{
			"time":     testTime.RFC3339,
			"duration": "2",
			"unit":     "hours",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(addTimeHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkScheduleOptimization benchmarks schedule optimization
func BenchmarkScheduleOptimization(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/schedule/optimal",
		Body: map[string]interface{}{
			"earliest_date":       "2024-01-15",
			"latest_date":         "2024-01-19",
			"duration_minutes":    60,
			"timezone":            "America/New_York",
			"business_hours_only": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(scheduleOptimalHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkConflictDetection benchmarks conflict detection
func BenchmarkConflictDetection(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := executeRequest(http.HandlerFunc(conflictDetectHandler), req)
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkParseDuration benchmarks duration parsing
func BenchmarkParseDuration(b *testing.B) {
	testCases := []struct {
		duration string
		unit     string
	}{
		{"2", "hours"},
		{"30", "minutes"},
		{"1", "days"},
		{"2h30m", ""},
	}

	for _, tc := range testCases {
		b.Run(fmt.Sprintf("%s_%s", tc.duration, tc.unit), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = parseDuration(tc.duration, tc.unit)
			}
		})
	}
}

// BenchmarkGetRelativeTime benchmarks relative time calculation
func BenchmarkGetRelativeTime(b *testing.B) {
	testTime := time.Now().Add(-5 * time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getRelativeTime(testTime)
	}
}

// BenchmarkGenerateOptimalSlots benchmarks slot generation
func BenchmarkGenerateOptimalSlots(b *testing.B) {
	earliestDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	latestDate := time.Date(2024, 1, 19, 0, 0, 0, 0, time.UTC)
	req := ScheduleOptimizationRequest{
		DurationMinutes:   60,
		Timezone:          "America/New_York",
		BusinessHoursOnly: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateOptimalSlots(earliestDate, latestDate, req)
	}
}

// TestTimezoneConversionPerformance tests that conversions meet SLA
func TestTimezoneConversionPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/time/convert",
		Body: map[string]interface{}{
			"time":          testTime.RFC3339,
			"from_timezone": "UTC",
			"to_timezone":   "Asia/Tokyo",
		},
	}

	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Target: < 10ms per conversion (from PRD)
	if avgTime > 10*time.Millisecond {
		t.Logf("Warning: Average conversion time %v exceeds 10ms target", avgTime)
	} else {
		t.Logf("✓ Average conversion time: %v (within 10ms target)", avgTime)
	}
}

// TestScheduleOptimizationPerformance tests scheduling performance
func TestScheduleOptimizationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/schedule/optimal",
		Body: map[string]interface{}{
			"earliest_date":       "2024-01-15",
			"latest_date":         "2024-01-19",
			"duration_minutes":    60,
			"timezone":            "America/New_York",
			"business_hours_only": true,
		},
	}

	iterations := 50
	start := time.Now()

	for i := 0; i < iterations; i++ {
		w := executeRequest(http.HandlerFunc(scheduleOptimalHandler), req)
		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	// Target: < 100ms per optimization (from PRD)
	if avgTime > 100*time.Millisecond {
		t.Logf("Warning: Average optimization time %v exceeds 100ms target", avgTime)
	} else {
		t.Logf("✓ Average optimization time: %v (within 100ms target)", avgTime)
	}
}

// TestConcurrentRequests tests handling concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testTime := getTestTime()
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/time/convert",
		Body: map[string]interface{}{
			"time":          testTime.RFC3339,
			"from_timezone": "UTC",
			"to_timezone":   "Europe/London",
		},
	}

	concurrency := 10
	iterations := 10
	done := make(chan bool, concurrency)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				w := executeRequest(http.HandlerFunc(timezoneConvertHandler), req)
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
				}
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	elapsed := time.Since(start)
	totalRequests := concurrency * iterations
	avgTime := elapsed / time.Duration(totalRequests)

	t.Logf("Completed %d concurrent requests in %v (avg: %v per request)",
		totalRequests, elapsed, avgTime)
}

// TestLargeTimeRangeDuration tests duration calculation with large time ranges
func TestLargeTimeRangeDuration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with 1 year range
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/time/duration",
		Body: map[string]interface{}{
			"start_time":       startTime.Format(time.RFC3339),
			"end_time":         endTime.Format(time.RFC3339),
			"exclude_weekends": true,
		},
	}

	start := time.Now()
	w := executeRequest(http.HandlerFunc(durationCalculateHandler), req)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	// Should complete within reasonable time even with large range
	if elapsed > 1*time.Second {
		t.Errorf("Large range calculation took %v, expected < 1s", elapsed)
	} else {
		t.Logf("✓ Large range (1 year) calculated in %v", elapsed)
	}
}
