package main

import (
	"testing"
	"time"
)

// BenchmarkSuggestDates benchmarks date suggestion generation
func BenchmarkSuggestDates(b *testing.B) {
	req := DateSuggestionRequest{
		CoupleID:  "test-couple",
		DateType:  "romantic",
		BudgetMax: 150,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDynamicSuggestions(req)
	}
}

// BenchmarkSuggestDatesWithFiltering benchmarks with budget filtering
func BenchmarkSuggestDatesWithFiltering(b *testing.B) {
	req := DateSuggestionRequest{
		CoupleID:          "test-couple",
		DateType:          "adventure",
		BudgetMax:         100,
		WeatherPreference: "outdoor",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateDynamicSuggestions(req)
	}
}

// BenchmarkGenerateUUID benchmarks UUID generation
func BenchmarkGenerateUUID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateUUID()
	}
}

// TestPerformance_SuggestDatesResponseTime tests response time for suggestions
func TestPerformance_SuggestDatesResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	start := time.Now()
	req := DateSuggestionRequest{
		CoupleID:  "test",
		DateType:  "romantic",
		BudgetMax: 150,
	}
	suggestions := generateDynamicSuggestions(req)
	elapsed := time.Since(start)

	if len(suggestions) == 0 {
		t.Error("Expected suggestions")
	}

	// Should complete in under 10ms (very generous for in-memory generation)
	if elapsed > 10*time.Millisecond {
		t.Logf("Warning: Suggestion generation took %v (expected < 10ms)", elapsed)
	}
}

// TestPerformance_ConcurrentSuggestions tests concurrent request handling
func TestPerformance_ConcurrentSuggestions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	const concurrency = 10
	done := make(chan bool, concurrency)

	start := time.Now()
	for i := 0; i < concurrency; i++ {
		go func() {
			req := DateSuggestionRequest{
				CoupleID:  "test",
				DateType:  "romantic",
				BudgetMax: 150,
			}
			suggestions := generateDynamicSuggestions(req)
			if len(suggestions) == 0 {
				t.Error("Expected suggestions")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}
	elapsed := time.Since(start)

	// All should complete in under 100ms
	if elapsed > 100*time.Millisecond {
		t.Logf("Warning: %d concurrent requests took %v (expected < 100ms)", concurrency, elapsed)
	}
}

// TestPerformance_UUIDUniqueness tests UUID generation uniqueness
func TestPerformance_UUIDUniqueness(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	const count = 1000
	uuids := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		uuid := generateUUID()
		if uuids[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		uuids[uuid] = true
	}

	if len(uuids) != count {
		t.Errorf("Expected %d unique UUIDs, got %d", count, len(uuids))
	}
}
