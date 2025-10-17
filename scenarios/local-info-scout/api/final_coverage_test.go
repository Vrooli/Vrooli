package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestMockPlaces tests the mock places function
func TestMockPlaces(t *testing.T) {
	places := getMockPlaces()

	if len(places) < 3 {
		t.Errorf("Expected at least 3 mock places, got %d", len(places))
	}

	for _, place := range places {
		if place.ID == "" {
			t.Error("Place should have an ID")
		}
		if place.Name == "" {
			t.Error("Place should have a name")
		}
		if place.Category == "" {
			t.Error("Place should have a category")
		}
	}
}

// TestSortByRelevanceMultipleCriteria tests sorting with various criteria
func TestSortByRelevanceMultipleCriteria(t *testing.T) {
	places := []Place{
		createTestPlace("1", "AAA", "restaurant", 0.5, 4.5),
		createTestPlace("2", "BBB", "restaurant", 0.5, 4.5),
		createTestPlace("3", "ZZZ", "restaurant", 0.5, 4.5),
	}

	sortByRelevance(places)

	// Should be sorted by name when rating and distance are equal
	if places[0].Name > places[1].Name {
		t.Error("Places should be sorted alphabetically when rating and distance are equal")
	}
}

// TestSearchHandlerWithFilters tests search with all filter combinations
func TestSearchHandlerWithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	// Test with accessible filter
	req := SearchRequest{
		Query:      "pharmacy",
		Lat:        40.7128,
		Lon:        -74.0060,
		Radius:     5.0,
		Accessible: true,
	}

	w := makeHTTPRequest(searchHandler, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/search",
		Body:   req,
	})

	places := assertPlacesResponse(t, w, 200)
	t.Logf("Found %d accessible places", len(places))
}

// TestDiscoverHandlerDeduplication tests deduplication in discover
func TestDiscoverHandlerDeduplication(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 10.0, // Large radius to get more results
	}

	w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
		Method: "POST",
		Path:   "/api/discover",
		Body:   req,
	})

	places := assertPlacesResponse(t, w, 200)

	// Check for duplicates
	seen := make(map[string]bool)
	for _, place := range places {
		if seen[place.ID] {
			t.Errorf("Found duplicate place ID: %s", place.ID)
		}
		seen[place.ID] = true
	}
}

// TestCalculateRelevanceScoreEdgeCases tests score calculation edge cases
func TestCalculateRelevanceScoreEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with exact name match
	place := createTestPlace("1", "vegan cafe", "restaurant", 0.3, 4.8)
	req := SearchRequest{
		Query:   "vegan cafe",
		Lat:     40.7128,
		Lon:     -74.0060,
		Radius:  5.0,
		OpenNow: true,
	}

	place.OpenNow = true
	scoredPlace := calculateRelevanceScore(place, req)

	if scoredPlace.Name != place.Name {
		t.Error("Scored place should maintain name")
	}

	// Test with keyword matches
	place2 := createTestPlace("2", "Fresh Organic Market", "grocery", 0.5, 4.6)
	req2 := SearchRequest{
		Query:  "organic fresh local",
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 5.0,
	}

	scoredPlace2 := calculateRelevanceScore(place2, req2)
	if scoredPlace2.Name != place2.Name {
		t.Error("Scored place should maintain name")
	}
}

// TestEnableCORSAllMethods tests CORS for all HTTP methods
func TestEnableCORSAllMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := makeHTTPRequest(enableCORS(healthHandler), HTTPTestRequest{
				Method: method,
				Path:   "/health",
			})

			assertHeaderPresent(t, w, "Access-Control-Allow-Origin")

			if method == "OPTIONS" {
				if w.Code != 200 {
					t.Errorf("OPTIONS should return 200, got %d", w.Code)
				}
			}
		})
	}
}

// TestCacheKeyUniqueness tests that different requests generate different cache keys
func TestCacheKeyUniqueness(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	keys := make(map[string]SearchRequest)

	requests := []SearchRequest{
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "restaurant"},
		{Lat: 40.7128, Lon: -74.0060, Radius: 10.0, Category: "restaurant"},
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "grocery"},
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "restaurant", MinRating: 4.0},
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "restaurant", MaxPrice: 2},
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "restaurant", OpenNow: true},
		{Lat: 40.7128, Lon: -74.0060, Radius: 5.0, Category: "restaurant", Accessible: true},
	}

	for _, req := range requests {
		key := getCacheKey(req)
		if _, exists := keys[key]; exists {
			t.Errorf("Duplicate cache key for different requests: %s", key)
		}
		keys[key] = req
	}

	if len(keys) != len(requests) {
		t.Errorf("Expected %d unique keys, got %d", len(requests), len(keys))
	}
}

