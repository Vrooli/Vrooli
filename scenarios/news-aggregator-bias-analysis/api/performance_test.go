// +build testing

package main

import (
	"fmt"
	"testing"
	"time"
)

// TestPerformance_GetArticles tests article retrieval performance
func TestPerformance_GetArticles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test data
	articles := GenerateTestArticles(t, env, 50, "Performance Test Source")
	defer CleanupTestArticles(articles)

	pattern := PerformanceTestPattern{
		Name:          "GetArticles",
		Description:   "Test article retrieval performance",
		MaxDuration:   5 * time.Second,
		MinThroughput: 10, // 10 requests per second minimum
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   "/articles",
			}
		},
		Validate: func(t *testing.T, duration time.Duration, throughput float64) {
			t.Logf("Article retrieval throughput: %.2f req/s", throughput)
			if duration > 5*time.Second {
				t.Errorf("Performance degraded: took %v", duration)
			}
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformance_GetFeeds tests feed retrieval performance
func TestPerformance_GetFeeds(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test data
	feeds := GenerateTestFeeds(t, env, 20)
	defer CleanupTestFeeds(feeds)

	pattern := PerformanceTestPattern{
		Name:          "GetFeeds",
		Description:   "Test feed retrieval performance",
		MaxDuration:   5 * time.Second,
		MinThroughput: 20, // 20 requests per second minimum
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   "/feeds",
			}
		},
		Validate: func(t *testing.T, duration time.Duration, throughput float64) {
			t.Logf("Feed retrieval throughput: %.2f req/s", throughput)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformance_ArticleSearch tests article search performance
func TestPerformance_ArticleSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test articles with searchable content
	for i := 0; i < 100; i++ {
		title := fmt.Sprintf("Performance Test Article %d", i)
		createTestArticle(t, env, title, "Test Source")
	}

	pattern := PerformanceTestPattern{
		Name:          "ArticleSearch",
		Description:   "Test article search performance with filters",
		MaxDuration:   10 * time.Second,
		MinThroughput: 5,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:      "GET",
				Path:        "/articles",
				QueryParams: map[string]string{"source": "Test Source", "limit": "50"},
			}
		},
		Validate: func(t *testing.T, duration time.Duration, throughput float64) {
			t.Logf("Article search throughput: %.2f req/s", throughput)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformance_GetPerspectives tests perspective retrieval performance
func TestPerformance_GetPerspectives(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test articles with various topics
	topics := []string{"climate", "economy", "politics", "technology"}
	for _, topic := range topics {
		for i := 0; i < 10; i++ {
			title := fmt.Sprintf("%s Article %d", topic, i)
			createTestArticle(t, env, title, "Test Source")
		}
	}

	pattern := PerformanceTestPattern{
		Name:          "GetPerspectives",
		Description:   "Test perspective retrieval performance",
		MaxDuration:   10 * time.Second,
		MinThroughput: 5,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    "/perspectives/climate",
				URLVars: map[string]string{"topic": "climate"},
			}
		},
		Validate: func(t *testing.T, duration time.Duration, throughput float64) {
			t.Logf("Perspective retrieval throughput: %.2f req/s", throughput)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformance_DatabaseQueries tests database query performance
func TestPerformance_DatabaseQueries(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create large dataset
	t.Log("Creating test dataset...")
	for i := 0; i < 200; i++ {
		title := fmt.Sprintf("DB Performance Test %d", i)
		createTestArticle(t, env, title, "Test Source")
	}
	t.Log("Test dataset created")

	t.Run("SelectPerformance", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			rows, err := env.DB.Query(`
				SELECT id, title, url, source FROM articles LIMIT 50
			`)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			rows.Close()
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()

		t.Logf("Database SELECT performance: %d queries in %v (%.2f queries/s)",
			iterations, duration, throughput)

		if duration > 5*time.Second {
			t.Errorf("Database queries too slow: %v", duration)
		}
	})

	t.Run("FilteredQueryPerformance", func(t *testing.T) {
		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			rows, err := env.DB.Query(`
				SELECT id, title FROM articles
				WHERE source = $1 AND title ILIKE $2
				ORDER BY published_at DESC
				LIMIT 10
			`, "Test Source", "%Performance%")
			if err != nil {
				t.Fatalf("Filtered query failed: %v", err)
			}
			rows.Close()
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()

		t.Logf("Filtered query performance: %d queries in %v (%.2f queries/s)",
			iterations, duration, throughput)
	})

	t.Run("AggregationPerformance", func(t *testing.T) {
		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			var count int
			err := env.DB.QueryRow(`
				SELECT COUNT(*) FROM articles WHERE source = $1
			`, "Test Source").Scan(&count)
			if err != nil {
				t.Fatalf("Aggregation query failed: %v", err)
			}
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()

		t.Logf("Aggregation query performance: %d queries in %v (%.2f queries/s)",
			iterations, duration, throughput)
	})
}

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create test data
	articles := GenerateTestArticles(t, env, 20, "Concurrent Test Source")
	defer CleanupTestArticles(articles)

	t.Run("ConcurrentGET", func(t *testing.T) {
		concurrency := 10
		requestsPerWorker := 10

		start := time.Now()
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < requestsPerWorker; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/articles",
					}

					w, err := makeHTTPRequest(env, req)
					if err != nil {
						t.Errorf("Worker %d request %d failed: %v", workerID, j, err)
					}

					if w.Code != 200 {
						t.Errorf("Worker %d request %d returned status %d", workerID, j, w.Code)
					}
				}
				done <- true
			}(i)
		}

		// Wait for all workers
		for i := 0; i < concurrency; i++ {
			<-done
		}

		duration := time.Since(start)
		totalRequests := concurrency * requestsPerWorker
		throughput := float64(totalRequests) / duration.Seconds()

		t.Logf("Concurrent requests: %d requests from %d workers in %v (%.2f req/s)",
			totalRequests, concurrency, duration, throughput)

		if duration > 10*time.Second {
			t.Errorf("Concurrent requests too slow: %v", duration)
		}
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LargeResultSet", func(t *testing.T) {
		// Create many articles
		for i := 0; i < 500; i++ {
			title := fmt.Sprintf("Memory Test Article %d", i)
			createTestArticle(t, env, title, "Memory Test Source")
		}

		// Retrieve all articles
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/articles",
			QueryParams: map[string]string{"limit": "500"},
		}

		start := time.Now()
		w, err := makeHTTPRequest(env, req)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Large result set retrieval: %v, response size: %d bytes",
			duration, w.Body.Len())

		if duration > 5*time.Second {
			t.Errorf("Large result set retrieval too slow: %v", duration)
		}
	})
}

// BenchmarkGetArticles benchmarks article retrieval
func BenchmarkGetArticles(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Create test data
	for i := 0; i < 50; i++ {
		title := fmt.Sprintf("Benchmark Article %d", i)
		createTestArticle(&testing.T{}, env, title, "Benchmark Source")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/articles",
		}

		makeHTTPRequest(env, req)
	}
}

// BenchmarkGetArticleByID benchmarks single article retrieval
func BenchmarkGetArticleByID(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	testArticle := createTestArticle(&testing.T{}, env, "Benchmark Single", "Benchmark Source")
	defer testArticle.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/articles/" + testArticle.Article.ID,
			URLVars: map[string]string{"id": testArticle.Article.ID},
		}

		makeHTTPRequest(env, req)
	}
}

// BenchmarkDatabaseQuery benchmarks raw database queries
func BenchmarkDatabaseQuery(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rows, _ := env.DB.Query("SELECT id, title FROM articles LIMIT 10")
		if rows != nil {
			rows.Close()
		}
	}
}
