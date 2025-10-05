package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestPerformanceHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("HealthCheckLatency", func(t *testing.T) {
		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := makeHTTPRequest("GET", "/health", nil, "")
			w := httptest.NewRecorder()

			env.Server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Health check failed with status %d", w.Code)
			}

			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average health check latency: %v", avgDuration)

		// Should be very fast (< 10ms average)
		if avgDuration > 10*time.Millisecond {
			t.Logf("Warning: Average latency %v exceeds 10ms target", avgDuration)
		}
	})
}

func TestPerformanceConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		concurrency := 50
		requestsPerWorker := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*requestsPerWorker)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					req := makeHTTPRequest("GET", "/health", nil, "")
					w := httptest.NewRecorder()

					env.Server.router.ServeHTTP(w, req)

					if w.Code != http.StatusOK {
						errors <- fmt.Errorf("worker %d request %d failed with status %d", workerID, j, w.Code)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalRequests := concurrency * requestsPerWorker

		t.Logf("Completed %d concurrent requests in %v", totalRequests, duration)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/duration.Seconds())

		errorCount := len(errors)
		if errorCount > 0 {
			t.Errorf("Failed requests: %d/%d", errorCount, totalRequests)
			for err := range errors {
				t.Error(err)
			}
		}
	})
}

func TestPerformanceDataParsing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ParseLargeCSV", func(t *testing.T) {
		// Generate large CSV
		rows := 1000
		csvData := "name,age,email\n"
		for i := 0; i < rows; i++ {
			csvData += fmt.Sprintf("User%d,%d,user%d@example.com\n", i, 20+i%50, i)
		}

		parseBody := map[string]interface{}{
			"data":   csvData,
			"format": "csv",
		}

		start := time.Now()

		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Parse failed with status %d", w.Code)
		}

		t.Logf("Parsed %d rows in %v", rows, duration)
		t.Logf("Throughput: %.2f rows/second", float64(rows)/duration.Seconds())

		// Should parse 1000 rows in under 1 second
		if duration > time.Second {
			t.Logf("Warning: Parsing %d rows took %v (> 1s)", rows, duration)
		}
	})
}

func TestPerformanceDataTransformation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("TransformLargeDataset", func(t *testing.T) {
		// Generate large dataset
		rows := 500
		data := make([]map[string]interface{}, rows)
		for i := 0; i < rows; i++ {
			data[i] = map[string]interface{}{
				"id":    float64(i),
				"value": float64(i * 2),
				"name":  fmt.Sprintf("Item%d", i),
			}
		}

		transformBody := map[string]interface{}{
			"data": data,
			"transformations": []map[string]interface{}{
				{
					"type": "filter",
					"parameters": map[string]interface{}{
						"condition": "value > 100",
					},
				},
			},
		}

		start := time.Now()

		req := makeHTTPRequest("POST", "/api/v1/data/transform", transformBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Transform failed with status %d", w.Code)
		}

		t.Logf("Transformed %d rows in %v", rows, duration)
		t.Logf("Throughput: %.2f rows/second", float64(rows)/duration.Seconds())
	})
}

func TestPerformanceDataValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidateLargeDataset", func(t *testing.T) {
		// Generate large dataset
		rows := 500
		data := make([]map[string]interface{}, rows)
		for i := 0; i < rows; i++ {
			data[i] = map[string]interface{}{
				"name":  fmt.Sprintf("User%d", i),
				"age":   float64(20 + i%50),
				"email": fmt.Sprintf("user%d@example.com", i),
			}
		}

		validateBody := map[string]interface{}{
			"data": data,
			"schema": map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "name", "type": "string", "nullable": false},
					{"name": "age", "type": "integer", "nullable": false},
					{"name": "email", "type": "string", "nullable": false},
				},
			},
			"quality_rules": []map[string]interface{}{},
		}

		start := time.Now()

		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Validation failed with status %d", w.Code)
		}

		t.Logf("Validated %d rows in %v", rows, duration)
		t.Logf("Throughput: %.2f rows/second", float64(rows)/duration.Seconds())
	})
}

func TestPerformanceQueryExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SimpleQueryPerformance", func(t *testing.T) {
		queryBody := map[string]interface{}{
			"sql": "SELECT generate_series(1, 100) as num, md5(random()::text) as hash",
			"options": map[string]interface{}{
				"limit": 100,
			},
		}

		iterations := 10
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := makeHTTPRequest("POST", "/api/v1/data/query", queryBody, "test-token")
			w := httptest.NewRecorder()

			env.Server.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Query failed with status %d", w.Code)
			}

			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average query execution time: %v", avgDuration)

		// Should be relatively fast (< 100ms average)
		if avgDuration > 100*time.Millisecond {
			t.Logf("Warning: Average query time %v exceeds 100ms target", avgDuration)
		}
	})
}

func TestPerformanceMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateManyResources", func(t *testing.T) {
		resourceCount := 100

		for i := 0; i < resourceCount; i++ {
			createBody := map[string]interface{}{
				"name":        fmt.Sprintf("Resource%d", i),
				"description": fmt.Sprintf("Description for resource %d", i),
				"config":      map[string]interface{}{"key": fmt.Sprintf("value%d", i)},
			}

			req := makeHTTPRequest("POST", "/api/v1/resources/create", createBody, "test-token")
			w := httptest.NewRecorder()

			env.Server.router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated && w.Code != http.StatusOK {
				t.Errorf("Failed to create resource %d: status %d", i, w.Code)
			}
		}

		t.Logf("Successfully created %d resources", resourceCount)

		// Verify we can list them all
		req := makeHTTPRequest("GET", "/api/v1/resources/list", nil, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Failed to list resources: status %d", w.Code)
		}
	})
}

func BenchmarkHealthCheck(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("GET", "/health", nil, "")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)
	}
}

func BenchmarkResourceCreate(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	createBody := map[string]interface{}{
		"name":        "Benchmark Resource",
		"description": "Resource for benchmarking",
		"config":      map[string]interface{}{"key": "value"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/resources/create", createBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)
	}
}

func BenchmarkDataParse(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	parseBody := map[string]interface{}{
		"data":   "name,age\nJohn,30\nJane,25\nBob,35",
		"format": "csv",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)
	}
}

func BenchmarkDataValidate(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	validateBody := map[string]interface{}{
		"data": []map[string]interface{}{
			{"name": "John", "age": 30.0},
			{"name": "Jane", "age": 25.0},
		},
		"schema": map[string]interface{}{
			"columns": []map[string]interface{}{
				{"name": "name", "type": "string", "nullable": false},
				{"name": "age", "type": "integer", "nullable": false},
			},
		},
		"quality_rules": []map[string]interface{}{},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)
	}
}
