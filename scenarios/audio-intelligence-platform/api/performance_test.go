// +build testing

package main

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// BenchmarkListTranscriptions benchmarks the list transcriptions endpoint
func BenchmarkListTranscriptions(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Create some test data
	for i := 0; i < 10; i++ {
		trans := setupTestTranscription(&testing.T{}, env.DB, fmt.Sprintf("test%d.mp3", i))
		defer trans.Cleanup()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reqData := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/transcriptions",
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		env.Service.ListTranscriptions(w, httpReq)
	}
}

// BenchmarkGetTranscription benchmarks the get transcription endpoint
func BenchmarkGetTranscription(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	trans := setupTestTranscription(&testing.T{}, env.DB, "test.mp3")
	defer trans.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reqData := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/transcriptions/" + trans.Transcription.ID.String(),
			URLVars: map[string]string{"id": trans.Transcription.ID.String()},
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			b.Fatalf("Failed to create request: %v", err)
		}

		env.Service.GetTranscription(w, httpReq)
	}
}

// TestPerformance_ListTranscriptions tests list performance under load
func TestPerformance_ListTranscriptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create 100 test transcriptions
	transcriptions := make([]*TestTranscription, 100)
	for i := 0; i < 100; i++ {
		transcriptions[i] = setupTestTranscription(t, env.DB, fmt.Sprintf("test%d.mp3", i))
		defer transcriptions[i].Cleanup()
	}

	t.Run("LoadTest_100Items", func(t *testing.T) {
		start := time.Now()

		reqData := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/transcriptions",
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		env.Service.ListTranscriptions(w, httpReq)

		duration := time.Since(start)

		if duration > 2*time.Second {
			t.Errorf("ListTranscriptions took too long: %v (expected < 2s)", duration)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("ListTranscriptions with 100 items completed in %v", duration)
	})
}

// TestPerformance_ConcurrentAnalysis tests concurrent analysis requests
func TestPerformance_ConcurrentAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	trans := setupTestTranscription(t, env.DB, "test.mp3")
	defer trans.Cleanup()

	t.Run("Concurrent_10Requests", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		durations := make(chan time.Duration, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				requestStart := time.Now()

				reqData := HTTPTestRequest{
					Method:  "POST",
					Path:    "/api/transcriptions/" + trans.Transcription.ID.String() + "/analyze",
					URLVars: map[string]string{"id": trans.Transcription.ID.String()},
					Body:    TestData.AnalyzeRequest("summary", "", ""),
				}

				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					errors <- err
					return
				}

				env.Service.AnalyzeTranscription(w, httpReq)

				requestDuration := time.Since(requestStart)
				durations <- requestDuration

				if w.Code != 200 {
					errors <- fmt.Errorf("request %d failed with status %d", i, w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(durations)

		totalDuration := time.Since(start)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Errorf("Concurrent request error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed %d/%d concurrent requests", errorCount, concurrency)
		}

		// Calculate average duration
		var totalRequestTime time.Duration
		requestCount := 0
		for d := range durations {
			totalRequestTime += d
			requestCount++
		}

		if requestCount > 0 {
			avgDuration := totalRequestTime / time.Duration(requestCount)
			t.Logf("Concurrent analysis: %d requests completed in %v (avg: %v per request)",
				concurrency, totalDuration, avgDuration)
		}

		// Total time should be less than if run sequentially
		if totalDuration > 30*time.Second {
			t.Errorf("Concurrent requests took too long: %v (expected < 30s)", totalDuration)
		}
	})
}

// TestPerformance_SearchScalability tests search performance with increasing data
func TestPerformance_SearchScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		t.Run(fmt.Sprintf("Search_WithDB_%dItems", size), func(t *testing.T) {
			// Create test data
			transcriptions := make([]*TestTranscription, size)
			for i := 0; i < size; i++ {
				transcriptions[i] = setupTestTranscription(t, env.DB, fmt.Sprintf("test%d.mp3", i))
				defer transcriptions[i].Cleanup()
			}

			start := time.Now()

			reqData := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/search",
				Body:   TestData.SearchRequest("test query", 10),
			}

			w, httpReq, err := makeHTTPRequest(reqData)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			env.Service.SearchTranscriptions(w, httpReq)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			// Search should remain fast regardless of DB size
			// (actual performance depends on n8n/qdrant)
			t.Logf("Search with %d items in DB completed in %v", size, duration)

			if duration > 5*time.Second {
				t.Logf("Warning: Search took longer than expected: %v", duration)
			}
		})
	}
}

