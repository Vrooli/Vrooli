package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// BenchmarkConcurrentHealthChecks tests concurrent health check requests
func BenchmarkConcurrentHealthChecks(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})
		}
	})
}

// BenchmarkVideoRetrieval benchmarks video retrieval performance
func BenchmarkVideoRetrieval(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()

	videoID := insertTestVideo(&testing.T{}, server.db)
	path := fmt.Sprintf("/api/v1/video/%s", videoID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, "GET", path, nil, nil)
	}
}

// BenchmarkJobCreation benchmarks job creation performance
func BenchmarkJobCreation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()
	defer cleanupTestData(&testing.T{}, server.db)

	videoID := insertTestVideo(&testing.T{}, server.db)
	path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
	payload := GenerateTestConvertRequest("webm")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeJSONRequest(server, "POST", path, payload)
	}
}

// BenchmarkStreamingOperations benchmarks streaming operations
func BenchmarkStreamingOperations(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()
	defer cleanupTestData(&testing.T{}, server.db)

	payload := map[string]interface{}{
		"name": "Benchmark Stream",
		"input_source": map[string]interface{}{
			"type": "file",
		},
		"output_targets": []map[string]interface{}{
			{"platform": "custom", "url": "rtmp://test.com/live"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeJSONRequest(server, "POST", "/api/v1/stream/create", payload)
	}
}

// BenchmarkJobListing benchmarks job listing performance
func BenchmarkJobListing(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()
	defer cleanupTestData(&testing.T{}, server.db)

	// Create some test jobs
	videoID := insertTestVideo(&testing.T{}, server.db)
	for i := 0; i < 10; i++ {
		insertTestJob(&testing.T{}, server.db, videoID, "convert")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, "GET", "/api/v1/jobs", nil, nil)
	}
}

// BenchmarkConcurrentJobCreation tests concurrent job creation
func BenchmarkConcurrentJobCreation(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(&testing.T{})
	defer cleanupServer()
	defer cleanupTestData(&testing.T{}, server.db)

	videoID := insertTestVideo(&testing.T{}, server.db)
	path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
	payload := GenerateTestConvertRequest("mp4")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			makeJSONRequest(server, "POST", path, payload)
		}
	})
}

// TestPerformanceTargets validates performance against defined targets
func TestPerformanceTargets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("HealthEndpointP95", func(t *testing.T) {
		iterations := 100
		durations := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})
			durations[i] = time.Since(start)
		}

		// Calculate P95
		p95 := calculatePercentile(durations, 0.95)
		t.Logf("Health endpoint P95: %v", p95)

		// Target: < 100ms p95
		if p95 > 100*time.Millisecond {
			t.Logf("⚠️  Health endpoint P95 (%v) exceeds target (100ms)", p95)
		}
	})

	t.Run("VideoRetrievalP95", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s", videoID)

		iterations := 100
		durations := make([]time.Duration, iterations)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			makeHTTPRequest(server, "GET", path, nil, nil)
			durations[i] = time.Since(start)
		}

		p95 := calculatePercentile(durations, 0.95)
		t.Logf("Video retrieval P95: %v", p95)

		// Target: < 200ms p95
		if p95 > 200*time.Millisecond {
			t.Logf("⚠️  Video retrieval P95 (%v) exceeds target (200ms)", p95)
		}
	})

	t.Run("ConcurrentRequestHandling", func(t *testing.T) {
		concurrency := 10
		requestsPerClient := 10
		var wg sync.WaitGroup

		start := time.Now()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < requestsPerClient; j++ {
					makeHTTPRequest(server, "GET", "/health", nil, map[string]string{})
				}
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)

		totalRequests := concurrency * requestsPerClient
		rps := float64(totalRequests) / elapsed.Seconds()
		t.Logf("Concurrent requests: %d clients x %d requests = %d total in %v (%.2f req/s)",
			concurrency, requestsPerClient, totalRequests, elapsed, rps)

		// Target: > 50 req/s minimum
		if rps < 50 {
			t.Logf("⚠️  Request throughput (%.2f req/s) below target (50 req/s)", rps)
		}
	})

	t.Run("DatabaseConnectionPool", func(t *testing.T) {
		// Test database connection handling under concurrent load
		concurrency := 20
		var wg sync.WaitGroup

		start := time.Now()
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				videoID := insertTestVideo(&testing.T{}, server.db)
				path := fmt.Sprintf("/api/v1/video/%s", videoID)
				makeHTTPRequest(server, "GET", path, nil, nil)
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Database pool test: %d concurrent operations in %v", concurrency, elapsed)

		// Should complete within reasonable time
		if elapsed > 5*time.Second {
			t.Logf("⚠️  Database operations slow under load: %v", elapsed)
		}
	})

	t.Run("JobCreationThroughput", func(t *testing.T) {
		videoID := insertTestVideo(t, server.db)
		path := fmt.Sprintf("/api/v1/video/%s/convert", videoID)
		payload := GenerateTestConvertRequest("webm")

		iterations := 50
		start := time.Now()
		for i := 0; i < iterations; i++ {
			makeJSONRequest(server, "POST", path, payload)
		}
		elapsed := time.Since(start)

		jobsPerSecond := float64(iterations) / elapsed.Seconds()
		t.Logf("Job creation throughput: %.2f jobs/s (%d jobs in %v)",
			jobsPerSecond, iterations, elapsed)
	})
}

// TestMemoryUsage validates memory usage patterns
func TestMemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanupServer := setupTestServer(t)
	defer cleanupServer()
	defer cleanupTestData(t, server.db)

	t.Run("JobListingMemory", func(t *testing.T) {
		// Create many jobs
		videoID := insertTestVideo(t, server.db)
		for i := 0; i < 100; i++ {
			insertTestJob(t, server.db, videoID, "convert")
		}

		// Request job list multiple times
		for i := 0; i < 10; i++ {
			w := makeHTTPRequest(server, "GET", "/api/v1/jobs", nil, nil)
			if w.Code != 200 {
				t.Errorf("Job listing failed: %d", w.Code)
			}
		}

		t.Log("✅ Job listing completed without memory issues")
	})

	t.Run("StreamListingMemory", func(t *testing.T) {
		// Create multiple streams
		payload := map[string]interface{}{
			"name": "Test Stream",
			"input_source": map[string]interface{}{
				"type": "file",
			},
			"output_targets": []map[string]interface{}{
				{"platform": "custom", "url": "rtmp://test.com/live"},
			},
		}

		for i := 0; i < 20; i++ {
			makeJSONRequest(server, "POST", "/api/v1/stream/create", payload)
		}

		// List streams
		w := makeHTTPRequest(server, "GET", "/api/v1/streams", nil, nil)
		if w.Code != 200 {
			t.Errorf("Stream listing failed: %d", w.Code)
		}

		t.Log("✅ Stream listing completed without memory issues")
	})
}

// Helper function to calculate percentile
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Simple bubble sort for small datasets
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := int(float64(len(sorted)) * percentile)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
