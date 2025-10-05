package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "travel-map-filler",
		})

		if response == nil {
			t.Fatal("Expected response to be non-nil")
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		// Verify CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS header to allow all origins")
		}
	})
}

// TestEnableCORS tests CORS header setting
func TestEnableCORS(t *testing.T) {
	w := httptest.NewRecorder()
	enableCORS(w)

	headers := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	}

	for _, header := range headers {
		if w.Header().Get(header) == "" {
			t.Errorf("Expected CORS header '%s' to be set", header)
		}
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin to be '*', got '%s'",
			w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// TestTravelsHandler tests the travels listing endpoint
func TestTravelsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_travels"

	// Create test data
	createTestTravel(t, testDB.DB, userID)
	createTestTravel(t, testDB.DB, userID)

	t.Run("Success_WithUserID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if travels == nil {
			t.Fatal("Expected travels array to be non-nil")
		}

		if len(travels) < 2 {
			t.Errorf("Expected at least 2 travels, got %d", len(travels))
		}
	})

	t.Run("Success_DefaultUser", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels", nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if travels == nil {
			t.Fatal("Expected travels array to be non-nil")
		}
	})

	t.Run("Success_WithYearFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID+"&year=2024", nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if travels == nil {
			t.Fatal("Expected travels array to be non-nil")
		}
	})

	t.Run("Success_WithTypeFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID+"&type=vacation", nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if travels == nil {
			t.Fatal("Expected travels array to be non-nil")
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/travels", nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("EmptyResult", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels?user_id=nonexistent_user", nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		travels := assertJSONArray(t, w, http.StatusOK)
		if travels == nil {
			t.Fatal("Expected empty travels array to be non-nil")
		}

		if len(travels) != 0 {
			t.Errorf("Expected 0 travels for nonexistent user, got %d", len(travels))
		}
	})
}

// TestStatsHandler tests the statistics endpoint
func TestStatsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_stats"

	// Create test data
	createTestTravel(t, testDB.DB, userID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
		w := httptest.NewRecorder()

		statsHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response to be non-nil")
		}

		// Verify stats fields exist
		fields := []string{"total_countries", "total_cities", "total_continents"}
		for _, field := range fields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in stats response", field)
			}
		}
	})

	t.Run("Success_NoStats", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/stats?user_id=nonexistent_user", nil)
		w := httptest.NewRecorder()

		statsHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response to be non-nil")
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/stats", nil)
		w := httptest.NewRecorder()

		statsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestAchievementsHandler tests the achievements endpoint
func TestAchievementsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_achievements"

	// Create test achievement
	createTestAchievement(t, testDB.DB, userID, "first_trip")

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/achievements?user_id="+userID, nil)
		w := httptest.NewRecorder()

		achievementsHandler(w, req)

		achievements := assertJSONArray(t, w, http.StatusOK)
		if achievements == nil {
			t.Fatal("Expected achievements array to be non-nil")
		}

		if len(achievements) < 1 {
			t.Errorf("Expected at least 1 achievement, got %d", len(achievements))
		}
	})

	t.Run("Success_EmptyResult", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/achievements?user_id=nonexistent_user", nil)
		w := httptest.NewRecorder()

		achievementsHandler(w, req)

		achievements := assertJSONArray(t, w, http.StatusOK)
		if achievements == nil {
			t.Fatal("Expected empty achievements array to be non-nil")
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/achievements", nil)
		w := httptest.NewRecorder()

		achievementsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestBucketListHandler tests the bucket list endpoint
func TestBucketListHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_bucket"

	// Create test bucket item
	createTestBucketItem(t, testDB.DB, userID)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bucket-list?user_id="+userID, nil)
		w := httptest.NewRecorder()

		bucketListHandler(w, req)

		bucketList := assertJSONArray(t, w, http.StatusOK)
		if bucketList == nil {
			t.Fatal("Expected bucket list array to be non-nil")
		}

		if len(bucketList) < 1 {
			t.Errorf("Expected at least 1 bucket item, got %d", len(bucketList))
		}
	})

	t.Run("Success_EmptyResult", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/bucket-list?user_id=nonexistent_user", nil)
		w := httptest.NewRecorder()

		bucketListHandler(w, req)

		bucketList := assertJSONArray(t, w, http.StatusOK)
		if bucketList == nil {
			t.Fatal("Expected empty bucket list array to be non-nil")
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/bucket-list", nil)
		w := httptest.NewRecorder()

		bucketListHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestAddTravelHandler tests the add travel endpoint
func TestAddTravelHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("Success", func(t *testing.T) {
		travelData := TestData.TravelRequest("test_user_add", "London, UK")
		travelJSON, _ := json.Marshal(travelData)

		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "success",
		})

		if response == nil {
			t.Fatal("Expected response to be non-nil")
		}

		// Verify travel ID was returned
		if _, exists := response["id"]; !exists {
			t.Error("Expected 'id' field in response")
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBufferString("{invalid}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/add-travel", nil)
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for GET method, got %d", w.Code)
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/add-travel", nil)
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestSearchTravelsHandler tests the search endpoint
func TestSearchTravelsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_GetWithQuery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels/search?q=Paris", nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		// Should return results or empty array even if n8n unavailable
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Success_PostWithBody", func(t *testing.T) {
		searchData := map[string]interface{}{
			"query":   "beach vacation",
			"limit":   5,
			"user_id": "test_user",
		}
		searchJSON, _ := json.Marshal(searchData)

		req := httptest.NewRequest("POST", "/api/travels/search", bytes.NewBuffer(searchJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Error_MissingQuery", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/travels/search", nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for missing query, got %d", w.Code)
		}
	})

	t.Run("OPTIONS", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/travels/search", nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestCheckAchievements tests achievement unlock logic
func TestCheckAchievements(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_check_achievements"

	t.Run("FirstTripAchievement", func(t *testing.T) {
		// Create first travel
		createTestTravel(t, testDB.DB, userID)

		// Check achievements
		checkAchievements(userID)

		// Verify achievement was created
		var count int
		err := testDB.DB.QueryRow("SELECT COUNT(*) FROM achievements WHERE user_id = $1 AND achievement_type = 'first_trip'", userID).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query achievements: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected 1 first_trip achievement, got %d", count)
		}
	})
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Error_MissingConfig", func(t *testing.T) {
		// Save original env vars
		origHost := os.Getenv("POSTGRES_HOST")
		origPort := os.Getenv("POSTGRES_PORT")
		origUser := os.Getenv("POSTGRES_USER")
		origPass := os.Getenv("POSTGRES_PASSWORD")
		origDB := os.Getenv("POSTGRES_DB")

		// Clear all env vars
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		// Test initialization fails
		err := initDB()
		if err == nil {
			t.Error("Expected error for missing database configuration")
		}

		if !strings.Contains(err.Error(), "Missing required database configuration") {
			t.Errorf("Expected error about missing config, got: %v", err)
		}

		// Restore env vars
		os.Setenv("POSTGRES_HOST", origHost)
		os.Setenv("POSTGRES_PORT", origPort)
		os.Setenv("POSTGRES_USER", origUser)
		os.Setenv("POSTGRES_PASSWORD", origPass)
		os.Setenv("POSTGRES_DB", origDB)
	})
}

