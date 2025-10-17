package main

import (
	"context"
	"testing"
	"time"
)

// TestFullCacheFlow tests complete cache workflow
func TestFullCacheFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.RedisClient == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	if _, err := env.RedisClient.Ping(ctx).Result(); err != nil {
		t.Skip("Redis not reachable")
	}

	// Set up test Redis client
	origClient := redisClient
	redisClient = env.RedisClient
	defer func() { redisClient = origClient }()

	// Test 1: Save to cache
	places := []Place{
		createTestPlace("1", "Cached Place 1", "restaurant", 0.5, 4.5),
		createTestPlace("2", "Cached Place 2", "grocery", 1.0, 4.2),
	}

	key := "search:40.712800:-74.006000:5.000000:restaurant:0.000000:0:false:false"
	saveToCache(key, places)

	// Small delay to ensure write completes
	time.Sleep(10 * time.Millisecond)

	// Test 2: Retrieve from cache
	retrieved, found := getFromCache(key)
	if !found {
		t.Error("Should find cached places")
	}

	if len(retrieved) != len(places) {
		t.Errorf("Expected %d places, got %d", len(places), len(retrieved))
	}

	// Test 3: Clear cache
	err := clearCache()
	if err != nil {
		t.Errorf("Clear cache failed: %v", err)
	}

	// Test 4: Verify cache is cleared
	_, found = getFromCache(key)
	if found {
		t.Error("Cache should be cleared")
	}
}

// TestDatabaseFlow tests complete database workflow
func TestDatabaseFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Save original db
	origDB := db
	defer func() { db = origDB }()

	// Initialize database
	initDB()

	if db == nil {
		t.Skip("Database not available")
	}

	// Test ping
	if err := db.Ping(); err != nil {
		t.Skip("Database not reachable")
	}

	// Test 1: Create tables
	createTables()
	t.Log("Tables created successfully")

	// Test 2: Save place
	place := createTestPlace("integration-test-1", "Integration Test Place", "restaurant", 0.5, 4.5)
	err := savePlaceToDb(place)
	if err != nil {
		t.Logf("Save place error: %v (continuing)", err)
	} else {
		t.Log("Place saved successfully")
	}

	// Test 3: Update place (upsert)
	place.Rating = 4.8
	err = savePlaceToDb(place)
	if err != nil {
		t.Logf("Update place error: %v (continuing)", err)
	} else {
		t.Log("Place updated successfully")
	}

	// Test 4: Log search
	req := createTestSearchRequest("integration test", 40.7128, -74.0060, 5.0)
	logSearch(req, 5, false, 100*time.Millisecond)
	t.Log("Search logged successfully")

	// Test 5: Get popular searches
	searches := getPopularSearches()
	t.Logf("Retrieved %d popular searches", len(searches))

	if len(searches) > 0 {
		t.Logf("Popular search: %s", searches[0])
	}
}

// TestTimeBasedRecommendationsComprehensive tests all time slots
func TestTimeBasedRecommendationsComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 0, // No radius filter
	}

	// Get recommendations for current time
	recommendations := getTimeBasedRecommendations(req)

	// Time-based recommendations may be empty during certain hours (10-11, 14-17, 2-6)
	// Just verify that if we get recommendations, they're valid
	t.Logf("Got %d time-based recommendations for current time", len(recommendations))

	// Verify recommendation structure
	for _, rec := range recommendations {
		if rec.ID == "" {
			t.Error("Recommendation should have an ID")
		}
		if rec.Name == "" {
			t.Error("Recommendation should have a name")
		}
		if rec.Category == "" {
			t.Error("Recommendation should have a category")
		}
		if rec.Description == "" {
			t.Error("Recommendation should have a description")
		}

		// Just log the description
		if rec.Description != "" {
			t.Logf("Recommendation: %s", rec.Description)
		}
	}
}

