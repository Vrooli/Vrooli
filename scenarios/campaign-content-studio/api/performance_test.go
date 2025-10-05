// +build testing

package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestPerformance_HealthEndpoint tests the performance of the health endpoint
func TestPerformance_HealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		maxDuration := 50 * time.Millisecond

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Health endpoint took %v, expected < %v", duration, maxDuration)
		} else {
			t.Logf("Health endpoint responded in %v", duration)
		}
	})

	t.Run("Concurrency_100Requests", func(t *testing.T) {
		concurrency := 10
		totalRequests := 100
		maxAvgDuration := 100 * time.Millisecond

		var wg sync.WaitGroup
		var successCount, errorCount int64
		var totalDuration int64

		start := time.Now()
		sem := make(chan struct{}, concurrency)

		for i := 0; i < totalRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				reqStart := time.Now()
				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				})
				reqDuration := time.Since(reqStart)

				atomic.AddInt64(&totalDuration, int64(reqDuration))

				if w.Code == 200 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}()
		}

		wg.Wait()
		elapsed := time.Since(start)

		avgDuration := time.Duration(totalDuration / int64(totalRequests))
		throughput := float64(totalRequests) / elapsed.Seconds()

		t.Logf("Concurrent health checks: %d requests in %v", totalRequests, elapsed)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)
		t.Logf("Average response time: %v", avgDuration)
		t.Logf("Throughput: %.2f requests/second", throughput)

		if errorCount > 0 {
			t.Errorf("Expected 0 errors, got %d", errorCount)
		}

		if avgDuration > maxAvgDuration {
			t.Errorf("Average duration %v exceeded max %v", avgDuration, maxAvgDuration)
		}
	})
}

// TestPerformance_ListCampaigns tests the performance of listing campaigns
func TestPerformance_ListCampaigns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ResponseTime_EmptyList", func(t *testing.T) {
		maxDuration := 100 * time.Millisecond

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		})
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("List campaigns took %v, expected < %v", duration, maxDuration)
		} else {
			t.Logf("List campaigns (empty) responded in %v", duration)
		}
	})

	t.Run("ResponseTime_WithData", func(t *testing.T) {
		// Create 10 test campaigns
		campaigns := make([]*TestCampaign, 10)
		for i := 0; i < 10; i++ {
			campaigns[i] = setupTestCampaign(t, env, fmt.Sprintf("perf-test-%d", i))
			defer campaigns[i].Cleanup()
		}

		maxDuration := 200 * time.Millisecond

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		})
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("List campaigns (10 items) took %v, expected < %v", duration, maxDuration)
		} else {
			t.Logf("List campaigns (10 items) responded in %v", duration)
		}
	})
}

// TestPerformance_CreateCampaign tests the performance of creating campaigns
func TestPerformance_CreateCampaign(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ResponseTime", func(t *testing.T) {
		maxDuration := 200 * time.Millisecond

		campaignReq := TestData.CampaignRequest("Performance Test", "Test Description")

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   campaignReq,
		})
		duration := time.Since(start)

		if w.Code != 201 {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("Create campaign took %v, expected < %v", duration, maxDuration)
		} else {
			t.Logf("Create campaign responded in %v", duration)
		}
	})

	t.Run("BulkCreate_Sequential", func(t *testing.T) {
		count := 50
		maxAvgDuration := 200 * time.Millisecond

		var totalDuration time.Duration
		var successCount int

		for i := 0; i < count; i++ {
			campaignReq := TestData.CampaignRequest(
				fmt.Sprintf("Bulk Test %d", i),
				fmt.Sprintf("Description %d", i),
			)

			start := time.Now()
			w := makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: "POST",
				Path:   "/campaigns",
				Body:   campaignReq,
			})
			duration := time.Since(start)
			totalDuration += duration

			if w.Code == 201 {
				successCount++
			}
		}

		avgDuration := totalDuration / time.Duration(count)
		t.Logf("Created %d campaigns sequentially in %v", count, totalDuration)
		t.Logf("Success rate: %d/%d", successCount, count)
		t.Logf("Average time per campaign: %v", avgDuration)

		if successCount != count {
			t.Errorf("Expected %d successes, got %d", count, successCount)
		}

		if avgDuration > maxAvgDuration {
			t.Errorf("Average creation time %v exceeded max %v", avgDuration, maxAvgDuration)
		}
	})

	t.Run("BulkCreate_Concurrent", func(t *testing.T) {
		count := 50
		concurrency := 5
		maxTotalDuration := 5 * time.Second

		var wg sync.WaitGroup
		var successCount, errorCount int64
		sem := make(chan struct{}, concurrency)

		start := time.Now()

		for i := 0; i < count; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				campaignReq := TestData.CampaignRequest(
					fmt.Sprintf("Concurrent Test %d", index),
					fmt.Sprintf("Description %d", index),
				)

				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/campaigns",
					Body:   campaignReq,
				})

				if w.Code == 201 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(start)

		t.Logf("Created %d campaigns concurrently (max %d concurrent) in %v",
			count, concurrency, totalDuration)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)
		t.Logf("Throughput: %.2f campaigns/second", float64(count)/totalDuration.Seconds())

		if errorCount > 0 {
			t.Errorf("Expected 0 errors, got %d", errorCount)
		}

		if totalDuration > maxTotalDuration {
			t.Errorf("Total duration %v exceeded max %v", totalDuration, maxTotalDuration)
		}
	})
}