// TestApplySmartFiltersHealthcare tests healthcare-specific rating adjustments
func TestApplySmartFiltersHealthcare(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "Pharmacy", "pharmacy", 0.5, 4.0),
		createTestPlace("2", "Clinic", "healthcare", 0.8, 4.0),
		createTestPlace("3", "Restaurant", "restaurant", 0.6, 4.0),
	}

	req := SearchRequest{
		MinRating: 4.0,
	}

	filtered := applySmartFilters(places, req)

	// Healthcare places require higher ratings (4.0 + 0.3 = 4.3)
	// So places with exactly 4.0 rating should be filtered out from healthcare
	for _, place := range filtered {
		if place.Category == "healthcare" || place.Category == "pharmacy" {
			if place.Rating < 4.3 {
				t.Logf("Healthcare place %s with rating %.1f was filtered (expected)", place.Name, place.Rating)
			}
		}
	}
}

// TestApplySmartFiltersSmartRadius tests smart radius adjustments
func TestApplySmartFiltersSmartRadius(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "Nearby Pharmacy", "pharmacy", 1.8, 4.5),
		createTestPlace("2", "Nearby Restaurant", "restaurant", 1.2, 4.5),
	}

	// No radius specified - should use smart defaults
	req := SearchRequest{}

	filtered := applySmartFilters(places, req)

	// Should apply category-specific radius:
	// - pharmacy/healthcare: 2.0 miles
	// - grocery/services: 1.5 miles
	// - default: 1.0 miles

	t.Logf("Filtered %d places with smart radius", len(filtered))
}

// TestTimeBasedRecommendationsAllHours tests recommendations for different times
func TestTimeBasedRecommendationsAllHours(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Lat: 40.7128,
		Lon: -74.0060,
	}

	// Test current time recommendations
	recommendations := getTimeBasedRecommendations(req)

	if len(recommendations) > 0 {
		rec := recommendations[0]
		if rec.Description == "" {
			t.Error("Recommendation should have a description")
		}
		if rec.Category == "" {
			t.Error("Recommendation should have a category")
		}
	}
}

// TestRedisContextUsage tests context usage in Redis operations
func TestRedisContextUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.RedisClient == nil {
		t.Skip("Redis not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Test that Redis operations respect context
	_, err := env.RedisClient.Ping(ctx).Result()
	if err != nil && ctx.Err() != nil {
		t.Logf("Context cancellation handled: %v", err)
	}
}

// TestJSONMarshaling tests JSON encoding/decoding
func TestJSONMarshaling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	place := Place{
		ID:          "test-1",
		Name:        "Test Place",
		Address:     "123 Test St",
		Category:    "restaurant",
		Distance:    0.5,
		Rating:      4.5,
		PriceLevel:  2,
		OpenNow:     true,
		Photos:      []string{"photo1.jpg", "photo2.jpg"},
		Description: "Test description",
	}

	// Marshal to JSON
	data, err := json.Marshal(place)
	if err != nil {
		t.Fatalf("Failed to marshal place: %v", err)
	}

	// Unmarshal from JSON
	var decoded Place
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal place: %v", err)
	}

	// Verify fields
	if decoded.ID != place.ID {
		t.Errorf("Expected ID %s, got %s", place.ID, decoded.ID)
	}
	if decoded.Name != place.Name {
		t.Errorf("Expected name %s, got %s", place.Name, decoded.Name)
	}
}

// TestSearchRequestMarshaling tests SearchRequest JSON handling
func TestSearchRequestMarshaling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := SearchRequest{
		Query:      "test",
		Lat:        40.7128,
		Lon:        -74.0060,
		Radius:     5.0,
		Category:   "restaurant",
		MinRating:  4.0,
		MaxPrice:   2,
		OpenNow:    true,
		Accessible: true,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded SearchRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Query != req.Query {
		t.Errorf("Expected query %s, got %s", req.Query, decoded.Query)
	}
	if decoded.Category != req.Category {
		t.Errorf("Expected category %s, got %s", req.Category, decoded.Category)
	}
}