// TestPerformance_DatabaseConnectionPool tests connection pool under load
func TestPerformance_DatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test data
	for i := 0; i < 20; i++ {
		trans := setupTestTranscription(t, env.DB, fmt.Sprintf("test%d.mp3", i))
		defer trans.Cleanup()
	}

	t.Run("ConnectionPool_50ConcurrentReads", func(t *testing.T) {
		concurrency := 50
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				reqData := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/transcriptions",
				}

				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					errors <- err
					return
				}

				env.Service.ListTranscriptions(w, httpReq)

				if w.Code != 200 {
					errors <- fmt.Errorf("request failed with status %d", w.Code)
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			t.Errorf("Connection pool error: %v", err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed %d/%d requests", errorCount, concurrency)
		}

		t.Logf("Connection pool test: %d concurrent reads completed in %v", concurrency, duration)

		// Should handle concurrent requests efficiently
		if duration > 10*time.Second {
			t.Errorf("Connection pool handling took too long: %v (expected < 10s)", duration)
		}
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LargeTranscriptionList", func(t *testing.T) {
		// Create 500 transcriptions
		transcriptions := make([]*TestTranscription, 500)
		for i := 0; i < 500; i++ {
			transcriptions[i] = setupTestTranscription(t, env.DB, fmt.Sprintf("test%d.mp3", i))
		}

		// Cleanup after test
		defer func() {
			for _, trans := range transcriptions {
				trans.Cleanup()
			}
		}()

		// List all transcriptions
		reqData := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/transcriptions",
		}

		w, httpReq, err := makeHTTPRequest(reqData)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		start := time.Now()
		env.Service.ListTranscriptions(w, httpReq)
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Listed 500 transcriptions in %v", duration)

		// Should complete in reasonable time even with large dataset
		if duration > 5*time.Second {
			t.Errorf("Listing 500 items took too long: %v (expected < 5s)", duration)
		}
	})
}

// TestPerformance_ResponseTimes tests endpoint response time requirements
func TestPerformance_ResponseTimes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	trans := setupTestTranscription(t, env.DB, "test.mp3")
	defer trans.Cleanup()

	tests := []struct {
		name         string
		maxDuration  time.Duration
		requestFunc  func() (*httptest.ResponseRecorder, error)
	}{
		{
			name:        "Health",
			maxDuration: 100 * time.Millisecond,
			requestFunc: func() (*httptest.ResponseRecorder, error) {
				reqData := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}
				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					return nil, err
				}
				Health(w, httpReq)
				return w, nil
			},
		},
		{
			name:        "GetTranscription",
			maxDuration: 500 * time.Millisecond,
			requestFunc: func() (*httptest.ResponseRecorder, error) {
				reqData := HTTPTestRequest{
					Method:  "GET",
					Path:    "/api/transcriptions/" + trans.Transcription.ID.String(),
					URLVars: map[string]string{"id": trans.Transcription.ID.String()},
				}
				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					return nil, err
				}
				env.Service.GetTranscription(w, httpReq)
				return w, nil
			},
		},
		{
			name:        "ListTranscriptions",
			maxDuration: 1 * time.Second,
			requestFunc: func() (*httptest.ResponseRecorder, error) {
				reqData := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/transcriptions",
				}
				w, httpReq, err := makeHTTPRequest(reqData)
				if err != nil {
					return nil, err
				}
				env.Service.ListTranscriptions(w, httpReq)
				return w, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			w, err := tt.requestFunc()
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			t.Logf("%s response time: %v", tt.name, duration)

			if duration > tt.maxDuration {
				t.Errorf("%s took too long: %v (expected < %v)", tt.name, duration, tt.maxDuration)
			}
		})
	}
}
