package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestPerformance_ProfileCreation tests profile creation performance
func TestPerformance_ProfileCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name":         fmt.Sprintf("perf-profile-%d", i),
				"display_name": fmt.Sprintf("Performance Test Profile %d", i),
				"resources":    []interface{}{"postgres", "redis"},
				"scenarios":    []interface{}{"test-scenario"},
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		if rr.Code != 201 {
			t.Fatalf("Request %d returned status %d", i, rr.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Created %d profiles in %v (avg: %v per profile)", iterations, elapsed, avgTime)

	// Performance benchmark: should create profiles reasonably fast
	// Allow up to 100ms per profile on average (very generous for CI)
	if avgTime > 100*time.Millisecond {
		t.Logf("WARNING: Profile creation averaging %v (slower than expected)", avgTime)
	}

	// Verify all profiles were created
	profiles, err := env.Service.profileManager.ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}
	if len(profiles) != iterations {
		t.Errorf("Expected %d profiles, got %d", iterations, len(profiles))
	}
}

// TestPerformance_ProfileRetrieval tests profile retrieval performance
func TestPerformance_ProfileRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create test profiles
	numProfiles := 50
	for i := 0; i < numProfiles; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": fmt.Sprintf("retrieve-profile-%d", i),
			},
		}
		_, _ = makeHTTPRequest(env.Router, req)
	}

	// Test list performance
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		if rr.Code != 200 {
			t.Fatalf("Request %d returned status %d", i, rr.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Retrieved %d profiles %d times in %v (avg: %v per request)",
		numProfiles, iterations, elapsed, avgTime)

	// Should list profiles quickly (allow 50ms average)
	if avgTime > 50*time.Millisecond {
		t.Logf("WARNING: Profile listing averaging %v (slower than expected)", avgTime)
	}
}

// TestPerformance_ProfileUpdate tests profile update performance
func TestPerformance_ProfileUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create a test profile
	createReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/profiles",
		Body: map[string]interface{}{
			"name": "update-perf-test",
		},
	}
	_, err := makeHTTPRequest(env.Router, createReq)
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Test update performance
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/profiles/update-perf-test",
			Body: map[string]interface{}{
				"display_name": fmt.Sprintf("Updated %d", i),
				"description":  fmt.Sprintf("Update iteration %d", i),
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}

		if rr.Code != 200 {
			t.Fatalf("Request %d returned status %d", i, rr.Code)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Updated profile %d times in %v (avg: %v per update)",
		iterations, elapsed, avgTime)

	// Updates should be fast (allow 50ms average)
	if avgTime > 50*time.Millisecond {
		t.Logf("WARNING: Profile updates averaging %v (slower than expected)", avgTime)
	}
}

// TestPerformance_ConcurrentReads tests concurrent read performance
func TestPerformance_ConcurrentReads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create test profiles
	numProfiles := 20
	for i := 0; i < numProfiles; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": fmt.Sprintf("concurrent-read-%d", i),
			},
		}
		_, _ = makeHTTPRequest(env.Router, req)
	}

	// Test concurrent reads
	concurrency := 20
	requestsPerWorker := 50

	start := time.Now()
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*requestsPerWorker)

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				// Alternate between list and get operations
				var req HTTPTestRequest
				if i%2 == 0 {
					req = HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/profiles",
					}
				} else {
					profileIndex := i % numProfiles
					req = HTTPTestRequest{
						Method: "GET",
						Path:   fmt.Sprintf("/api/v1/profiles/concurrent-read-%d", profileIndex),
					}
				}

				rr, err := makeHTTPRequest(env.Router, req)
				if err != nil {
					errors <- fmt.Errorf("Worker %d request %d failed: %v", workerID, i, err)
					return
				}

				if rr.Code != 200 {
					errors <- fmt.Errorf("Worker %d request %d returned status %d",
						workerID, i, rr.Code)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)
	totalRequests := concurrency * requestsPerWorker
	avgTime := elapsed / time.Duration(totalRequests)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Got %d errors during concurrent reads", errorCount)
	}

	t.Logf("Completed %d concurrent read requests in %v (avg: %v per request)",
		totalRequests, elapsed, avgTime)

	// Concurrent reads should handle well (allow 10ms average)
	if avgTime > 10*time.Millisecond {
		t.Logf("WARNING: Concurrent reads averaging %v (slower than expected)", avgTime)
	}
}

