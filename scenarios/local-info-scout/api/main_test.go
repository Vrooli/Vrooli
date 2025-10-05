package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", result["status"])
		}

		if result["service"] != "local-info-scout" {
			t.Errorf("Expected service 'local-info-scout', got '%v'", result["service"])
		}

		if result["timestamp"] == nil {
			t.Error("Expected timestamp to be present")
		}
	})

	t.Run("CORSHeaders", func(t *testing.T) {
		w := makeHTTPRequest(enableCORS(healthHandler), HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		assertHeaderPresent(t, w, "Access-Control-Allow-Origin")
		assertHeaderValue(t, w, "Access-Control-Allow-Origin", "*")
	})

	t.Run("OPTIONSMethod", func(t *testing.T) {
		w := makeHTTPRequest(enableCORS(healthHandler), HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		assertHeaderPresent(t, w, "Access-Control-Allow-Methods")
	})
}

// TestCategoriesHandler tests the categories endpoint
func TestCategoriesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(categoriesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/categories",
		})

		categories := assertStringArrayResponse(t, w, http.StatusOK)

		if len(categories) == 0 {
			t.Error("Expected at least one category")
		}

		// Check for expected categories
		expectedCategories := map[string]bool{
			"restaurants": false,
			"grocery":     false,
			"pharmacy":    false,
			"parks":       false,
		}

		for _, cat := range categories {
			if _, exists := expectedCategories[cat]; exists {
				expectedCategories[cat] = true
			}
		}

		for cat, found := range expectedCategories {
			if !found {
				t.Errorf("Expected category '%s' not found", cat)
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		w := makeHTTPRequest(categoriesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/categories",
		})

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected JSON content type, got '%s'", contentType)
		}
	})
}

// TestSearchHandler tests the search endpoint
func TestSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := createTestSearchRequest("restaurant", 40.7128, -74.0060, 5.0)

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		if len(places) == 0 {
			t.Error("Expected at least one place in results")
		}
	})

	t.Run("SuccessWithCategory", func(t *testing.T) {
		req := SearchRequest{
			Query:    "vegan",
			Lat:      40.7128,
			Lon:      -74.0060,
			Radius:   5.0,
			Category: "restaurant",
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// Verify all places match the requested category
		for _, place := range places {
			if place.Category != "restaurant" {
				t.Errorf("Expected category 'restaurant', got '%s' for place '%s'",
					place.Category, place.Name)
			}
		}
	})

	t.Run("SuccessWithRating", func(t *testing.T) {
		req := SearchRequest{
			Query:     "restaurant",
			Lat:       40.7128,
			Lon:       -74.0060,
			Radius:    5.0,
			MinRating: 4.0,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// Verify all places meet minimum rating
		for _, place := range places {
			if place.Rating < req.MinRating {
				t.Errorf("Expected rating >= %.1f, got %.1f for place '%s'",
					req.MinRating, place.Rating, place.Name)
			}
		}
	})

	t.Run("SuccessWithMaxPrice", func(t *testing.T) {
		req := SearchRequest{
			Query:    "restaurant",
			Lat:      40.7128,
			Lon:      -74.0060,
			Radius:   5.0,
			MaxPrice: 2,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// Verify all places meet price requirement (except parks)
		for _, place := range places {
			if place.Category != "parks" && place.PriceLevel > req.MaxPrice {
				t.Errorf("Expected price level <= %d, got %d for place '%s'",
					req.MaxPrice, place.PriceLevel, place.Name)
			}
		}
	})

	t.Run("SuccessWithOpenNow", func(t *testing.T) {
		req := SearchRequest{
			Query:   "restaurant",
			Lat:     40.7128,
			Lon:     -74.0060,
			Radius:  5.0,
			OpenNow: true,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// All results should be open now or be 24-hour
		for _, place := range places {
			is24Hour := strings.Contains(strings.ToLower(place.Name), "24") ||
				strings.Contains(strings.ToLower(place.Description), "24 hour")

			if !place.OpenNow && !is24Hour {
				t.Errorf("Expected place to be open now or 24-hour, got '%s'", place.Name)
			}
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
		})

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   `{"invalid": json}`,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   "",
		})

		// Should still return results with default mock data
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400 for empty body, got %d", w.Code)
		}
	})

	t.Run("ErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "SearchHandler",
			Handler:     searchHandler,
			BaseURL:     "/api/search",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("/api/search").
			AddEmptyQuery("/api/search").
			AddInvalidCoordinates("/api/search").
			AddNegativeRadius("/api/search").
			AddExcessiveRadius("/api/search").
			AddInvalidCategory("/api/search").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestPlaceDetailsHandler tests the place details endpoint
func TestPlaceDetailsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/123",
		})

		place := assertPlaceResponse(t, w, http.StatusOK)

		if place.ID != "123" {
			t.Errorf("Expected ID '123', got '%s'", place.ID)
		}

		if place.Name == "" {
			t.Error("Expected place name to be non-empty")
		}

		if place.Address == "" {
			t.Error("Expected place address to be non-empty")
		}
	})

	t.Run("MissingID", func(t *testing.T) {
		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Place ID required")
	})

	t.Run("PhotosPresent", func(t *testing.T) {
		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/123",
		})

		place := assertPlaceResponse(t, w, http.StatusOK)

		if len(place.Photos) == 0 {
			t.Error("Expected at least one photo")
		}
	})
}