// TestPerformance_ListDocuments tests the performance of listing documents
func TestPerformance_ListDocuments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	campaign := setupTestCampaign(t, env, "perf-test-docs")
	defer campaign.Cleanup()

	t.Run("ResponseTime_WithDocuments", func(t *testing.T) {
		// Create 20 test documents
		docs := make([]*TestDocument, 20)
		for i := 0; i < 20; i++ {
			docs[i] = setupTestDocument(t, env, campaign.Campaign.ID, fmt.Sprintf("doc%d.pdf", i))
			defer docs[i].Cleanup()
		}

		maxDuration := 200 * time.Millisecond

		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/campaigns/%s/documents", campaign.Campaign.ID),
			URLVars: map[string]string{"campaignId": campaign.Campaign.ID.String()},
		})
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if duration > maxDuration {
			t.Errorf("List documents (20 items) took %v, expected < %v", duration, maxDuration)
		} else {
			t.Logf("List documents (20 items) responded in %v", duration)
		}
	})
}

// TestPerformance_DatabaseConnections tests database connection handling
func TestPerformance_DatabaseConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConcurrentDatabaseAccess", func(t *testing.T) {
		concurrency := 20
		iterations := 100
		maxTotalDuration := 10 * time.Second

		var wg sync.WaitGroup
		var successCount, errorCount int64
		sem := make(chan struct{}, concurrency)

		start := time.Now()

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				// Alternate between read and write operations
				if index%2 == 0 {
					// Read operation
					w := makeHTTPRequest(env.Router, HTTPTestRequest{
						Method: "GET",
						Path:   "/campaigns",
					})
					if w.Code == 200 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				} else {
					// Write operation
					campaignReq := TestData.CampaignRequest(
						fmt.Sprintf("DB Test %d", index),
						"Test",
					)
					w := makeHTTPRequest(env.Router, HTTPTestRequest{
						Method: "POST",
						Path:   "/campaigns",
						Body:   campaignReq,
					})
					if w.Code == 201 {
						atomic.AddInt64(&successCount, 1)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(start)

		t.Logf("Executed %d concurrent database operations in %v", iterations, totalDuration)
		t.Logf("Success: %d, Errors: %d", successCount, errorCount)
		t.Logf("Throughput: %.2f operations/second", float64(iterations)/totalDuration.Seconds())

		if errorCount > int64(iterations/10) { // Allow up to 10% errors
			t.Errorf("Too many errors: %d out of %d", errorCount, iterations)
		}

		if totalDuration > maxTotalDuration {
			t.Errorf("Total duration %v exceeded max %v", totalDuration, maxTotalDuration)
		}
	})
}

// BenchmarkHealthEndpoint benchmarks the health endpoint
func BenchmarkHealthEndpoint(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
	}
}

// BenchmarkListCampaigns benchmarks the list campaigns endpoint
func BenchmarkListCampaigns(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	// Create some test data
	for i := 0; i < 10; i++ {
		setupTestCampaign(&testing.T{}, env, fmt.Sprintf("benchmark-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/campaigns",
		})
	}
}

// BenchmarkCreateCampaign benchmarks campaign creation
func BenchmarkCreateCampaign(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		campaignReq := TestData.CampaignRequest(
			fmt.Sprintf("Benchmark %d", i),
			"Benchmark test",
		)
		makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/campaigns",
			Body:   campaignReq,
		})
	}
}
