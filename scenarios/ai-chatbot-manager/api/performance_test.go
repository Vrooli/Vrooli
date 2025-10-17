package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealthHandlerPerformance tests health endpoint performance
func TestHealthHandlerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	t.Run("LatencyTest", func(t *testing.T) {
		iterations := 100
		maxLatency := 100 * time.Millisecond

		var totalDuration time.Duration
		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.HealthHandler(w, httpReq)
			duration := time.Since(start)
			totalDuration += duration

			if duration > maxLatency {
				t.Logf("Warning: Health check took %v (max: %v)", duration, maxLatency)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Health check performance: %d iterations, avg: %v, total: %v", iterations, avgDuration, totalDuration)

		if avgDuration > maxLatency {
			t.Errorf("Average latency %v exceeded max %v", avgDuration, maxLatency)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 50
		iterations := 10

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}

					w, httpReq, err := makeHTTPRequest(req)
					if err != nil {
						errors <- err
						return
					}

					server.HealthHandler(w, httpReq)

					if w.Code != 200 {
						errors <- fmt.Errorf("unexpected status: %d", w.Code)
					}
				}
			}()
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("Error: %v", err)
		}

		totalRequests := concurrency * iterations
		t.Logf("Concurrent health checks: %d requests in %v (%.2f req/sec)",
			totalRequests, duration, float64(totalRequests)/duration.Seconds())

		if errorCount > 0 {
			t.Errorf("Encountered %d errors out of %d requests", errorCount, totalRequests)
		}
	})

	t.Run("CacheEfficiency", func(t *testing.T) {
		// First request (cache miss)
		start1 := time.Now()
		req1 := HTTPTestRequest{Method: "GET", Path: "/health"}
		w1, httpReq1, _ := makeHTTPRequest(req1)
		server.HealthHandler(w1, httpReq1)
		duration1 := time.Since(start1)

		// Second request (should hit cache)
		start2 := time.Now()
		req2 := HTTPTestRequest{Method: "GET", Path: "/health"}
		w2, httpReq2, _ := makeHTTPRequest(req2)
		server.HealthHandler(w2, httpReq2)
		duration2 := time.Since(start2)

		t.Logf("First request (cache miss): %v", duration1)
		t.Logf("Second request (cache hit): %v", duration2)

		// Cached request should be faster
		if duration2 > duration1 {
			t.Logf("Warning: Cached request was slower (%v vs %v)", duration2, duration1)
		}
	})
}

// TestChatbotCRUDPerformance tests chatbot CRUD operation performance
func TestChatbotCRUDPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
		OllamaURL:   os.Getenv("OLLAMA_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	t.Run("BulkCreatePerformance", func(t *testing.T) {
		count := 50
		maxDuration := 10 * time.Second

		start := time.Now()
		chatbotIDs := make([]string, 0, count)

		for i := 0; i < count; i++ {
			chatbotData := TestData.CreateChatbotRequest(fmt.Sprintf("Bulk Test %d", i))

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/chatbots",
				Body:   chatbotData,
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.CreateChatbotHandler(w, httpReq)

			if w.Code == 201 {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
					if id, ok := response["id"].(string); ok {
						chatbotIDs = append(chatbotIDs, id)
					}
				}
			}
		}

		duration := time.Since(start)
		t.Logf("Created %d chatbots in %v (%.2f chatbots/sec)",
			len(chatbotIDs), duration, float64(len(chatbotIDs))/duration.Seconds())

		if duration > maxDuration {
			t.Errorf("Bulk create took %v, exceeded max %v", duration, maxDuration)
		}

		// Cleanup
		for _, id := range chatbotIDs {
			testDB.db.DeleteChatbot(id)
		}
	})

	t.Run("BulkReadPerformance", func(t *testing.T) {
		// Create test chatbots
		chatbotCount := 20
		chatbots := make([]*TestChatbot, chatbotCount)
		for i := 0; i < chatbotCount; i++ {
			chatbots[i] = setupTestChatbot(t, testDB.db, fmt.Sprintf("Read Test %d", i))
			defer chatbots[i].Cleanup()
		}

		// Test reading all chatbots
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/chatbots",
			}

			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			server.ListChatbotsHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("List chatbots failed with status %d", w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("List chatbots performance: %d iterations, avg: %v, total: %v",
			iterations, avgDuration, duration)

		maxAvgDuration := 50 * time.Millisecond
		if avgDuration > maxAvgDuration {
			t.Errorf("Average list duration %v exceeded max %v", avgDuration, maxAvgDuration)
		}
	})

	t.Run("ConcurrentUpdatePerformance", func(t *testing.T) {
		// Create a test chatbot
		chatbot := setupTestChatbot(t, testDB.db, "Concurrent Update Test")
		defer chatbot.Cleanup()

		concurrency := 10
		iterations := 5

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					updateData := map[string]interface{}{
						"name": fmt.Sprintf("Updated by worker %d iteration %d", workerID, j),
					}

					req := HTTPTestRequest{
						Method:  "PUT",
						Path:    "/api/v1/chatbots/" + chatbot.Chatbot.ID,
						URLVars: map[string]string{"id": chatbot.Chatbot.ID},
						Body:    updateData,
					}

					w, httpReq, err := makeHTTPRequest(req)
					if err != nil {
						errors <- err
						return
					}

					server.UpdateChatbotHandler(w, httpReq)

					if w.Code != 200 {
						errors <- fmt.Errorf("update failed with status %d", w.Code)
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		duration := time.Since(start)

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("Error: %v", err)
		}

		totalRequests := concurrency * iterations
		t.Logf("Concurrent updates: %d requests in %v (%.2f req/sec)",
			totalRequests, duration, float64(totalRequests)/duration.Seconds())

		if errorCount > totalRequests/10 {
			t.Errorf("Too many errors: %d out of %d requests", errorCount, totalRequests)
		}
	})
}