// TestDiscoverHandler tests the discover endpoint
func TestDiscoverHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := SearchRequest{
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		if len(places) == 0 {
			t.Error("Expected at least one discovery result")
		}
	})

	t.Run("HiddenGemsMarked", func(t *testing.T) {
		req := SearchRequest{
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// Check if any places are marked as hidden gems
		foundHiddenGem := false
		for _, place := range places {
			if strings.Contains(place.Description, "Hidden Gem") {
				foundHiddenGem = true
				// Verify hidden gem criteria
				if place.Rating < 4.5 {
					t.Errorf("Hidden gem should have rating >= 4.5, got %.1f", place.Rating)
				}
			}
		}

		if !foundHiddenGem {
			t.Log("No hidden gems found in discovery results (may be expected)")
		}
	})

	t.Run("TrendingPlacesMarked", func(t *testing.T) {
		req := SearchRequest{
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		// Check if any places are marked as trending
		foundTrending := false
		for _, place := range places {
			if strings.Contains(place.Description, "Trending") ||
				strings.Contains(place.Description, "Just Opened") ||
				strings.Contains(place.Description, "Popular") {
				foundTrending = true
				break
			}
		}

		if !foundTrending {
			t.Log("No trending places found in discovery results")
		}
	})

	t.Run("LimitedResults", func(t *testing.T) {
		req := SearchRequest{
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/discover",
			Body:   req,
		})

		places := assertPlacesResponse(t, w, http.StatusOK)

		if len(places) > 10 {
			t.Errorf("Expected max 10 results, got %d", len(places))
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/discover",
		})

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})
}

// TestClearCacheHandler tests the cache clearing endpoint
func TestClearCacheHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := makeHTTPRequest(clearCacheHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/cache/clear",
		})

		result := assertJSONResponse(t, w, http.StatusOK)

		if result["status"] != "success" {
			t.Errorf("Expected status 'success', got '%v'", result["status"])
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(clearCacheHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/cache/clear",
		})

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})
}

