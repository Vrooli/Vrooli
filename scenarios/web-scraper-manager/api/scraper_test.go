// +build testing

package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewScraperOrchestrator(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateOrchestrator", func(t *testing.T) {
		mockDB := &sql.DB{}
		so := NewScraperOrchestrator(mockDB)

		if so == nil {
			t.Fatal("Expected orchestrator to be created")
		}

		if so.db != mockDB {
			t.Error("Expected db to be set")
		}

		if so.httpClient == nil {
			t.Error("Expected httpClient to be initialized")
		}

		if so.jobs == nil {
			t.Error("Expected jobs channel to be initialized")
		}

		if so.workers != 5 {
			t.Errorf("Expected 5 workers, got %d", so.workers)
		}

		if so.rateLimiter == nil {
			t.Error("Expected rateLimiter to be initialized")
		}
	})
}

func TestSubmitJob(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("SubmitJobSuccess", func(t *testing.T) {
		job := &ScrapeJob{
			URL:        "https://example.com",
			Type:       "static",
			Method:     "GET",
			MaxRetries: 3,
			Timeout:    30,
			Status:     "pending",
		}

		err := so.SubmitJob(job)
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}

		if job.ID == "" {
			t.Error("Expected job ID to be generated")
		}
	})

	t.Run("SubmitJobWithID", func(t *testing.T) {
		jobID := uuid.New().String()
		job := &ScrapeJob{
			ID:         jobID,
			URL:        "https://example.com",
			Type:       "static",
			Method:     "GET",
			MaxRetries: 3,
			Status:     "pending",
		}

		err := so.SubmitJob(job)
		if err != nil {
			t.Errorf("Failed to submit job: %v", err)
		}

		if job.ID != jobID {
			t.Errorf("Expected job ID %s, got %s", jobID, job.ID)
		}
	})
}

func TestScrapeStatic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<html>
				<head>
					<title>Test Page</title>
					<meta name="description" content="Test description">
				</head>
				<body>
					<h1>Test Heading</h1>
					<p>Test paragraph</p>
					<a href="/test" class="link">Test Link</a>
				</body>
			</html>
		`))
	}))
	defer ts.Close()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("ScrapeStaticSuccess", func(t *testing.T) {
		job := &ScrapeJob{
			ID:     uuid.New().String(),
			URL:    ts.URL,
			Type:   "static",
			Method: "GET",
			Selectors: map[string]string{
				"title":       "h1",
				"description": "meta:description",
				"link":        "a.link@href",
			},
			UserAgent: "TestAgent/1.0",
			Timeout:   10,
		}

		result, err := so.scrapeStatic(job)
		if err != nil {
			t.Fatalf("Failed to scrape: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", result.StatusCode)
		}

		if result.Data["title"] != "Test Heading" {
			t.Errorf("Expected title 'Test Heading', got '%v'", result.Data["title"])
		}

		if result.Data["description"] != "Test description" {
			t.Errorf("Expected description, got '%v'", result.Data["description"])
		}

		if result.Data["link"] != "/test" {
			t.Errorf("Expected link '/test', got '%v'", result.Data["link"])
		}
	})

	t.Run("ScrapeStaticMultipleElements", func(t *testing.T) {
		job := &ScrapeJob{
			ID:     uuid.New().String(),
			URL:    ts.URL,
			Type:   "static",
			Method: "GET",
			Selectors: map[string]string{
				"paragraphs": "p:all",
			},
			Timeout: 10,
		}

		result, err := so.scrapeStatic(job)
		if err != nil {
			t.Fatalf("Failed to scrape: %v", err)
		}

		paragraphs, ok := result.Data["paragraphs"].([]string)
		if !ok {
			t.Fatal("Expected paragraphs to be string array")
		}

		if len(paragraphs) == 0 {
			t.Error("Expected at least one paragraph")
		}
	})

	t.Run("ScrapeStaticWithProxy", func(t *testing.T) {
		job := &ScrapeJob{
			ID:       uuid.New().String(),
			URL:      ts.URL,
			Type:     "static",
			Method:   "GET",
			ProxyURL: "http://invalid-proxy:8080",
			Timeout:  2,
		}

		_, err := so.scrapeStatic(job)
		// We expect an error because the proxy is invalid
		if err == nil {
			t.Error("Expected error with invalid proxy")
		}
	})

	t.Run("ScrapeStaticInvalidURL", func(t *testing.T) {
		job := &ScrapeJob{
			ID:     uuid.New().String(),
			URL:    "http://invalid-url-that-does-not-exist-12345.com",
			Type:   "static",
			Method: "GET",
			Timeout: 5,
		}

		_, err := so.scrapeStatic(job)
		if err == nil {
			t.Error("Expected error with invalid URL")
		}
	})
}

func TestScrapeAPI(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a test API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "data": {"items": [1, 2, 3]}}`))
	}))
	defer ts.Close()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("ScrapeAPISuccess", func(t *testing.T) {
		job := &ScrapeJob{
			ID:      uuid.New().String(),
			URL:     ts.URL,
			Type:    "api",
			Method:  "GET",
			Headers: map[string]string{"X-API-Key": "test-key"},
			Timeout: 10,
		}

		result, err := so.scrapeAPI(job)
		if err != nil {
			t.Fatalf("Failed to scrape API: %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", result.StatusCode)
		}

		responseData, ok := result.Data["response"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected response data")
		}

		if responseData["status"] != "success" {
			t.Errorf("Expected status 'success', got '%v'", responseData["status"])
		}
	})

	t.Run("ScrapeAPIWithPayload", func(t *testing.T) {
		job := &ScrapeJob{
			ID:     uuid.New().String(),
			URL:    ts.URL,
			Type:   "api",
			Method: "POST",
			Payload: map[string]interface{}{
				"query": "test",
				"limit": 10,
			},
			Timeout: 10,
		}

		result, err := so.scrapeAPI(job)
		if err != nil {
			t.Fatalf("Failed to scrape API: %v", err)
		}

		if result.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", result.StatusCode)
		}
	})

	t.Run("ScrapeAPINonJSON", func(t *testing.T) {
		textServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("plain text response"))
		}))
		defer textServer.Close()

		job := &ScrapeJob{
			ID:      uuid.New().String(),
			URL:     textServer.URL,
			Type:    "api",
			Method:  "GET",
			Timeout: 10,
		}

		result, err := so.scrapeAPI(job)
		if err != nil {
			t.Fatalf("Failed to scrape API: %v", err)
		}

		responseData, ok := result.Data["response"].(string)
		if !ok {
			t.Fatal("Expected string response")
		}

		if responseData != "plain text response" {
			t.Errorf("Expected 'plain text response', got '%s'", responseData)
		}
	})
}

