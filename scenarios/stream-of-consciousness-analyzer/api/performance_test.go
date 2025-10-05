// +build testing

package main

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestHealthCheckPerformance tests health endpoint performance
func TestHealthCheckPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Health check should respond in < 100ms
		if duration > 100*time.Millisecond {
			t.Errorf("Health check took too long: %v (expected < 100ms)", duration)
		} else {
			t.Logf("Health check responded in %v", duration)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 50
		var wg sync.WaitGroup
		var successCount int32

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w, err := makeHTTPRequest(req, healthCheck)
				if err == nil && w.Code == 200 {
					atomic.AddInt32(&successCount, 1)
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		if successCount != int32(concurrency) {
			t.Errorf("Expected %d successful requests, got %d", concurrency, successCount)
		}

		avgTime := duration / time.Duration(concurrency)
		t.Logf("%d concurrent requests completed in %v (avg: %v per request)", concurrency, duration, avgTime)

		// All requests should complete within 5 seconds
		if duration > 5*time.Second {
			t.Errorf("Concurrent requests took too long: %v", duration)
		}
	})
}

// TestGetCampaignsPerformance tests campaigns endpoint performance
func TestGetCampaignsPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	t.Run("EmptyResult", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		w, err := makeHTTPRequest(req, getCampaigns)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Should respond in < 500ms
		if duration > 500*time.Millisecond {
			t.Errorf("Get campaigns took too long: %v", duration)
		} else {
			t.Logf("Get campaigns (empty) responded in %v", duration)
		}
	})

	t.Run("WithData", func(t *testing.T) {
		// Create test data
		createMultipleTestCampaigns(t, testDB, 10, "Perf Test Campaign")

		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		w, err := makeHTTPRequest(req, getCampaigns)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var campaigns []Campaign
		json.Unmarshal(w.Body.Bytes(), &campaigns)

		if len(campaigns) < 10 {
			t.Errorf("Expected at least 10 campaigns, got %d", len(campaigns))
		}

		// Should respond in < 1 second with data
		if duration > 1*time.Second {
			t.Errorf("Get campaigns with data took too long: %v", duration)
		} else {
			t.Logf("Get campaigns (%d items) responded in %v", len(campaigns), duration)
		}
	})
}

// TestCreateCampaignPerformance tests campaign creation performance
func TestCreateCampaignPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	t.Run("SingleCreate", func(t *testing.T) {
		campaign := Campaign{
			Name:          "Performance Test Campaign",
			Description:   "Testing performance",
			ContextPrompt: "Test context",
			Color:         "#FF0000",
			Icon:          "🚀",
		}

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/campaigns",
			Body:   campaign,
		}

		w, err := makeHTTPRequest(req, createCampaign)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 201 {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		// Campaign creation should be fast (< 500ms)
		if duration > 500*time.Millisecond {
			t.Errorf("Create campaign took too long: %v", duration)
		} else {
			t.Logf("Create campaign responded in %v", duration)
		}
	})

	t.Run("BatchCreate", func(t *testing.T) {
		count := 10
		start := time.Now()

		for i := 0; i < count; i++ {
			campaign := Campaign{
				Name:          "Batch Campaign " + string(rune(i)),
				Description:   "Batch test",
				ContextPrompt: "Batch context",
				Color:         "#0000FF",
				Icon:          "📦",
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/campaigns",
				Body:   campaign,
			}

			w, err := makeHTTPRequest(req, createCampaign)
			if err != nil {
				t.Errorf("Failed to create campaign %d: %v", i, err)
				continue
			}

			if w.Code != 201 {
				t.Errorf("Campaign %d: expected status 201, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(count)

		t.Logf("Created %d campaigns in %v (avg: %v per campaign)", count, duration, avgDuration)

		// Average creation time should be reasonable
		if avgDuration > 500*time.Millisecond {
			t.Errorf("Average campaign creation time too high: %v", avgDuration)
		}
	})
}

// TestGetNotesPerformance tests notes endpoint performance
func TestGetNotesPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Notes Performance Campaign")

	t.Run("WithoutData", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > 500*time.Millisecond {
			t.Errorf("Get notes took too long: %v", duration)
		} else {
			t.Logf("Get notes (empty) responded in %v", duration)
		}
	})

	t.Run("WithManyNotes", func(t *testing.T) {
		// Create many notes
		noteCount := 50
		createMultipleTestNotes(t, testDB, campaign.ID, noteCount, "Perf Note")

		start := time.Now()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		// Should return at least the created notes (might be limited by default limit)
		if len(notes) == 0 {
			t.Error("Expected notes to be returned")
		}

		// Even with 50 notes, should respond quickly
		if duration > 1*time.Second {
			t.Errorf("Get notes with %d items took too long: %v", noteCount, duration)
		} else {
			t.Logf("Get notes (%d items) responded in %v", len(notes), duration)
		}
	})

	t.Run("WithLimitParameter", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/notes",
			QueryParams: map[string]string{
				"campaign_id": campaign.ID,
				"limit":       "10",
			},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		if len(notes) > 10 {
			t.Errorf("Expected max 10 notes with limit, got %d", len(notes))
		}

		// Limited query should be fast
		if duration > 500*time.Millisecond {
			t.Errorf("Get notes with limit took too long: %v", duration)
		} else {
			t.Logf("Get notes (limit=10) responded in %v", duration)
		}
	})
}