// TestPerformance_ConcurrentWrites tests concurrent write performance
func TestPerformance_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	concurrency := 10
	profilesPerWorker := 10

	start := time.Now()
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*profilesPerWorker)
	successes := make(chan bool, concurrency*profilesPerWorker)

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < profilesPerWorker; i++ {
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/profiles",
					Body: map[string]interface{}{
						"name": fmt.Sprintf("concurrent-write-w%d-p%d", workerID, i),
					},
				}

				rr, err := makeHTTPRequest(env.Router, req)
				if err != nil {
					errors <- fmt.Errorf("Worker %d profile %d failed: %v", workerID, i, err)
					return
				}

				if rr.Code != 201 {
					errors <- fmt.Errorf("Worker %d profile %d returned status %d",
						workerID, i, rr.Code)
					return
				}

				successes <- true
			}
		}(worker)
	}

	wg.Wait()
	close(errors)
	close(successes)

	elapsed := time.Since(start)
	totalWrites := concurrency * profilesPerWorker

	// Count successes
	successCount := 0
	for range successes {
		successCount++
	}

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
	}

	avgTime := elapsed / time.Duration(successCount)

	t.Logf("Completed %d concurrent write requests in %v (avg: %v per write, %d errors)",
		successCount, elapsed, avgTime, errorCount)

	if errorCount > 0 {
		t.Fatalf("Got %d errors during concurrent writes", errorCount)
	}

	// Verify all profiles were created
	profiles, err := env.Service.profileManager.ListProfiles()
	if err != nil {
		t.Fatalf("Failed to list profiles: %v", err)
	}
	if len(profiles) != totalWrites {
		t.Errorf("Expected %d profiles, got %d", totalWrites, len(profiles))
	}

	// Concurrent writes should handle reasonably (allow 100ms average due to DB locks)
	if avgTime > 100*time.Millisecond {
		t.Logf("WARNING: Concurrent writes averaging %v (slower than expected)", avgTime)
	}
}

// TestPerformance_DatabaseConnectionPool tests connection pool behavior
func TestPerformance_DatabaseConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create a profile for testing
	createReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/profiles",
		Body: map[string]interface{}{
			"name": "pool-test",
		},
	}
	_, _ = makeHTTPRequest(env.Router, createReq)

	// Simulate many concurrent connections
	concurrency := 50 // More than max DB connections
	requestsPerWorker := 20

	start := time.Now()
	var wg sync.WaitGroup
	errors := make(chan error, concurrency*requestsPerWorker)

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < requestsPerWorker; i++ {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/profiles/pool-test",
				}

				rr, err := makeHTTPRequest(env.Router, req)
				if err != nil {
					errors <- fmt.Errorf("Worker %d request %d failed: %v", workerID, i, err)
					return
				}

				if rr.Code != 200 {
					errors <- fmt.Errorf("Worker %d request %d returned status %d",
						workerID, i, rr.Code)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	close(errors)

	elapsed := time.Since(start)
	totalRequests := concurrency * requestsPerWorker

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Error(err)
		errorCount++
		if errorCount > 10 {
			t.Fatal("Too many errors, stopping test")
		}
	}

	if errorCount > 0 {
		t.Logf("Warning: Got %d errors during connection pool test", errorCount)
	}

	avgTime := elapsed / time.Duration(totalRequests)

	t.Logf("Completed %d requests with %d concurrent workers in %v (avg: %v per request)",
		totalRequests, concurrency, elapsed, avgTime)

	// Connection pooling should handle high concurrency reasonably
	if avgTime > 20*time.Millisecond {
		t.Logf("WARNING: Connection pool requests averaging %v (slower than expected)", avgTime)
	}
}

// TestPerformance_LargeProfileData tests handling of profiles with large data
func TestPerformance_LargeProfileData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create profile with large arrays
	largeResourceList := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		largeResourceList[i] = fmt.Sprintf("resource-%d", i)
	}

	largeScenarioList := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		largeScenarioList[i] = fmt.Sprintf("scenario-%d", i)
	}

	largeURLList := make([]interface{}, 50)
	for i := 0; i < 50; i++ {
		largeURLList[i] = fmt.Sprintf("http://localhost:%d/dashboard", 3000+i)
	}

	largeEnvVars := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeEnvVars[fmt.Sprintf("VAR_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	// Create large metadata
	largeMetadata := make(map[string]interface{})
	for i := 0; i < 50; i++ {
		largeMetadata[fmt.Sprintf("key_%d", i)] = map[string]interface{}{
			"subkey1": fmt.Sprintf("value1_%d", i),
			"subkey2": fmt.Sprintf("value2_%d", i),
			"array":   []interface{}{i, i + 1, i + 2},
		}
	}

	start := time.Now()

	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/profiles",
		Body: map[string]interface{}{
			"name":             "large-profile",
			"resources":        largeResourceList,
			"scenarios":        largeScenarioList,
			"auto_browser":     largeURLList,
			"environment_vars": largeEnvVars,
			"metadata":         largeMetadata,
		},
	}

	rr, err := makeHTTPRequest(env.Router, req)
	if err != nil {
		t.Fatalf("Failed to create large profile: %v", err)
	}

	createElapsed := time.Since(start)

	if rr.Code != 201 {
		t.Fatalf("Expected status 201, got %d", rr.Code)
	}

	t.Logf("Created large profile in %v", createElapsed)

	// Test retrieval performance
	start = time.Now()

	getReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/profiles/large-profile",
	}

	rr, err = makeHTTPRequest(env.Router, getReq)
	if err != nil {
		t.Fatalf("Failed to retrieve large profile: %v", err)
	}

	retrieveElapsed := time.Since(start)

	if rr.Code != 200 {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	t.Logf("Retrieved large profile in %v", retrieveElapsed)

	// Large profile operations should still be reasonably fast
	if createElapsed > 500*time.Millisecond {
		t.Logf("WARNING: Large profile creation took %v (slower than expected)", createElapsed)
	}
	if retrieveElapsed > 100*time.Millisecond {
		t.Logf("WARNING: Large profile retrieval took %v (slower than expected)", retrieveElapsed)
	}

	// Verify data integrity
	response := assertJSONResponse(t, rr, 200)
	resources, ok := response["resources"].([]interface{})
	if !ok || len(resources) != 100 {
		t.Errorf("Expected 100 resources, got %v", len(resources))
	}
}

