package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDatabaseConnectionPooling tests connection pool performance
func TestDatabaseConnectionPooling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConcurrentQueries", func(t *testing.T) {
		campaign := createTestCampaign(t, env.DB, "pool-test")
		defer campaign.Cleanup()

		// Create test ideas
		for i := 0; i < 10; i++ {
			createTestIdea(t, env.DB, campaign.ID, "Idea "+string(rune(i)), "Content")
		}

		start := time.Now()
		concurrency := 20
		iterations := 50

		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterations)

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					query := `SELECT COUNT(*) FROM ideas WHERE campaign_id = $1`
					var count int
					err := env.DB.QueryRow(query, campaign.ID).Scan(&count)
					if err != nil {
						errors <- err
					}
				}
			}()
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalQueries := concurrency * iterations

		t.Logf("Executed %d concurrent queries in %v", totalQueries, duration)
		t.Logf("Average query time: %v", duration/time.Duration(totalQueries))

		// Check for errors
		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("Query error: %v", err)
		}

		if errorCount > 0 {
			t.Errorf("Encountered %d errors during concurrent queries", errorCount)
		}

		// Performance threshold: should complete within reasonable time
		maxDuration := 10 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent queries took too long: %v (max: %v)", duration, maxDuration)
		}
	})
}

// TestAPIResponseTime tests API endpoint response times
func TestAPIResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	campaign := createTestCampaign(t, env.DB, "perf-test")
	defer campaign.Cleanup()

	testCases := []struct {
		name        string
		endpoint    string
		method      string
		maxDuration time.Duration
	}{
		{"Health", "/health", "GET", 100 * time.Millisecond},
		{"Status", "/status", "GET", 100 * time.Millisecond},
		{"Campaigns_List", "/campaigns", "GET", 200 * time.Millisecond},
		{"Ideas_List", "/ideas", "GET", 500 * time.Millisecond},
		{"Workflows", "/workflows", "GET", 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()

			req := HTTPTestRequest{
				Method: tc.method,
				Path:   tc.endpoint,
			}

			w := makeHTTPRequest(env, req)
			duration := time.Since(start)

			if w.Code < 200 || w.Code >= 300 {
				t.Errorf("Request failed with status %d", w.Code)
			}

			t.Logf("%s response time: %v", tc.name, duration)

			if duration > tc.maxDuration {
				t.Errorf("%s took too long: %v (max: %v)", tc.name, duration, tc.maxDuration)
			}
		})
	}
}

// TestConcurrentRequestHandling tests concurrent request processing
func TestConcurrentRequestHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	campaign := createTestCampaign(t, env.DB, "concurrent-test")
	defer campaign.Cleanup()

	t.Run("ConcurrentCampaignCreation", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)
		results := make(chan int, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/campaigns",
					Body: map[string]interface{}{
						"name":        "Concurrent Campaign " + string(rune(index)),
						"description": "Test",
						"color":       "#FF5733",
					},
				}

				w := makeHTTPRequest(env, req)
				results <- w.Code

				if w.Code != http.StatusCreated {
					errors <- fmt.Errorf("request %d failed with status %d", index, w.Code)
				}
			}(i)
		}

		wg.Wait()
		close(errors)
		close(results)

		duration := time.Since(start)
		t.Logf("Created %d campaigns concurrently in %v", concurrency, duration)

		// Check results
		successCount := 0
		for code := range results {
			if code == http.StatusCreated {
				successCount++
			}
		}

		errorCount := 0
		for err := range errors {
			errorCount++
			t.Logf("Error: %v", err)
		}

		if successCount < concurrency {
			t.Errorf("Only %d/%d requests succeeded", successCount, concurrency)
		}

		// Should complete quickly
		maxDuration := 2 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent requests took too long: %v (max: %v)", duration, maxDuration)
		}
	})

	t.Run("ConcurrentIdeaRetrieval", func(t *testing.T) {
		// Create test ideas
		for i := 0; i < 20; i++ {
			createTestIdea(t, env.DB, campaign.ID, "Perf Idea "+string(rune(i)), "Content")
		}

		concurrency := 50
		iterations := 10
		var wg sync.WaitGroup
		durations := make(chan time.Duration, concurrency*iterations)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					reqStart := time.Now()

					req := HTTPTestRequest{
						Method: "GET",
						Path:   "/ideas?campaign_id=" + campaign.ID,
					}

					makeHTTPRequest(env, req)
					durations <- time.Since(reqStart)
				}
			}()
		}

		wg.Wait()
		close(durations)

		totalDuration := time.Since(start)
		totalRequests := concurrency * iterations

		// Calculate average
		var totalTime time.Duration
		var maxTime time.Duration
		var minTime time.Duration = 1 * time.Hour

		for d := range durations {
			totalTime += d
			if d > maxTime {
				maxTime = d
			}
			if d < minTime {
				minTime = d
			}
		}

		avgTime := totalTime / time.Duration(totalRequests)

		t.Logf("Processed %d concurrent idea retrievals in %v", totalRequests, totalDuration)
		t.Logf("Average response time: %v", avgTime)
		t.Logf("Min: %v, Max: %v", minTime, maxTime)

		// Performance thresholds
		if avgTime > 500*time.Millisecond {
			t.Errorf("Average response time too high: %v", avgTime)
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	campaign := createTestCampaign(t, env.DB, "memory-test")
	defer campaign.Cleanup()

	t.Run("LargeDatasetRetrieval", func(t *testing.T) {
		// Create many ideas
		ideaCount := 1000
		t.Logf("Creating %d test ideas...", ideaCount)

		for i := 0; i < ideaCount; i++ {
			createTestIdea(t, env.DB, campaign.ID, "Idea "+string(rune(i)), "Content for idea number "+string(rune(i)))
		}

		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/ideas?campaign_id=" + campaign.ID,
		}

		w := makeHTTPRequest(env, req)
		duration := time.Since(start)

		if w.Code != http.StatusOK {
			t.Errorf("Request failed with status %d", w.Code)
		}

		t.Logf("Retrieved large dataset in %v", duration)

		// Should handle large datasets reasonably
		maxDuration := 2 * time.Second
		if duration > maxDuration {
			t.Errorf("Large dataset retrieval too slow: %v (max: %v)", duration, maxDuration)
		}
	})
}

