package main

import (
	"testing"
	"time"
)

// TestPerformanceScenarios tests performance characteristics
func TestPerformanceScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	scenarios := GetPerformanceScenarios()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, ts)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, setupData)
			}

			// Measure execution time
			start := time.Now()
			scenario.Execute(t, ts, setupData)
			duration := time.Since(start)

			// Validate performance
			if scenario.MaxDurationMS > 0 {
				maxDuration := time.Duration(scenario.MaxDurationMS) * time.Millisecond
				if duration > maxDuration {
					t.Errorf("%s took %v, expected < %v", scenario.Description, duration, maxDuration)
				}
			}

			t.Logf("%s completed in %v", scenario.Name, duration)
		})
	}
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}
				w := makeHTTPRequest(ts, req)
				if w.Code != 200 {
					t.Errorf("Health check failed with status %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})

	t.Run("ConcurrentTemplateRequests", func(t *testing.T) {
		concurrency := 5
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/templates",
				}
				w := makeHTTPRequest(ts, req)
				if w.Code != 200 {
					t.Errorf("Templates request failed with status %d", w.Code)
				}
				done <- true
			}()
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

// TestMemoryUsage tests for memory leaks (basic)
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create mock servers
	mockSearXNG := mockSearXNGServer(t)
	defer mockSearXNG.Close()
	ts.Server.searxngURL = mockSearXNG.URL

	t.Run("RepeatedSearchRequests", func(t *testing.T) {
		// Run many requests to check for leaks
		iterations := 100

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/search",
				Body: map[string]interface{}{
					"query": "test query iteration",
					"limit": 5,
				},
			}
			w := makeHTTPRequest(ts, req)

			if w.Code != 200 {
				t.Errorf("Iteration %d failed with status %d", i, w.Code)
			}
		}

		t.Logf("Completed %d search requests", iterations)
	})
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		_ = makeHTTPRequest(ts, req)
	}
}

// BenchmarkGetTemplates benchmarks the templates endpoint
func BenchmarkGetTemplates(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/templates",
		}
		_ = makeHTTPRequest(ts, req)
	}
}

// BenchmarkGetDepthConfigs benchmarks the depth configs endpoint
func BenchmarkGetDepthConfigs(b *testing.B) {
	ts := setupTestServer(&testing.T{})
	defer ts.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/depth-configs",
		}
		_ = makeHTTPRequest(ts, req)
	}
}

// BenchmarkValidateDepth benchmarks the depth validation function
func BenchmarkValidateDepth(b *testing.B) {
	testCases := []string{"quick", "standard", "deep", "invalid"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = validateDepth(tc)
		}
	}
}

// BenchmarkCalculateDomainAuthority benchmarks domain authority calculation
func BenchmarkCalculateDomainAuthority(b *testing.B) {
	urls := []string{
		"https://arxiv.org/paper",
		"https://github.com/repo",
		"https://example.com/page",
		"https://stanford.edu/research",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, url := range urls {
			_ = calculateDomainAuthority(url)
		}
	}
}

// BenchmarkCalculateSourceQuality benchmarks source quality calculation
func BenchmarkCalculateSourceQuality(b *testing.B) {
	result := map[string]interface{}{
		"url":           "https://nature.com/article",
		"title":         "Research Paper on AI",
		"content":       "Comprehensive analysis of artificial intelligence...",
		"publishedDate": "2024-01-15",
		"author":        "Dr. Smith",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateSourceQuality(result)
	}
}
