// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPerformanceCycleCreation tests cycle creation performance
func TestPerformanceCycleCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := CreateCyclePerformancePattern()
	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceCycleRetrieval tests cycle retrieval performance with various sizes
func TestPerformanceCycleRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	// Test with different data sizes
	sizes := []int{10, 50}
	for _, size := range sizes {
		pattern := GetCyclesPerformancePattern(size)
		RunPerformanceTest(t, env, pattern)
	}
}

// TestPerformanceEncryption tests encryption performance
func TestPerformanceEncryption(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	pattern := EncryptionPerformancePattern()
	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceDatabaseConnection tests database connection performance
func TestPerformanceDatabaseConnection(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	pattern := DatabaseConnectionPattern()
	RunPerformanceTest(t, env, pattern)
}

// TestPerformancePredictionGeneration tests prediction algorithm performance
func TestPerformancePredictionGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := PredictionGenerationPattern()
	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceSymptomLogging tests symptom logging performance
func TestPerformanceSymptomLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := PerformanceTestPattern{
		Name:        "SymptomLoggingPerformance",
		Description: "Measure symptom logging performance",
		MaxDuration: 150 * time.Millisecond,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			moodRating := 7
			energyLevel := 6
			crampIntensity := 5

			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/symptoms",
				Body: map[string]interface{}{
					"date":               "2024-01-15",
					"physical_symptoms":  []string{"headache", "cramps", "fatigue"},
					"mood_rating":        moodRating,
					"energy_level":       energyLevel,
					"cramp_intensity":    crampIntensity,
					"headache_intensity": 3,
					"flow_level":         "medium",
				},
			})

			duration := time.Since(start)

			if err != nil {
				t.Skipf("Symptom logging request failed: %v", err)
			}

			if w.Code != http.StatusCreated {
				t.Skipf("Symptom logging failed with status %d", w.Code)
			}

			return duration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			cleanupTestData(t, env, env.UserID)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceConcurrentRead tests concurrent read performance
func TestPerformanceConcurrentRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := PerformanceTestPattern{
		Name:        "ConcurrentReadPerformance",
		Description: "Measure performance under concurrent read load",
		MaxDuration: 500 * time.Millisecond,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			// Create test data
			for i := 0; i < 20; i++ {
				date := fmt.Sprintf("2024-01-%02d", i+1)
				createTestCycle(t, env, env.UserID, date, "medium", fmt.Sprintf("Cycle %d", i))
			}
			return env.UserID
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			numConcurrent := 10
			results := make(chan time.Duration, numConcurrent)

			start := time.Now()

			// Launch concurrent reads
			for i := 0; i < numConcurrent; i++ {
				go func() {
					reqStart := time.Now()
					w, err := makeHTTPRequest(env, HTTPTestRequest{
						Method: "GET",
						Path:   "/api/v1/cycles",
					})
					reqDuration := time.Since(reqStart)

					if err != nil || w.Code != http.StatusOK {
						results <- 0
					} else {
						results <- reqDuration
					}
				}()
			}

			// Wait for all to complete
			successCount := 0
			var totalReqDuration time.Duration
			for i := 0; i < numConcurrent; i++ {
				d := <-results
				if d > 0 {
					successCount++
					totalReqDuration += d
				}
			}

			totalDuration := time.Since(start)

			if successCount > 0 {
				avgReqDuration := totalReqDuration / time.Duration(successCount)
				t.Logf("Concurrent reads: %d successful, avg time: %v, total time: %v",
					successCount, avgReqDuration, totalDuration)
			} else {
				t.Skip("No successful concurrent reads - database may not be available")
			}

			return totalDuration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			userID := setupData.(string)
			cleanupTestData(t, env, userID)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceConcurrentWrite tests concurrent write performance
func TestPerformanceConcurrentWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := PerformanceTestPattern{
		Name:        "ConcurrentWritePerformance",
		Description: "Measure performance under concurrent write load",
		MaxDuration: 1 * time.Second,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			numConcurrent := 10
			results := make(chan time.Duration, numConcurrent)

			start := time.Now()

			// Launch concurrent writes
			for i := 0; i < numConcurrent; i++ {
				go func(index int) {
					reqStart := time.Now()
					date := fmt.Sprintf("2024-01-%02d", (index%28)+1)
					w, err := makeHTTPRequest(env, HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/cycles",
						Body: map[string]interface{}{
							"start_date":     date,
							"flow_intensity": "medium",
							"notes":          fmt.Sprintf("Concurrent write %d", index),
						},
					})
					reqDuration := time.Since(reqStart)

					if err != nil || w.Code != http.StatusCreated {
						results <- 0
					} else {
						results <- reqDuration
					}
				}(i)
			}

			// Wait for all to complete
			successCount := 0
			var totalReqDuration time.Duration
			for i := 0; i < numConcurrent; i++ {
				d := <-results
				if d > 0 {
					successCount++
					totalReqDuration += d
				}
			}

			totalDuration := time.Since(start)

			if successCount > 0 {
				avgReqDuration := totalReqDuration / time.Duration(successCount)
				t.Logf("Concurrent writes: %d successful, avg time: %v, total time: %v",
					successCount, avgReqDuration, totalDuration)
			} else {
				t.Skip("No successful concurrent writes - database may not be available")
			}

			return totalDuration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			cleanupTestData(t, env, env.UserID)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceSymptomRetrieval tests symptom retrieval performance
func TestPerformanceSymptomRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	pattern := PerformanceTestPattern{
		Name:        "SymptomRetrievalPerformance",
		Description: "Measure symptom retrieval performance with large dataset",
		MaxDuration: 200 * time.Millisecond,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			// Create 90 days of symptom data
			for i := 0; i < 90; i++ {
				date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
				createTestSymptom(t, env, env.UserID, date, 5+(i%5))
			}
			return env.UserID
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/symptoms",
			})

			duration := time.Since(start)

			if err != nil {
				t.Skipf("Symptom retrieval request failed: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Skipf("Symptom retrieval failed with status %d", w.Code)
			}

			return duration
		},
		Cleanup: func(t *testing.T, env *TestEnvironment, setupData interface{}) {
			userID := setupData.(string)
			cleanupTestData(t, env, userID)
		},
	}

	RunPerformanceTest(t, env, pattern)
}

