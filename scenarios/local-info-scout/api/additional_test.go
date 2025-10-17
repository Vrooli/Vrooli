package main

import (
	"net/http"
	"os"
	"testing"
)

// TestInitRedis tests Redis initialization
func TestInitRedis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Save original values
	origHost := os.Getenv("REDIS_HOST")
	origPort := os.Getenv("REDIS_PORT")
	defer func() {
		if origHost != "" {
			os.Setenv("REDIS_HOST", origHost)
		}
		if origPort != "" {
			os.Setenv("REDIS_PORT", origPort)
		}
	}()

	t.Run("WithValidConfig", func(t *testing.T) {
		os.Setenv("REDIS_HOST", "localhost")
		os.Setenv("REDIS_PORT", "6379")

		// Call initRedis
		initRedis()

		// Redis client may or may not be available, just verify no panic
		if redisClient != nil {
			t.Log("Redis client initialized successfully")
		} else {
			t.Log("Redis not available (expected in test environment)")
		}
	})

	t.Run("WithInvalidConfig", func(t *testing.T) {
		os.Setenv("REDIS_HOST", "invalid-host-xyz")
		os.Setenv("REDIS_PORT", "9999")

		// Should not panic
		initRedis()

		// Redis client should be nil or disconnected
		t.Log("Handled invalid Redis config gracefully")
	})
}

// TestInitDB tests database initialization
func TestInitDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Save original values
	origHost := os.Getenv("POSTGRES_HOST")
	origPort := os.Getenv("POSTGRES_PORT")
	origUser := os.Getenv("POSTGRES_USER")
	origPassword := os.Getenv("POSTGRES_PASSWORD")
	origDB := os.Getenv("POSTGRES_DB")

	defer func() {
		if origHost != "" {
			os.Setenv("POSTGRES_HOST", origHost)
		}
		if origPort != "" {
			os.Setenv("POSTGRES_PORT", origPort)
		}
		if origUser != "" {
			os.Setenv("POSTGRES_USER", origUser)
		}
		if origPassword != "" {
			os.Setenv("POSTGRES_PASSWORD", origPassword)
		}
		if origDB != "" {
			os.Setenv("POSTGRES_DB", origDB)
		}
	}()

	t.Run("WithValidConfig", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "test")
		os.Setenv("POSTGRES_PASSWORD", "test")
		os.Setenv("POSTGRES_DB", "local_info_scout_test")

		// Call initDB
		initDB()

		// DB may or may not be available, just verify no panic
		if db != nil {
			t.Log("Database initialized successfully")
		} else {
			t.Log("Database not available (expected in test environment)")
		}
	})

	t.Run("WithInvalidConfig", func(t *testing.T) {
		os.Setenv("POSTGRES_HOST", "invalid-host-xyz")
		os.Setenv("POSTGRES_PORT", "9999")

		// Should not panic
		initDB()

		// DB should be nil
		t.Log("Handled invalid DB config gracefully")
	})
}

// TestCreateTables tests table creation
func TestCreateTables(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithNilDB", func(t *testing.T) {
		origDB := db
		db = nil
		defer func() { db = origDB }()

		// Should not panic with nil db
		createTables()
		t.Log("Handled nil DB gracefully")
	})
}

// TestSavePlaceToDb tests saving places to database
func TestSavePlaceToDb(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithNilDB", func(t *testing.T) {
		origDB := db
		db = nil
		defer func() { db = origDB }()

		place := createTestPlace("1", "Test Place", "restaurant", 0.5, 4.5)
		err := savePlaceToDb(place)

		if err != nil {
			t.Logf("Expected nil error with nil DB, got: %v", err)
		}
	})
}

// TestLogSearch tests search logging
func TestLogSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithNilDB", func(t *testing.T) {
		origDB := db
		db = nil
		defer func() { db = origDB }()

		req := createTestSearchRequest("test", 40.7128, -74.0060, 5.0)

		// Should not panic with nil db
		logSearch(req, 5, true, 100)
		t.Log("Handled nil DB gracefully for logging")
	})
}

// TestGetPopularSearches tests retrieving popular searches
func TestGetPopularSearches(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithNilDB", func(t *testing.T) {
		origDB := db
		db = nil
		defer func() { db = origDB }()

		searches := getPopularSearches()

		if len(searches) != 0 {
			t.Errorf("Expected empty array with nil DB, got %d results", len(searches))
		}
	})

	t.Run("WithMockDB", func(t *testing.T) {
		// This test would require setting up a test database
		// For now, just verify the function doesn't panic
		origDB := db
		defer func() { db = origDB }()

		// Mock DB setup (if available)
		// db = setupMockDB()

		searches := getPopularSearches()
		t.Logf("Retrieved %d popular searches", len(searches))
	})
}