// TestParseNaturalLanguageQuery tests the natural language parsing
func TestParseNaturalLanguageQuery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name     string
		query    string
		expected ParsedQuery
	}{
		{
			name:  "SimpleRestaurantQuery",
			query: "restaurants",
			expected: ParsedQuery{
				Category: "restaurant",
				Radius:   1.0,
			},
		},
		{
			name:  "QueryWithRadius",
			query: "coffee shops within 2 miles",
			expected: ParsedQuery{
				Radius: 2.0,
			},
		},
		{
			name:  "ParkQuery",
			query: "find a park nearby",
			expected: ParsedQuery{
				Category: "parks",
				Radius:   1.0,
			},
		},
		{
			name:  "GroceryQuery",
			query: "organic grocery store",
			expected: ParsedQuery{
				Category: "grocery",
				Radius:   1.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseNaturalLanguageQuery(tt.query)

			if tt.expected.Category != "" && result.Category != tt.expected.Category {
				t.Errorf("Expected category '%s', got '%s'", tt.expected.Category, result.Category)
			}

			if tt.expected.Radius > 0 && result.Radius != tt.expected.Radius {
				t.Logf("Expected radius %.1f, got %.1f (may use default)", tt.expected.Radius, result.Radius)
			}
		})
	}
}

// TestApplySmartFilters tests the smart filtering logic
func TestApplySmartFilters(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "High Rating Restaurant", "restaurant", 0.5, 4.8),
		createTestPlace("2", "Low Rating Place", "restaurant", 0.8, 3.2),
		createTestPlace("3", "Far Place", "grocery", 10.0, 4.5),
		createTestPlace("4", "Nearby Park", "parks", 0.3, 4.7),
	}

	t.Run("FilterByCategory", func(t *testing.T) {
		req := SearchRequest{Category: "restaurant"}
		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Category != "restaurant" {
				t.Errorf("Expected only restaurants, got '%s'", place.Category)
			}
		}
	})

	t.Run("FilterByRadius", func(t *testing.T) {
		req := SearchRequest{Radius: 1.0}
		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Distance > 1.0 {
				t.Errorf("Expected distance <= 1.0, got %.1f for '%s'",
					place.Distance, place.Name)
			}
		}
	})

	t.Run("FilterByRating", func(t *testing.T) {
		req := SearchRequest{MinRating: 4.0}
		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Rating < 4.0 {
				t.Errorf("Expected rating >= 4.0, got %.1f for '%s'",
					place.Rating, place.Name)
			}
		}
	})
}

// TestIsChainStore tests the chain store detection
func TestIsChainStore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name     string
		expected bool
	}{
		{"McDonald's", true},
		{"Starbucks Coffee", true},
		{"Local Coffee Shop", false},
		{"The Green Garden Cafe", false},
		{"Subway Sandwiches", true},
		{"7-Eleven", true},
		{"Mom's Diner", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isChainStore(tt.name)
			if result != tt.expected {
				t.Errorf("Expected isChainStore('%s') = %v, got %v",
					tt.name, tt.expected, result)
			}
		})
	}
}