func TestApplyRateLimit(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("RateLimitCreation", func(t *testing.T) {
		start := time.Now()
		so.applyRateLimit("https://example.com/test")
		duration := time.Since(start)

		// First call should create rate limiter and not wait
		if duration > 100*time.Millisecond {
			t.Errorf("First rate limit call took too long: %v", duration)
		}

		// Check that rate limiter was created
		so.mu.RLock()
		_, exists := so.rateLimiter["example.com"]
		so.mu.RUnlock()

		if !exists {
			t.Error("Expected rate limiter to be created for domain")
		}
	})

	t.Run("RateLimitWait", func(t *testing.T) {
		// First call creates the limiter
		so.applyRateLimit("https://test.com/page1")

		// Second call should wait
		start := time.Now()
		so.applyRateLimit("https://test.com/page2")
		duration := time.Since(start)

		// Should wait approximately 1 second (rate limit interval)
		if duration < 900*time.Millisecond || duration > 1100*time.Millisecond {
			t.Errorf("Rate limit wait time unexpected: %v", duration)
		}
	})

	t.Run("RateLimitInvalidURL", func(t *testing.T) {
		// Should not panic with invalid URL
		so.applyRateLimit("not-a-valid-url")
	})
}

func TestScheduleRetry(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("ScheduleRetry", func(t *testing.T) {
		// Create a job first
		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        "https://example.com",
			Type:       "static",
			Method:     "GET",
			MaxRetries: 3,
			RetryCount: 0,
			Status:     "pending",
		}

		// Store it
		err := so.storeJob(job)
		if err != nil {
			t.Fatalf("Failed to store job: %v", err)
		}

		// Schedule retry
		so.scheduleRetry(job)

		if job.RetryCount != 1 {
			t.Errorf("Expected retry count 1, got %d", job.RetryCount)
		}
	})
}

func TestStoreJob(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("StoreJobSuccess", func(t *testing.T) {
		job := &ScrapeJob{
			ID:        uuid.New().String(),
			URL:       "https://example.com",
			Type:      "static",
			Method:    "GET",
			Headers:   map[string]string{"User-Agent": "TestBot"},
			Selectors: map[string]string{"title": "h1"},
			Schedule:  "*/5 * * * *",
			MaxRetries: 3,
			Timeout:   30,
			UserAgent: "TestBot/1.0",
			Status:    "pending",
		}

		err := so.storeJob(job)
		if err != nil {
			t.Fatalf("Failed to store job: %v", err)
		}
	})

	t.Run("StoreJobConflict", func(t *testing.T) {
		jobID := uuid.New().String()
		job := &ScrapeJob{
			ID:        jobID,
			URL:       "https://example.com",
			Type:      "static",
			Method:    "GET",
			MaxRetries: 3,
			Status:    "pending",
		}

		// Store once
		err := so.storeJob(job)
		if err != nil {
			t.Fatalf("Failed to store job: %v", err)
		}

		// Store again with same ID (should update)
		err = so.storeJob(job)
		if err != nil {
			t.Fatalf("Failed to update job: %v", err)
		}
	})
}

func TestStoreResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("StoreResultSuccess", func(t *testing.T) {
		result := &ScrapeResult{
			JobID:      uuid.New().String(),
			URL:        "https://example.com",
			StatusCode: 200,
			Data:       map[string]interface{}{"title": "Test"},
			HTML:       "<html>test</html>",
			Duration:   1234,
			ScrapedAt:  time.Now(),
		}

		so.storeResult(result)
		// No error expected, just verify it doesn't panic
	})

	t.Run("StoreResultWithScreenshot", func(t *testing.T) {
		result := &ScrapeResult{
			JobID:      uuid.New().String(),
			URL:        "https://example.com",
			StatusCode: 200,
			Data:       map[string]interface{}{"title": "Test"},
			Screenshot: []byte("fake screenshot data"),
			Duration:   2345,
			ScrapedAt:  time.Now(),
		}

		so.storeResult(result)
	})
}

func TestUpdateJobStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("UpdateJobStatus", func(t *testing.T) {
		// Create job first
		job := &ScrapeJob{
			ID:        uuid.New().String(),
			URL:       "https://example.com",
			Type:      "static",
			Method:    "GET",
			MaxRetries: 3,
			Status:    "pending",
		}

		err := so.storeJob(job)
		if err != nil {
			t.Fatalf("Failed to store job: %v", err)
		}

		// Update status
		so.updateJobStatus(job.ID, "running")
		so.updateJobStatus(job.ID, "completed")
	})
}

func TestScraperOrchestratorLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("StartAndStop", func(t *testing.T) {
		so := NewScraperOrchestrator(db)

		err := so.Start()
		if err != nil {
			t.Fatalf("Failed to start orchestrator: %v", err)
		}

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		so.Stop()

		// Verify context is cancelled
		select {
		case <-so.ctx.Done():
			// Expected
		case <-time.After(1 * time.Second):
			t.Error("Context not cancelled after stop")
		}
	})
}

func TestProcessJob(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	// Create a test HTTP server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><h1>Test</h1></html>`))
	}))
	defer ts.Close()

	so := NewScraperOrchestrator(db)
	err := so.Start()
	if err != nil {
		t.Fatalf("Failed to start orchestrator: %v", err)
	}
	defer so.Stop()

	t.Run("ProcessJobStaticSuccess", func(t *testing.T) {
		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        ts.URL,
			Type:       "static",
			Method:     "GET",
			MaxRetries: 3,
			Timeout:    10,
			Selectors:  map[string]string{"title": "h1"},
		}

		// Process it directly (not through worker)
		so.processJob(job)
		// Verify no panic
	})

	t.Run("ProcessJobAPIType", func(t *testing.T) {
		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		}))
		defer apiServer.Close()

		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        apiServer.URL,
			Type:       "api",
			Method:     "GET",
			MaxRetries: 3,
			Timeout:    10,
		}

		so.processJob(job)
	})

	t.Run("ProcessJobDynamicType", func(t *testing.T) {
		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        ts.URL,
			Type:       "dynamic",
			Method:     "GET",
			MaxRetries: 3,
			Timeout:    10,
		}

		// This will fail since we don't have browserless configured
		// But it should not panic
		so.processJob(job)
	})

	t.Run("ProcessJobFailureWithRetry", func(t *testing.T) {
		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        "http://invalid-url-12345.com",
			Type:       "static",
			Method:     "GET",
			MaxRetries: 2,
			RetryCount: 0,
			Timeout:    2,
		}

		// Process it (should fail and schedule retry)
		so.processJob(job)

		// Verify retry count incremented
		if job.RetryCount != 1 {
			t.Errorf("Expected retry count 1, got %d", job.RetryCount)
		}
	})

	t.Run("ProcessJobMaxRetriesExceeded", func(t *testing.T) {
		job := &ScrapeJob{
			ID:         uuid.New().String(),
			URL:        "http://invalid-url-12345.com",
			Type:       "static",
			Method:     "GET",
			MaxRetries: 2,
			RetryCount: 2, // Already at max
			Timeout:    2,
		}

		// Process it (should fail without retry)
		so.processJob(job)
	})
}

// Test scrapeDynamic
func TestScrapeDynamic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("ScrapeDynamicNoBrowserless", func(t *testing.T) {
		job := &ScrapeJob{
			ID:      uuid.New().String(),
			URL:     "https://example.com",
			Type:    "dynamic",
			Method:  "GET",
			Timeout: 10,
		}

		// Should fail gracefully since browserless is not configured
		_, err := so.scrapeDynamic(job)
		if err == nil {
			t.Error("Expected error when browserless is not available")
		}
	})
}

// Test checkScheduledJobs
func TestCheckScheduledJobs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("CheckScheduledJobsNoJobs", func(t *testing.T) {
		// Should not panic with no scheduled jobs
		so.checkScheduledJobs()
	})
}

// Test checkRetries
func TestCheckRetries(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	if db == nil {
		t.Skip("Database not available for testing")
	}

	so := NewScraperOrchestrator(db)
	defer so.Stop()

	t.Run("CheckRetriesNoJobs", func(t *testing.T) {
		// Should not panic with no retries
		so.checkRetries()
	})
}