// TestDatabaseConnectionPooling tests database connection pool performance
func TestDatabaseConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(t)
	defer testDB.Cleanup()

	t.Run("ConnectionReuse", func(t *testing.T) {
		concurrency := 20
		iterations := 10

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					// Perform a simple query
					chatbot := setupTestChatbot(t, testDB.db, "Pool Test")
					chatbot.Cleanup()
				}
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		totalOps := concurrency * iterations
		t.Logf("Connection pool test: %d operations in %v (%.2f ops/sec)",
			totalOps, duration, float64(totalOps)/duration.Seconds())
	})
}

// BenchmarkHealthHandler benchmarks the health check handler
func BenchmarkHealthHandler(b *testing.B) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(&testing.T{})
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, _ := makeHTTPRequest(req)
		server.HealthHandler(w, httpReq)
	}
}

// BenchmarkCreateChatbot benchmarks chatbot creation
func BenchmarkCreateChatbot(b *testing.B) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(&testing.T{})
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		chatbotData := TestData.CreateChatbotRequest(fmt.Sprintf("Benchmark %d", i))

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/chatbots",
			Body:   chatbotData,
		}

		w, httpReq, _ := makeHTTPRequest(req)
		server.CreateChatbotHandler(w, httpReq)

		// Cleanup
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
			if id, ok := response["id"].(string); ok {
				testDB.db.DeleteChatbot(id)
			}
		}
	}
}

// BenchmarkListChatbots benchmarks listing chatbots
func BenchmarkListChatbots(b *testing.B) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	testDB := setupTestDatabase(&testing.T{})
	defer testDB.Cleanup()

	cfg := &Config{
		APIPort:     "8080",
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}
	server := NewServer(cfg, testDB.db, logger)

	// Create some test data
	for i := 0; i < 10; i++ {
		chatbot := &Chatbot{
			ID:          uuid.New().String(),
			Name:        fmt.Sprintf("Benchmark Chatbot %d", i),
			Description: "Benchmark test",
			IsActive:    true,
		}
		testDB.db.CreateChatbot(chatbot)
		defer testDB.db.DeleteChatbot(chatbot.ID)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/chatbots",
		}

		w, httpReq, _ := makeHTTPRequest(req)
		server.ListChatbotsHandler(w, httpReq)
	}
}

// BenchmarkIsValidUUID benchmarks UUID validation
func BenchmarkIsValidUUID(b *testing.B) {
	validUUID := uuid.New().String()
	invalidUUID := "invalid-uuid"

	b.Run("ValidUUID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isValidUUID(validUUID)
		}
	})

	b.Run("InvalidUUID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isValidUUID(invalidUUID)
		}
	})
}