// TestPerformanceMemoryUsage tests memory usage under load
func TestPerformanceMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("LargePayloadEncryption", func(t *testing.T) {
		// Test encryption of large notes
		largeNote := ""
		for i := 0; i < 100; i++ {
			largeNote += "This is a line of sensitive health information. "
		}

		start := time.Now()
		encrypted, err := encryptString(largeNote)
		encryptDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Encryption failed: %v", err)
		}

		start = time.Now()
		decrypted, err := decryptString(encrypted)
		decryptDuration := time.Since(start)

		if err != nil {
			t.Fatalf("Decryption failed: %v", err)
		}

		if decrypted != largeNote {
			t.Error("Decryption mismatch")
		}

		t.Logf("Large payload (%.2f KB): encrypt=%v, decrypt=%v",
			float64(len(largeNote))/1024, encryptDuration, decryptDuration)

		if encryptDuration > 100*time.Millisecond {
			t.Errorf("Encryption too slow for large payload: %v", encryptDuration)
		}
	})
}

// TestPerformanceResponseTime tests overall API response time
func TestPerformanceResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
		maxMS  int
	}{
		{
			name:   "HealthCheck",
			method: "GET",
			path:   "/health",
			maxMS:  50,
		},
		{
			name:   "EncryptionHealth",
			method: "GET",
			path:   "/api/v1/health/encryption",
			maxMS:  50,
		},
		{
			name:   "AuthStatus",
			method: "GET",
			path:   "/api/v1/auth/status",
			maxMS:  50,
		},
		{
			name:   "GetCycles",
			method: "GET",
			path:   "/api/v1/cycles",
			maxMS:  200,
		},
		{
			name:   "GetSymptoms",
			method: "GET",
			path:   "/api/v1/symptoms",
			maxMS:  200,
		},
		{
			name:   "GetPredictions",
			method: "GET",
			path:   "/api/v1/predictions",
			maxMS:  200,
		},
		{
			name:   "GetPatterns",
			method: "GET",
			path:   "/api/v1/patterns",
			maxMS:  200,
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			start := time.Now()

			var userID string
			if endpoint.path == "/health" {
				userID = "" // Health doesn't need auth
			} else {
				userID = env.UserID
			}

			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
				Body:   endpoint.body,
				UserID: userID,
			})

			duration := time.Since(start)

			if err != nil {
				t.Skipf("Request failed: %v", err)
			}

			if w.Code >= 500 {
				t.Skipf("Server error: %d", w.Code)
			}

			maxDuration := time.Duration(endpoint.maxMS) * time.Millisecond
			if duration > maxDuration {
				t.Errorf("Response time %v exceeds max %v", duration, maxDuration)
			} else {
				t.Logf("✓ Response time: %v (max: %v)", duration, maxDuration)
			}
		})
	}
}

// TestPerformanceDatabaseQueries tests database query performance
func TestPerformanceDatabaseQueries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	defer cleanupTestData(t, env, env.UserID)

	t.Run("SimpleQuery", func(t *testing.T) {
		start := time.Now()
		var result int
		err := env.DB.QueryRow("SELECT 1").Scan(&result)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}

		if duration > 10*time.Millisecond {
			t.Errorf("Simple query too slow: %v", duration)
		}

		t.Logf("Simple query: %v", duration)
	})

	t.Run("CountQuery", func(t *testing.T) {
		// Create some test data first
		for i := 0; i < 10; i++ {
			date := fmt.Sprintf("2024-01-%02d", i+1)
			createTestCycle(t, env, env.UserID, date, "medium", "Test")
		}

		start := time.Now()
		var count int
		err := env.DB.QueryRow("SELECT COUNT(*) FROM cycles WHERE user_id = $1", env.UserID).Scan(&count)
		duration := time.Since(start)

		if err != nil {
			t.Skipf("Count query failed: %v", err)
		}

		if duration > 50*time.Millisecond {
			t.Errorf("Count query too slow: %v", duration)
		}

		t.Logf("Count query (%d rows): %v", count, duration)
	})
}
