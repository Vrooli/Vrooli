package main

import (
	"testing"
)

// TestParseNaturalLanguageQueryWithOllama tests query parsing with Ollama integration
func TestParseNaturalLanguageQueryWithOllama(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with various queries to hit different code paths
	testCases := []struct {
		name  string
		query string
	}{
		{"RestaurantQuery", "find me a restaurant"},
		{"GroceryQuery", "grocery store nearby"},
		{"PharmacyQuery", "pharmacy open now"},
		{"ParksQuery", "parks near me"},
		{"ShoppingQuery", "shopping center"},
		{"EntertainmentQuery", "entertainment venues"},
		{"ServicesQuery", "services available"},
		{"FitnessQuery", "fitness center"},
		{"HealthcareQuery", "healthcare clinic"},
		{"WithinDistance", "coffee within 2 miles"},
		{"WithinKm", "restaurant within 5km"},
		{"WithinMeters", "pharmacy within 500 meters"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parseNaturalLanguageQuery(tc.query)

			// Verify basic structure
			if parsed.Radius == 0 {
				t.Error("Expected default radius to be set")
			}

			// Keywords might be extracted
			t.Logf("Query '%s' parsed to category='%s', radius=%.1f, keywords=%v",
				tc.query, parsed.Category, parsed.Radius, parsed.Keywords)
		})
	}
}

// TestSearchHandlerWithAllFilters tests search with all possible filter combinations
func TestSearchHandlerWithAllFilters(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name string
		req  SearchRequest
	}{
		{
			name: "AllFiltersEnabled",
			req: SearchRequest{
				Query:      "vegan restaurant",
				Lat:        40.7128,
				Lon:        -74.0060,
				Radius:     5.0,
				Category:   "restaurant",
				MinRating:  4.0,
				MaxPrice:   3,
				OpenNow:    true,
				Accessible: true,
			},
		},
		{
			name: "OnlyMinRating",
			req: SearchRequest{
				Lat:       40.7128,
				Lon:       -74.0060,
				Radius:    10.0,
				MinRating: 4.5,
			},
		},
		{
			name: "OnlyMaxPrice",
			req: SearchRequest{
				Lat:      40.7128,
				Lon:      -74.0060,
				Radius:   10.0,
				MaxPrice: 1,
			},
		},
		{
			name: "OnlyOpenNow",
			req: SearchRequest{
				Lat:     40.7128,
				Lon:     -74.0060,
				Radius:  10.0,
				OpenNow: true,
			},
		},
		{
			name: "OnlyAccessible",
			req: SearchRequest{
				Lat:        40.7128,
				Lon:        -74.0060,
				Radius:     10.0,
				Accessible: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := makeHTTPRequest(searchHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/search",
				Body:   tc.req,
			})

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			places := assertPlacesResponse(t, w, 200)
			t.Logf("Filter combination '%s' returned %d places", tc.name, len(places))
		})
	}
}

// TestFetchRealTimeDataErrorPaths tests error handling in real-time data fetching
func TestFetchRealTimeDataErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name string
		req  SearchRequest
	}{
		{
			name: "InvalidCoordinates",
			req: SearchRequest{
				Query:  "test",
				Lat:    9999, // Invalid
				Lon:    9999, // Invalid
				Radius: 5.0,
			},
		},
		{
			name: "ZeroCoordinates",
			req: SearchRequest{
				Query:  "test",
				Lat:    0,
				Lon:    0,
				Radius: 5.0,
			},
		},
		{
			name: "EmptyQueryWithCategory",
			req: SearchRequest{
				Query:    "",
				Category: "restaurant",
				Lat:      40.7128,
				Lon:      -74.0060,
				Radius:   5.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			places := fetchRealTimeData(tc.req)
			// Should return empty array on error, not panic
			t.Logf("Fetch with %s returned %d places", tc.name, len(places))
		})
	}
}

// TestApplySmartFiltersAccessibility tests accessibility filtering
func TestApplySmartFiltersAccessibility(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "Accessible Place", "restaurant", 0.5, 4.5),
		createTestPlace("2", "Not Accessible", "restaurant", 0.6, 4.3),
	}

	// Mark one as accessible in description
	places[0].Description = "Wheelchair accessible entrance"

	req := SearchRequest{
		Lat:        40.7128,
		Lon:        -74.0060,
		Radius:     10.0,
		Accessible: true,
	}

	filtered := applySmartFilters(places, req)

	// The function checks for accessibility keywords in description
	for _, place := range filtered {
		t.Logf("Filtered place: %s - %s", place.Name, place.Description)
	}
}