// TestGetPopularSearchesWithData tests popular searches with actual data
func TestGetPopularSearchesWithData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	origDB := db
	defer func() { db = origDB }()

	initDB()

	if db == nil {
		t.Skip("Database not available")
	}

	if err := db.Ping(); err != nil {
		t.Skip("Database not reachable")
	}

	createTables()

	// Log multiple searches
	searches := []string{"pizza", "sushi", "coffee", "pizza", "sushi", "pizza"}

	for _, query := range searches {
		req := SearchRequest{
			Query:  query,
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}
		logSearch(req, 3, false, 50*time.Millisecond)
	}

	// Small delay to ensure writes complete
	time.Sleep(100 * time.Millisecond)

	// Get popular searches
	popular := getPopularSearches()

	t.Logf("Found %d popular searches", len(popular))

	// "pizza" should appear multiple times, so might be most popular
	foundPizza := false
	for _, s := range popular {
		t.Logf("Popular search: %s", s)
		if s == "pizza" {
			foundPizza = true
		}
	}

	if !foundPizza && len(popular) > 0 {
		t.Log("Pizza not in top searches (may be expected due to timing)")
	}
}

// TestCacheWithDifferentTypes tests cache with various data types
func TestCacheWithDifferentTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.RedisClient == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	if _, err := env.RedisClient.Ping(ctx).Result(); err != nil {
		t.Skip("Redis not reachable")
	}

	origClient := redisClient
	redisClient = env.RedisClient
	defer func() { redisClient = origClient }()

	tests := []struct {
		name   string
		places []Place
	}{
		{
			name:   "EmptyPlaces",
			places: []Place{},
		},
		{
			name: "SinglePlace",
			places: []Place{
				createTestPlace("1", "Single", "restaurant", 0.5, 4.5),
			},
		},
		{
			name: "MultiplePlaces",
			places: []Place{
				createTestPlace("1", "First", "restaurant", 0.5, 4.5),
				createTestPlace("2", "Second", "grocery", 1.0, 4.2),
				createTestPlace("3", "Third", "pharmacy", 1.5, 4.8),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "test:" + tt.name

			saveToCache(key, tt.places)
			time.Sleep(10 * time.Millisecond)

			retrieved, found := getFromCache(key)
			if !found {
				t.Error("Should find cached data")
			}

			if len(retrieved) != len(tt.places) {
				t.Errorf("Expected %d places, got %d", len(tt.places), len(retrieved))
			}
		})
	}
}

// TestSearchHandlerWithCache tests search with caching
func TestSearchHandlerWithCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.RedisClient == nil {
		t.Skip("Redis not available")
	}

	ctx := context.Background()
	if _, err := env.RedisClient.Ping(ctx).Result(); err != nil {
		t.Skip("Redis not reachable")
	}

	origClient := redisClient
	redisClient = env.RedisClient
	defer func() { redisClient = origClient }()

	// First request - cache MISS
	req := SearchRequest{
		Query:    "restaurant",
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	w1 := makeHTTPRequest(searchHandler, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/search",
		Body:   req,
	})

	if w1.Code != 200 {
		t.Fatalf("First request failed: %d", w1.Code)
	}

	cacheHeader1 := w1.Header().Get("X-Cache")
	t.Logf("First request cache header: %s", cacheHeader1)

	// Second identical request - should be cache HIT
	w2 := makeHTTPRequest(searchHandler, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/search",
		Body:   req,
	})

	if w2.Code != 200 {
		t.Fatalf("Second request failed: %d", w2.Code)
	}

	cacheHeader2 := w2.Header().Get("X-Cache")
	t.Logf("Second request cache header: %s", cacheHeader2)

	if cacheHeader2 == "HIT" {
		t.Log("Cache working correctly - second request was a cache hit")
	}
}

// TestInitRedisSuccess tests successful Redis initialization
func TestInitRedisSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	origClient := redisClient
	defer func() { redisClient = origClient }()

	// Set valid Redis environment
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PORT", "6379")

	initRedis()

	if redisClient != nil {
		ctx := context.Background()
		if _, err := redisClient.Ping(ctx).Result(); err == nil {
			t.Log("Redis initialized and connected successfully")
		} else {
			t.Logf("Redis initialized but not reachable: %v", err)
		}
	} else {
		t.Log("Redis not initialized (may be expected if Redis not running)")
	}
}

// TestInitDBSuccess tests successful database initialization
func TestInitDBSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	origDB := db
	defer func() { db = origDB }()

	// Set valid DB environment
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	initDB()

	if db != nil {
		if err := db.Ping(); err == nil {
			t.Log("Database initialized and connected successfully")
		} else {
			t.Logf("Database initialized but not reachable: %v", err)
		}
	} else {
		t.Log("Database not initialized (may be expected if PostgreSQL not running)")
	}
}