// TestPerformance_ActiveProfileOperations tests active profile switching performance
func TestPerformance_ActiveProfileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	env := setupTestDB(t)
	defer env.Cleanup()

	cleanup := setupTestLogger()
	defer cleanup()

	cleanupProfiles(env)

	// Create multiple profiles
	numProfiles := 10
	profileIDs := make([]string, numProfiles)

	for i := 0; i < numProfiles; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": fmt.Sprintf("active-perf-%d", i),
			},
		}

		rr, _ := makeHTTPRequest(env.Router, req)
		response := assertJSONResponse(t, rr, 201)
		profileIDs[i] = response["id"].(string)
	}

	pm := env.Service.profileManager

	// Test rapid active profile switching
	iterations := 50
	start := time.Now()

	for i := 0; i < iterations; i++ {
		profileID := profileIDs[i%numProfiles]

		// Set active
		err := pm.SetActiveProfile(profileID)
		if err != nil {
			t.Fatalf("Iteration %d: SetActiveProfile failed: %v", i, err)
		}

		// Get active
		active, err := pm.GetActiveProfile()
		if err != nil {
			t.Fatalf("Iteration %d: GetActiveProfile failed: %v", i, err)
		}
		if active == nil || active.ID != profileID {
			t.Fatalf("Iteration %d: Active profile mismatch", i)
		}

		// Clear active
		err = pm.ClearActiveProfile()
		if err != nil {
			t.Fatalf("Iteration %d: ClearActiveProfile failed: %v", i, err)
		}
	}

	elapsed := time.Since(start)
	avgTime := elapsed / time.Duration(iterations)

	t.Logf("Completed %d active profile operations in %v (avg: %v per cycle)",
		iterations, elapsed, avgTime)

	// Active profile operations should be fast
	if avgTime > 50*time.Millisecond {
		t.Logf("WARNING: Active profile operations averaging %v (slower than expected)", avgTime)
	}
}

// BenchmarkProfileCreation benchmarks profile creation
func BenchmarkProfileCreation(b *testing.B) {
	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	cleanupProfiles(env)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": fmt.Sprintf("bench-profile-%d", i),
			},
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil || rr.Code != 201 {
			b.Fatalf("Profile creation failed at iteration %d", i)
		}
	}
}

// BenchmarkProfileRetrieval benchmarks profile retrieval
func BenchmarkProfileRetrieval(b *testing.B) {
	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	cleanupProfiles(env)

	// Create a test profile
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/profiles",
		Body: map[string]interface{}{
			"name": "bench-retrieve-profile",
		},
	}
	_, _ = makeHTTPRequest(env.Router, req)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles/bench-retrieve-profile",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil || rr.Code != 200 {
			b.Fatalf("Profile retrieval failed at iteration %d", i)
		}
	}
}

// BenchmarkProfileList benchmarks listing all profiles
func BenchmarkProfileList(b *testing.B) {
	env := setupTestDB(&testing.T{})
	defer env.Cleanup()

	cleanupProfiles(env)

	// Create multiple profiles
	for i := 0; i < 20; i++ {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": fmt.Sprintf("bench-list-profile-%d", i),
			},
		}
		_, _ = makeHTTPRequest(env.Router, req)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		rr, err := makeHTTPRequest(env.Router, req)
		if err != nil || rr.Code != 200 {
			b.Fatalf("Profile list failed at iteration %d", i)
		}
	}
}