// TestPerformance tests handler performance
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_performance"

	// Create test data
	for i := 0; i < 10; i++ {
		createTestTravel(t, testDB.DB, userID)
	}

	t.Run("TravelsHandler_Performance", func(t *testing.T) {
		start := time.Now()

		req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
		w := httptest.NewRecorder()

		travelsHandler(w, req)

		duration := time.Since(start)

		if duration > 200*time.Millisecond {
			t.Errorf("TravelsHandler too slow: %v (expected < 200ms)", duration)
		} else {
			t.Logf("TravelsHandler completed in %v", duration)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("StatsHandler_Performance", func(t *testing.T) {
		start := time.Now()

		req := httptest.NewRequest("GET", "/api/stats?user_id="+userID, nil)
		w := httptest.NewRecorder()

		statsHandler(w, req)

		duration := time.Since(start)

		if duration > 200*time.Millisecond {
			t.Errorf("StatsHandler too slow: %v (expected < 200ms)", duration)
		} else {
			t.Logf("StatsHandler completed in %v", duration)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestConcurrency tests concurrent request handling
func TestConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	userID := "test_user_concurrency"
	createTestTravel(t, testDB.DB, userID)

	t.Run("ConcurrentReads", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(iteration int) {
				req := httptest.NewRequest("GET", "/api/travels?user_id="+userID, nil)
				w := httptest.NewRecorder()

				travelsHandler(w, req)

				if w.Code != http.StatusOK {
					errors <- fmt.Errorf("iteration %d: expected status 200, got %d", iteration, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < concurrency; i++ {
			<-done
		}

		close(errors)
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Concurrency test failed with %d errors", errorCount)
		}
	})
}

// TestDataStructures tests data structure handling
func TestDataStructures(t *testing.T) {
	t.Run("Travel_JSONSerialization", func(t *testing.T) {
		travel := BuildTestTravel("test_user")

		// Marshal to JSON
		jsonData, err := json.Marshal(travel)
		if err != nil {
			t.Fatalf("Failed to marshal travel: %v", err)
		}

		// Unmarshal back
		var unmarshaled Travel
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal travel: %v", err)
		}

		// Verify data integrity
		if unmarshaled.UserID != travel.UserID {
			t.Errorf("Expected UserID %s, got %s", travel.UserID, unmarshaled.UserID)
		}
		if unmarshaled.Location != travel.Location {
			t.Errorf("Expected Location %s, got %s", travel.Location, unmarshaled.Location)
		}
	})

	t.Run("Stats_JSONSerialization", func(t *testing.T) {
		stats := BuildTestStats()

		jsonData, err := json.Marshal(stats)
		if err != nil {
			t.Fatalf("Failed to marshal stats: %v", err)
		}

		var unmarshaled Stats
		if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal stats: %v", err)
		}

		if unmarshaled.TotalCountries != stats.TotalCountries {
			t.Errorf("Expected TotalCountries %d, got %d", stats.TotalCountries, unmarshaled.TotalCountries)
		}
	})
}

// TestEdgeCases tests edge case scenarios
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	defer testDB.Cleanup()

	t.Run("AddTravel_MinimalData", func(t *testing.T) {
		minimalTravel := map[string]interface{}{
			"location": "Test",
		}
		travelJSON, _ := json.Marshal(minimalTravel)

		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		// Should handle minimal data gracefully
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for minimal data, got %d", w.Code)
		}
	})

	t.Run("AddTravel_EmptyArrays", func(t *testing.T) {
		travelData := TestData.TravelRequest("test_user_empty", "Test Location")
		travelData["photos"] = []string{}
		travelData["tags"] = []string{}
		travelJSON, _ := json.Marshal(travelData)

		req := httptest.NewRequest("POST", "/api/add-travel", bytes.NewBuffer(travelJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		addTravelHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty arrays, got %d", w.Code)
		}
	})

	t.Run("Search_SpecialCharacters", func(t *testing.T) {
		specialChars := "Paris & France! @#$%"
		req := httptest.NewRequest("GET", "/api/travels/search?q="+specialChars, nil)
		w := httptest.NewRecorder()

		searchTravelsHandler(w, req)

		// Should handle special characters without crashing
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status for special characters: %d", w.Code)
		}
	})
}
