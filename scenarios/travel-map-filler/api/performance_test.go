package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHandlerPerformance tests API endpoint performance
func TestHandlerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Health_Performance", testHealthPerformance)
	t.Run("Travels_Performance", func(t *testing.T) { testTravelsPerformance(t, testDB) })
	t.Run("Stats_Performance", func(t *testing.T) { testStatsPerformance(t, testDB) })
	t.Run("Achievements_Performance", func(t *testing.T) { testAchievementsPerformance(t, testDB) })
	t.Run("BucketList_Performance", func(t *testing.T) { testBucketListPerformance(t, testDB) })
	t.Run("AddTravel_Performance", func(t *testing.T) { testAddTravelPerformance(t, testDB) })
	t.Run("Search_Performance", testSearchPerformance)
}

func testHealthPerformance(t *testing.T) {
	pattern := PerformanceTestPattern{
		Name:        "HealthEndpoint",
		Description: "Health check should respond quickly",
		MaxDuration: 50 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			healthHandler(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)
}

func testTravelsPerformance(t *testing.T, testDB *TestDB) {
	// Create test data with varying sizes
	userID := "test_perf_travels"
	datasets := []struct {
		name  string
		count int
		max   time.Duration
	}{
		{"SmallDataset_10", 10, 100 * time.Millisecond},
		{"MediumDataset_50", 50, 200 * time.Millisecond},
		{"LargeDataset_100", 100, 300 * time.Millisecond},
	}

	for _, dataset := range datasets {
		t.Run(dataset.name, func(t *testing.T) {
			// Setup: Create test data
			testUserID := userID + "_" + dataset.name
			for i := 0; i < dataset.count; i++ {
				createTestTravel(t, testDB.DB, testUserID)
			}

			pattern := PerformanceTestPattern{
				Name:        "TravelsEndpoint_" + dataset.name,
				Description: "Travels listing performance with " + dataset.name,
				MaxDuration: dataset.max,
				Execute: func(t *testing.T, setupData interface{}) time.Duration {
					start := time.Now()

					req := httptest.NewRequest("GET", "/api/travels?user_id="+testUserID, nil)
					w := httptest.NewRecorder()

					travelsHandler(w, req)

					duration := time.Since(start)

					if w.Code != 200 {
						t.Errorf("Expected status 200, got %d", w.Code)
					}

					return duration
				},
			}

			RunPerformanceTest(t, pattern)
		})
	}

	// Test with filters
	t.Run("WithFilters", func(t *testing.T) {
		testUserID := userID + "_filters"
		for i := 0; i < 20; i++ {
			createTestTravel(t, testDB.DB, testUserID)
		}

		pattern := PerformanceTestPattern{
			Name:        "TravelsEndpoint_WithFilters",
			Description: "Travels with year and type filters",
			MaxDuration: 150 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()

				req := httptest.NewRequest("GET", "/api/travels?user_id="+testUserID+"&year=2024&type=vacation", nil)
				w := httptest.NewRecorder()

				travelsHandler(w, req)

				return time.Since(start)
			},
		}

		RunPerformanceTest(t, pattern)
	})
}

func testStatsPerformance(t *testing.T, testDB *TestDB) {
	userID := "test_perf_stats"

	// Create diverse data for stats calculation
	for i := 0; i < 30; i++ {
		createTestTravel(t, testDB.DB, userID)
	}

	pattern := PerformanceTestPattern{
		Name:        "StatsEndpoint",
		Description: "Stats calculation performance",
		MaxDuration: 200 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
			w := httptest.NewRecorder()

			statsHandler(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)

	// Test with no existing stats (calculation path)
	t.Run("Calculation", func(t *testing.T) {
		newUserID := userID + "_calc"
		for i := 0; i < 20; i++ {
			createTestTravel(t, testDB.DB, newUserID)
		}

		pattern := PerformanceTestPattern{
			Name:        "StatsCalculation",
			Description: "Stats calculation from scratch",
			MaxDuration: 250 * time.Millisecond,
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				start := time.Now()

				req := httptest.NewRequest("GET", "/api/stats?user_id="+newUserID, nil)
				w := httptest.NewRecorder()

				statsHandler(w, req)

				return time.Since(start)
			},
		}

		RunPerformanceTest(t, pattern)
	})
}