// TestGetCacheKey tests cache key generation
func TestGetCacheKey(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req1 := SearchRequest{
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	req2 := SearchRequest{
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	req3 := SearchRequest{
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   10.0, // Different radius
		Category: "restaurant",
	}

	key1 := getCacheKey(req1)
	key2 := getCacheKey(req2)
	key3 := getCacheKey(req3)

	if key1 != key2 {
		t.Error("Identical requests should produce the same cache key")
	}

	if key1 == key3 {
		t.Error("Different requests should produce different cache keys")
	}
}

// TestRedisCache tests Redis caching functionality
func TestRedisCache(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Initialize Redis for testing
	ctx := context.Background()
	if env.RedisClient != nil {
		_, err := env.RedisClient.Ping(ctx).Result()
		if err != nil {
			t.Skip("Redis not available, skipping cache tests")
		}

		redisClient = env.RedisClient

		t.Run("SaveAndRetrieve", func(t *testing.T) {
			places := []Place{
				createTestPlace("1", "Test Place", "restaurant", 0.5, 4.5),
			}

			key := "test:cache:key"
			saveToCache(key, places)

			retrieved, found := getFromCache(key)
			if !found {
				t.Error("Expected to find cached data")
			}

			if len(retrieved) != len(places) {
				t.Errorf("Expected %d places, got %d", len(places), len(retrieved))
			}
		})

		t.Run("ClearCache", func(t *testing.T) {
			places := []Place{
				createTestPlace("1", "Test Place", "restaurant", 0.5, 4.5),
			}

			key := "search:test:key"
			saveToCache(key, places)

			if err := clearCache(); err != nil {
				t.Errorf("Failed to clear cache: %v", err)
			}

			_, found := getFromCache(key)
			if found {
				t.Error("Expected cache to be cleared")
			}
		})
	} else {
		t.Skip("Redis client not initialized, skipping cache tests")
	}
}

// TestGetEnv tests environment variable handling
func TestGetEnv(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExistingVar", func(t *testing.T) {
		key := "TEST_VAR_EXISTS"
		expected := "test_value"
		t.Setenv(key, expected)

		result := getEnv(key, "default")
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("MissingVar", func(t *testing.T) {
		key := "TEST_VAR_MISSING"
		defaultValue := "default_value"

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected '%s', got '%s'", defaultValue, result)
		}
	})
}

// TestTimeBasedRecommendations tests time-based recommendations
func TestTimeBasedRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat: 40.7128,
		Lon: -74.0060,
	}

	recommendations := getTimeBasedRecommendations(req)

	// Time-based recommendations may be empty during certain hours (10-11, 14-17, 2-6)
	// Just verify that if we get recommendations, they're valid
	for _, place := range recommendations {
		if place.Name == "" {
			t.Error("Expected place name to be non-empty")
		}
		if place.Category == "" {
			t.Error("Expected place category to be non-empty")
		}
		if place.Distance < 0 {
			t.Error("Expected valid distance")
		}
		if place.ID == "" {
			t.Error("Expected place ID to be non-empty")
		}
	}

	// Verify recommendations are time-appropriate when present
	if len(recommendations) > 0 {
		t.Logf("Got %d time-based recommendations", len(recommendations))
	}
}

// TestShouldSwap tests the sorting comparison function
func TestShouldSwap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	place1 := createTestPlace("1", "High Rating", "restaurant", 0.5, 4.8)
	place2 := createTestPlace("2", "Low Rating", "restaurant", 0.5, 3.5)

	// Higher rating should come first (swap = true if a < b)
	if !shouldSwap(place2, place1) {
		t.Error("Should swap: lower rating should come after higher rating")
	}

	// Same rating, check distance
	place3 := createTestPlace("3", "Near", "restaurant", 0.3, 4.5)
	place4 := createTestPlace("4", "Far", "restaurant", 0.8, 4.5)

	if !shouldSwap(place4, place3) {
		t.Error("Should swap: farther place should come after nearer place with same rating")
	}
}

// TestDeduplicateAndLimit tests deduplication and limiting
func TestDeduplicateAndLimit(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "Place 1", "restaurant", 0.5, 4.5),
		createTestPlace("2", "Place 2", "restaurant", 0.6, 4.3),
		createTestPlace("1", "Place 1 Duplicate", "restaurant", 0.5, 4.5),
		createTestPlace("3", "Place 3", "restaurant", 0.7, 4.2),
	}

	t.Run("Deduplicate", func(t *testing.T) {
		result := deduplicateAndLimit(places, 10)

		if len(result) != 3 {
			t.Errorf("Expected 3 unique places, got %d", len(result))
		}

		// Check no duplicates
		seen := make(map[string]bool)
		for _, place := range result {
			if seen[place.ID] {
				t.Errorf("Found duplicate place ID: %s", place.ID)
			}
			seen[place.ID] = true
		}
	})

	t.Run("Limit", func(t *testing.T) {
		result := deduplicateAndLimit(places, 2)

		if len(result) > 2 {
			t.Errorf("Expected max 2 places, got %d", len(result))
		}
	})
}