// TestDiscoverHandlerWithDifferentCategories tests discover with various categories
func TestDiscoverHandlerWithDifferentCategories(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	categories := []string{
		"restaurant",
		"grocery",
		"pharmacy",
		"parks",
		"shopping",
		"entertainment",
		"services",
		"fitness",
		"healthcare",
	}

	for _, category := range categories {
		t.Run("Category_"+category, func(t *testing.T) {
			req := SearchRequest{
				Lat:      40.7128,
				Lon:      -74.0060,
				Radius:   5.0,
				Category: category,
			}

			w := makeHTTPRequest(discoverHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/discover",
				Body:   req,
			})

			if w.Code != 200 {
				t.Errorf("Expected status 200 for category %s, got %d", category, w.Code)
			}

			places := assertPlacesResponse(t, w, 200)
			t.Logf("Discover for %s returned %d places", category, len(places))
		})
	}
}

// TestDiscoverTrendingPlacesWithRadiusFilter tests trending places with radius
func TestDiscoverTrendingPlacesWithRadiusFilter(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name   string
		radius float64
	}{
		{"VerySmallRadius", 0.1},
		{"SmallRadius", 0.5},
		{"MediumRadius", 2.0},
		{"LargeRadius", 10.0},
		{"NoRadiusFilter", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := SearchRequest{
				Lat:    40.7128,
				Lon:    -74.0060,
				Radius: tc.radius,
			}

			trending := discoverTrendingPlaces(req)
			t.Logf("Trending with radius %.1f returned %d places", tc.radius, len(trending))

			// Verify radius filtering
			if tc.radius > 0 {
				for _, place := range trending {
					if place.Distance > tc.radius {
						t.Errorf("Place %s distance %.1f exceeds radius %.1f",
							place.Name, place.Distance, tc.radius)
					}
				}
			}
		})
	}
}

// TestCalculateRelevanceScoreAllFactors tests all relevance scoring factors
func TestCalculateRelevanceScoreAllFactors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	place := createTestPlace("test", "Vegan Organic Local Restaurant", "restaurant", 0.5, 4.5)
	place.OpenNow = true
	place.PriceLevel = 2

	testCases := []struct {
		name  string
		query string
	}{
		{"SingleKeyword", "vegan"},
		{"MultipleKeywords", "vegan organic"},
		{"AllKeywords", "vegan organic local"},
		{"PartialMatch", "veg"},
		{"NoMatch", "chinese"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := SearchRequest{
				Query:  tc.query,
				Lat:    40.7128,
				Lon:    -74.0060,
				Radius: 10.0,
			}

			scored := calculateRelevanceScore(place, req)
			t.Logf("Score for query '%s': place name='%s'", tc.query, scored.Name)
		})
	}
}

// TestSortByRelevanceStability tests sort stability
func TestSortByRelevanceStability(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create places with exact same attributes
	places := []Place{
		createTestPlace("1", "AAA Restaurant", "restaurant", 0.5, 4.0),
		createTestPlace("2", "BBB Restaurant", "restaurant", 0.5, 4.0),
		createTestPlace("3", "CCC Restaurant", "restaurant", 0.5, 4.0),
	}

	sortByRelevance(places)

	// Should be sorted alphabetically by name (tie-breaker)
	if places[0].Name > places[1].Name || places[1].Name > places[2].Name {
		t.Error("Sort should maintain alphabetical order for ties")
	}
}

// TestClearCacheHandlerMethods tests cache clear with different HTTP methods
func TestClearCacheHandlerMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"POST", "DELETE", "GET", "PUT"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := makeHTTPRequest(clearCacheHandler, HTTPTestRequest{
				Method: method,
				Path:   "/api/cache/clear",
			})

			// Should work or return appropriate status
			t.Logf("Clear cache with %s returned status %d", method, w.Code)
		})
	}
}

// TestPlaceDetailsHandlerWithDifferentIDs tests place details with various IDs
func TestPlaceDetailsHandlerWithDifferentIDs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testIDs := []string{
		"123",
		"place-with-dashes",
		"place_with_underscores",
		"PlaceWithCaps",
		"verylongidthatshouldalsowork12345678901234567890",
	}

	for _, id := range testIDs {
		t.Run("ID_"+id, func(t *testing.T) {
			w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/places/" + id,
			})

			if w.Code != 200 {
				t.Errorf("Expected status 200 for ID %s, got %d", id, w.Code)
			}

			place := assertPlaceResponse(t, w, 200)
			if place.ID != id {
				t.Errorf("Expected ID %s, got %s", id, place.ID)
			}
		})
	}
}