func testAchievementsPerformance(t *testing.T, testDB *TestDB) {
	userID := "test_perf_achievements"

	// Create multiple achievements
	achievementTypes := []string{"first_trip", "five_countries", "ten_countries", "explorer", "adventurer"}
	for _, achType := range achievementTypes {
		createTestAchievement(t, testDB.DB, userID, achType)
	}

	pattern := PerformanceTestPattern{
		Name:        "AchievementsEndpoint",
		Description: "Achievements listing performance",
		MaxDuration: 100 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
			w := httptest.NewRecorder()

			achievementsHandler(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)
}

func testBucketListPerformance(t *testing.T, testDB *TestDB) {
	userID := "test_perf_bucket"

	// Create bucket list items
	for i := 0; i < 20; i++ {
		createTestBucketItem(t, testDB.DB, userID)
	}

	pattern := PerformanceTestPattern{
		Name:        "BucketListEndpoint",
		Description: "Bucket list retrieval performance",
		MaxDuration: 150 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/bucket-list?user_id="+userID, nil)
			w := httptest.NewRecorder()

			bucketListHandler(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)
}

func testAddTravelPerformance(t *testing.T, testDB *TestDB) {
	pattern := PerformanceTestPattern{
		Name:        "AddTravelEndpoint",
		Description: "Add travel performance",
		MaxDuration: 200 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			travelData := TestData.TravelRequest("test_perf_add", "Performance City")
			travelJSON, _ := json.Marshal(travelData)

			start := time.Now()

			req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			addTravelHandler(w, req)

			duration := time.Since(start)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)

	// Test achievement checking performance
	t.Run("WithAchievementCheck", func(t *testing.T) {
		userID := "test_perf_add_ach"

		pattern := PerformanceTestPattern{
			Name:        "AddTravel_WithAchievements",
			Description: "Add travel with achievement checking",
			MaxDuration: 250 * time.Millisecond,
			Setup: func(t *testing.T) interface{} {
				// Create some existing travels
				for i := 0; i < 5; i++ {
					createTestTravel(t, testDB.DB, userID)
				}
				return userID
			},
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				userID := setupData.(string)
				travelData := TestData.TravelRequest(userID, "Achievement City")
				travelJSON, _ := json.Marshal(travelData)

				start := time.Now()

				req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
				req.Header.Set("Content-Type", "application/json")
				w := httptest.NewRecorder()

				addTravelHandler(w, req)

				return time.Since(start)
			},
		}

		RunPerformanceTest(t, pattern)
	})
}

func testSearchPerformance(t *testing.T) {
	pattern := PerformanceTestPattern{
		Name:        "SearchEndpoint",
		Description: "Search travel performance",
		MaxDuration: 300 * time.Millisecond, // Higher max due to n8n call
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/travels/search?q=Paris", nil)
			w := httptest.NewRecorder()

			searchTravelsHandler(w, req)

			duration := time.Since(start)

			// Allow 200 or 500 (if n8n unavailable)
			if w.Code != 200 && w.Code != 500 {
				t.Errorf("Expected status 200 or 500, got %d", w.Code)
			}

			return duration
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestThroughput tests request throughput
func TestThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("ReadThroughput", func(t *testing.T) {
		userID := "test_throughput_read"
		createTestTravel(t, testDB.DB, userID)

		requestCount := 100
		duration := time.Duration(0)

		start := time.Now()
		for i := 0; i < requestCount; i++ {
			req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
		duration = time.Since(start)

		avgDuration := duration / time.Duration(requestCount)
		throughput := float64(requestCount) / duration.Seconds()

		t.Logf("Read throughput: %.2f req/s (avg: %v per request)", throughput, avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Average request time too high: %v", avgDuration)
		}
	})

	t.Run("WriteThroughput", func(t *testing.T) {
		userID := "test_throughput_write"
		requestCount := 50

		start := time.Now()
		for i := 0; i < requestCount; i++ {
			travelData := TestData.TravelRequest(userID, "Throughput Test")
			travelJSON, _ := json.Marshal(travelData)

			req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			addTravelHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
		duration := time.Since(start)

		avgDuration := duration / time.Duration(requestCount)
		throughput := float64(requestCount) / duration.Seconds()

		t.Logf("Write throughput: %.2f req/s (avg: %v per request)", throughput, avgDuration)

		if avgDuration > 200*time.Millisecond {
			t.Errorf("Average write time too high: %v", avgDuration)
		}
	})
}

// TestMemoryUsage tests memory consumption patterns
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("LargeResultSet", func(t *testing.T) {
		userID := "test_memory_large"

		// Create large dataset
		for i := 0; i < 100; i++ {
			createTestTravel(t, testDB.DB, userID)
		}

		// Retrieve and process
		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify response size is reasonable
		bodySize := len(w.Body.Bytes())
		t.Logf("Response size for 100 travels: %d bytes", bodySize)

		if bodySize > 1000000 { // 1MB limit
			t.Errorf("Response too large: %d bytes", bodySize)
		}
	})

	t.Run("RepeatedRequests", func(t *testing.T) {
		userID := "test_memory_repeated"
		createTestTravel(t, testDB.DB, userID)

		// Make many repeated requests to test for memory leaks
		iterations := 100
		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			if w.Code != 200 {
				t.Errorf("Iteration %d failed", i)
			}

			// Allow GC to run
			if i%10 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		t.Logf("Completed %d repeated requests successfully", iterations)
	})
}

// TestScalability tests behavior under load
func TestScalability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("IncreasingLoad", func(t *testing.T) {
		userID := "test_scalability"

		loadLevels := []int{10, 50, 100}
		for _, level := range loadLevels {
			testUserID := userID + string(rune(level))

			// Create test data
			for i := 0; i < level; i++ {
				createTestTravel(t, testDB.DB, testUserID)
			}

			// Measure performance
			start := time.Now()

			req := httptest.NewRequest("GET", "/api/travels?user_id="+testUserID, nil)
			w := httptest.NewRecorder()

			travelsHandler(w, req)

			duration := time.Since(start)

			t.Logf("Load level %d: %v", level, duration)

			// Performance should degrade gracefully
			maxExpected := time.Duration(level) * 2 * time.Millisecond
			if duration > maxExpected {
				t.Logf("Performance degradation at load level %d: %v (expected < %v)", level, duration, maxExpected)
			}
		}
	})
}