// TestPromptBuildingPerformance tests prompt generation efficiency
func TestPromptBuildingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)

	t.Run("ComplexPromptBuilding", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Performance Test Campaign",
			"description": "Testing prompt building performance with complex data",
			"context":     "This is additional context for the campaign",
		}

		// Create complex document set
		documents := make([]map[string]interface{}, 5)
		for i := 0; i < 5; i++ {
			documents[i] = map[string]interface{}{
				"original_name":  "document" + string(rune(i)) + ".pdf",
				"extracted_text": strings.Repeat("Complex document content. ", 100),
			}
		}

		// Create existing ideas
		existingIdeas := make([]map[string]interface{}, 3)
		for i := 0; i < 3; i++ {
			existingIdeas[i] = map[string]interface{}{
				"title":    "Existing Idea " + string(rune(i)),
				"content":  strings.Repeat("Detailed idea content. ", 50),
				"category": "innovation",
			}
		}

		req := GenerateIdeasRequest{
			Prompt: "Generate innovative ideas based on all available context",
			Count:  5,
		}

		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)
			if len(prompt) == 0 {
				t.Error("Generated empty prompt")
			}
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Built %d complex prompts in %v", iterations, duration)
		t.Logf("Average prompt building time: %v", avgTime)

		// Should be very fast
		if avgTime > 5*time.Millisecond {
			t.Errorf("Prompt building too slow: %v per prompt", avgTime)
		}
	})
}

// TestDatabaseOperationPerformance tests database operation efficiency
func TestDatabaseOperationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "db-perf-test")
	defer campaign.Cleanup()

	t.Run("IdeaStoragePerformance", func(t *testing.T) {
		ctx := context.Background()
		iterations := 100

		idea := GeneratedIdea{
			Title:               "Performance Test Idea",
			Description:         "Testing storage performance",
			Category:            "innovation",
			Tags:                []string{"test", "performance", "benchmark"},
			ImplementationNotes: "Performance testing notes",
		}

		start := time.Now()

		for i := 0; i < iterations; i++ {
			ideaID := uuid.New().String()
			err := processor.storeIdea(ctx, ideaID, campaign.ID, idea, "test-user")
			if err != nil {
				t.Errorf("Failed to store idea %d: %v", i, err)
			}
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Stored %d ideas in %v", iterations, duration)
		t.Logf("Average storage time: %v", avgTime)

		// Should be reasonably fast
		if avgTime > 50*time.Millisecond {
			t.Errorf("Idea storage too slow: %v per idea", avgTime)
		}
	})

	t.Run("CampaignDataRetrievalPerformance", func(t *testing.T) {
		ctx := context.Background()
		iterations := 500

		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := processor.getCampaignData(ctx, campaign.ID)
			if err != nil {
				t.Errorf("Failed to get campaign data: %v", err)
			}
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Retrieved campaign data %d times in %v", iterations, duration)
		t.Logf("Average retrieval time: %v", avgTime)

		// Should be very fast for simple queries
		if avgTime > 10*time.Millisecond {
			t.Errorf("Campaign retrieval too slow: %v per retrieval", avgTime)
		}
	})
}

// BenchmarkPromptBuilding benchmarks prompt building
func BenchmarkPromptBuilding(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)

	campaign := map[string]interface{}{
		"name":        "Benchmark Campaign",
		"description": "Benchmark test",
	}
	documents := []map[string]interface{}{}
	existingIdeas := []map[string]interface{}{}
	req := GenerateIdeasRequest{
		Prompt: "Generate ideas",
		Count:  1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)
	}
}

// BenchmarkIdeaStorage benchmarks idea storage
func BenchmarkIdeaStorage(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(&testing.T{}, env.DB, "benchmark-storage")
	defer campaign.Cleanup()

	ctx := context.Background()
	idea := GeneratedIdea{
		Title:       "Benchmark Idea",
		Description: "Testing",
		Category:    "innovation",
		Tags:        []string{"bench"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ideaID := uuid.New().String()
		processor.storeIdea(ctx, ideaID, campaign.ID, idea, "user")
	}
}