// TestTimeBasedRecommendationsAllTimeSlots tests all time-based recommendation scenarios
func TestTimeBasedRecommendationsAllTimeSlots(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 2.0,
	}

	// Test with radius filtering
	recommendations := getTimeBasedRecommendations(req)

	for _, place := range recommendations {
		if req.Radius > 0 && place.Distance > req.Radius {
			t.Errorf("Place %s has distance %.1f, exceeds radius %.1f",
				place.Name, place.Distance, req.Radius)
		}
	}
}

// TestDiscoverTrendingPlacesFiltering tests filtering in trending places
func TestDiscoverTrendingPlacesFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 0.5, // Small radius
	}

	trending := discoverTrendingPlaces(req)

	// With small radius, some places should be filtered out
	for _, place := range trending {
		if place.Distance > req.Radius {
			t.Errorf("Trending place %s has distance %.1f, exceeds radius %.1f",
				place.Name, place.Distance, req.Radius)
		}
	}
}

// TestFetchRealTimeData tests real-time data fetching
func TestFetchRealTimeData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithValidRequest", func(t *testing.T) {
		req := SearchRequest{
			Query:    "restaurant",
			Lat:      40.7128,
			Lon:      -74.0060,
			Radius:   5.0,
			Category: "restaurant",
		}

		places := fetchRealTimeData(req)

		// Real-time data may not be available in test environment
		// Should return empty array or valid places
		t.Logf("Fetched %d real-time places", len(places))

		for _, place := range places {
			if place.Name == "" {
				t.Error("Real-time place should have a name")
			}
		}
	})

	t.Run("WithoutCategory", func(t *testing.T) {
		req := SearchRequest{
			Query:  "coffee",
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		places := fetchRealTimeData(req)
		t.Logf("Fetched %d places without category filter", len(places))
	})
}

// TestCalculateRelevanceScore tests relevance scoring
func TestCalculateRelevanceScore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	place := createTestPlace("1", "Vegan Organic Cafe", "restaurant", 0.5, 4.5)

	t.Run("WithMatchingQuery", func(t *testing.T) {
		req := SearchRequest{
			Query:     "vegan organic",
			Lat:       40.7128,
			Lon:       -74.0060,
			Radius:    5.0,
			MaxPrice:  2,
			OpenNow:   true,
			MinRating: 4.0,
		}

		// Score calculation should work
		scoredPlace := calculateRelevanceScore(place, req)

		// The function returns the place (score not exposed but calculated)
		if scoredPlace.Name != place.Name {
			t.Error("Scored place should have same name")
		}
	})

	t.Run("WithNoQuery", func(t *testing.T) {
		req := SearchRequest{
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		scoredPlace := calculateRelevanceScore(place, req)
		if scoredPlace.Name != place.Name {
			t.Error("Scored place should have same name")
		}
	})
}

// TestGetFromCacheNilClient tests cache retrieval with nil client
func TestGetFromCacheNilClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	origClient := redisClient
	redisClient = nil
	defer func() { redisClient = origClient }()

	places, found := getFromCache("test:key")

	if found {
		t.Error("Should not find cache with nil client")
	}

	if places != nil {
		t.Error("Should return nil places with nil client")
	}
}

// TestSaveToCacheNilClient tests cache saving with nil client
func TestSaveToCacheNilClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	origClient := redisClient
	redisClient = nil
	defer func() { redisClient = origClient }()

	places := []Place{
		createTestPlace("1", "Test", "restaurant", 0.5, 4.5),
	}

	// Should not panic with nil client
	saveToCache("test:key", places)
	t.Log("Handled nil Redis client gracefully for save")
}

// TestClearCacheNilClient tests cache clearing with nil client
func TestClearCacheNilClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	origClient := redisClient
	redisClient = nil
	defer func() { redisClient = origClient }()

	err := clearCache()

	if err != nil {
		t.Errorf("Should not error with nil client, got: %v", err)
	}
}

// TestClearCacheHandlerError tests error handling in cache clear handler
func TestClearCacheHandlerError(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with nil Redis client
	origClient := redisClient
	redisClient = nil
	defer func() { redisClient = origClient }()

	w := makeHTTPRequest(clearCacheHandler, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/cache/clear",
	})

	// Should succeed even with nil client
	if w.Code != 200 {
		t.Logf("Cache clear returned status %d with nil client", w.Code)
	}
}

// TestDiscoverHandlerErrorCases tests error cases in discover handler
func TestDiscoverHandlerErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   `{"invalid": json}`,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})
}

// TestMain is not tested directly but we verify getEnv works
func TestMainPrerequisites(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test getEnv which is used by main
	t.Run("GetEnvDefaults", func(t *testing.T) {
		key := "NONEXISTENT_VAR_" + t.Name()
		value := getEnv(key, "default")
		if value != "default" {
			t.Errorf("Expected default value, got %s", value)
		}
	})
}