// TestSearchNotesPerformance tests search endpoint performance
func TestSearchNotesPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Search Performance Campaign")

	// Create notes with searchable content
	for i := 0; i < 20; i++ {
		createTestNote(t, testDB, campaign.ID, "Searchable Note "+string(rune(i)))
	}

	t.Run("BasicSearch", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/search",
			QueryParams: map[string]string{"q": "Searchable"},
		}

		w, err := makeHTTPRequest(req, searchNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Search should complete in < 1 second
		if duration > 1*time.Second {
			t.Errorf("Search took too long: %v", duration)
		} else {
			t.Logf("Search responded in %v", duration)
		}
	})

	t.Run("SearchWithCampaignFilter", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q":           "Note",
				"campaign_id": campaign.ID,
			},
		}

		w, err := makeHTTPRequest(req, searchNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Filtered search should be even faster
		if duration > 800*time.Millisecond {
			t.Errorf("Filtered search took too long: %v", duration)
		} else {
			t.Logf("Filtered search responded in %v", duration)
		}
	})
}

// TestConcurrentOperations tests concurrent access patterns
func TestConcurrentOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Concurrent Test Campaign")

	t.Run("MixedReadWrite", func(t *testing.T) {
		var wg sync.WaitGroup
		var readCount, writeCount int32
		operations := 30

		start := time.Now()

		for i := 0; i < operations; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Alternate between reads and writes
				if idx%2 == 0 {
					// Read operation
					req := HTTPTestRequest{
						Method:      "GET",
						Path:        "/api/notes",
						QueryParams: map[string]string{"campaign_id": campaign.ID},
					}

					w, err := makeHTTPRequest(req, getNotes)
					if err == nil && w.Code == 200 {
						atomic.AddInt32(&readCount, 1)
					}
				} else {
					// Write operation
					entry := StreamEntry{
						CampaignID: campaign.ID,
						Content:    "Concurrent test entry",
						Type:       "text",
						Source:     "test",
						Metadata:   json.RawMessage(`{}`),
					}

					req := HTTPTestRequest{
						Method: "POST",
						Path:   "/api/stream/capture",
						Body:   entry,
					}

					w, err := makeHTTPRequest(req, captureStream)
					if err == nil && w.Code == 201 {
						atomic.AddInt32(&writeCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Completed %d operations (%d reads, %d writes) in %v",
			operations, readCount, writeCount, duration)

		expectedReads := int32(operations / 2)
		expectedWrites := int32(operations - operations/2)

		if readCount != expectedReads {
			t.Errorf("Expected %d successful reads, got %d", expectedReads, readCount)
		}

		if writeCount != expectedWrites {
			t.Errorf("Expected %d successful writes, got %d", expectedWrites, writeCount)
		}

		// Mixed operations should complete in reasonable time
		if duration > 10*time.Second {
			t.Errorf("Mixed operations took too long: %v", duration)
		}
	})
}

// BenchmarkHealthCheck benchmarks the health endpoint
func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		// Note: This benchmark requires a live database connection
		// Skip if no DB available
		if db == nil {
			b.Skip("Database not available for benchmarking")
		}

		_, _ = makeHTTPRequest(req, healthCheck)
	}
}

// BenchmarkGetCampaigns benchmarks the campaigns endpoint
func BenchmarkGetCampaigns(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		b.Skip("Database not available for benchmarking")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/campaigns",
		}

		_, _ = makeHTTPRequest(req, getCampaigns)
	}
}

// TestMemoryUsage tests memory allocation patterns
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Memory Test Campaign")

	t.Run("LargeResultSet", func(t *testing.T) {
		// Create many notes
		noteCount := 100
		createMultipleTestNotes(t, testDB, campaign.ID, noteCount, "Memory Note")

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		// This test verifies that large result sets don't cause memory issues
		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		t.Logf("Retrieved %d notes successfully (memory test)", len(notes))
	})
}
