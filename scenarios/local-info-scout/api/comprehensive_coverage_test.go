package main

import (
	"net/http"
	"testing"
)

// TestSearchHandlerInvalidMethod tests search handler with invalid HTTP methods
func TestSearchHandlerInvalidMethod(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GET_NotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d for GET, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("PUT_NotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/search",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d for PUT, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})

	t.Run("DELETE_NotAllowed", func(t *testing.T) {
		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/search",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d for DELETE, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
}

// TestSearchHandlerInvalidJSON tests search handler with malformed JSON
func TestSearchHandlerInvalidJSON(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name string
		body string
	}{
		{"EmptyBraces", "{}"},
		{"MalformedJSON", "{invalid json}"},
		{"UnclosedBraces", "{\"query\": \"test\""},
		{"ExtraComma", "{\"query\": \"test\",}"},
		{"InvalidNumbers", "{\"lat\": \"not-a-number\"}"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := makeHTTPRequest(searchHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/search",
				Body:   tc.body,
			})

			// Empty braces should succeed, others should fail
			if tc.name == "EmptyBraces" {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for empty JSON, got %d", w.Code)
				}
			} else {
				if w.Code == http.StatusOK {
					t.Errorf("Expected error status for %s, got 200", tc.name)
				}
			}
		})
	}
}

// TestSearchHandlerEdgeCases tests edge cases in search
func TestSearchHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ZeroRadius", func(t *testing.T) {
		req := SearchRequest{
			Query:  "restaurant",
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 0, // Zero radius
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("NegativeCoordinates", func(t *testing.T) {
		req := SearchRequest{
			Query:  "restaurant",
			Lat:    -40.7128,
			Lon:    -174.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for negative coords, got %d", w.Code)
		}
	})

	t.Run("VeryLargeRadius", func(t *testing.T) {
		req := SearchRequest{
			Query:  "restaurant",
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 1000.0, // Very large radius
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for large radius, got %d", w.Code)
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		req := SearchRequest{
			Query:  "",
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty query, got %d", w.Code)
		}
	})

	t.Run("VeryLongQuery", func(t *testing.T) {
		longQuery := ""
		for i := 0; i < 1000; i++ {
			longQuery += "restaurant "
		}

		req := SearchRequest{
			Query:  longQuery,
			Lat:    40.7128,
			Lon:    -74.0060,
			Radius: 5.0,
		}

		w := makeHTTPRequest(searchHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body:   req,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for long query, got %d", w.Code)
		}
	})
}

// TestPlaceDetailsHandlerEdgeCases tests edge cases in place details
func TestPlaceDetailsHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyID", func(t *testing.T) {
		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/",
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty ID, got %d", w.Code)
		}
	})

	t.Run("VeryLongID", func(t *testing.T) {
		longID := ""
		for i := 0; i < 500; i++ {
			longID += "a"
		}

		w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/places/" + longID,
		})

		// Should still work, just returns mock data
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for long ID, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInID", func(t *testing.T) {
		specialIDs := []string{
			"test-123",
			"test_place",
			"test.place",
			"test123",
		}

		for _, id := range specialIDs {
			w := makeHTTPRequest(placeDetailsHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/places/" + id,
			})

			if w.Code != http.StatusOK {
				t.Logf("Special char ID '%s' returned status %d", id, w.Code)
			}
		}
	})
}

// TestParseNaturalLanguageQueryEdgeCases tests edge cases in query parsing
func TestParseNaturalLanguageQueryEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name          string
		query         string
		expectKeyword bool
	}{
		{"EmptyString", "", false},
		{"OnlySpaces", "   ", false},
		{"OnlySymbols", "!@#$%^&*()", false},
		{"Numbers", "12345", false},
		{"VeganKeyword", "vegan restaurant", true},
		{"OrganicKeyword", "organic food", true},
		{"HealthyKeyword", "healthy options", true},
		{"FastKeyword", "fast food", true},
		{"CheapKeyword", "cheap eats", true},
		{"LuxuryKeyword", "luxury dining", true},
		{"LocalKeyword", "local shops", true},
		{"NewKeyword", "new places", true},
		{"TwentyFourHour", "24 hour stores", true},
		{"OpenLate", "open late", true},
		{"MixedCase", "VEGAN ORGANIC", true},
		{"MultipleKeywords", "vegan organic healthy local", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parseNaturalLanguageQuery(tc.query)

			if tc.expectKeyword && len(parsed.Keywords) == 0 {
				t.Errorf("Expected keywords for '%s', got none", tc.query)
			}

			// Verify defaults
			if parsed.Radius == 0 {
				t.Error("Expected default radius to be set")
			}
		})
	}
}

// TestApplySmartFiltersComprehensive tests all filter combinations
func TestApplySmartFiltersComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "High Rating Open", "restaurant", 0.5, 4.8),
		createTestPlace("2", "Low Rating Closed", "restaurant", 0.6, 3.2),
		createTestPlace("3", "Mid Rating Open", "grocery", 0.7, 4.0),
		createTestPlace("4", "Expensive Place", "restaurant", 0.8, 4.5),
	}
	places[0].OpenNow = true
	places[0].PriceLevel = 2
	places[1].OpenNow = false
	places[1].PriceLevel = 1
	places[2].OpenNow = true
	places[2].PriceLevel = 2
	places[3].PriceLevel = 4

	t.Run("FilterByCategory", func(t *testing.T) {
		req := SearchRequest{
			Category: "restaurant",
			Lat:      40.7128,
			Lon:      -74.0060,
			Radius:   10.0,
		}

		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Category != "restaurant" {
				t.Errorf("Expected only restaurants, got %s", place.Category)
			}
		}
	})

	t.Run("FilterByMinRating", func(t *testing.T) {
		req := SearchRequest{
			Lat:       40.7128,
			Lon:       -74.0060,
			Radius:    10.0,
			MinRating: 4.0,
		}

		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Rating < 4.0 {
				t.Errorf("Expected rating >= 4.0, got %.1f", place.Rating)
			}
		}
	})

	t.Run("FilterByMaxPrice", func(t *testing.T) {
		req := SearchRequest{
			Lat:      40.7128,
			Lon:      -74.0060,
			Radius:   10.0,
			MaxPrice: 2,
		}

		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.PriceLevel > 2 {
				t.Errorf("Expected price <= 2, got %d", place.PriceLevel)
			}
		}
	})

	t.Run("FilterByOpenNow", func(t *testing.T) {
		req := SearchRequest{
			Lat:     40.7128,
			Lon:     -74.0060,
			Radius:  10.0,
			OpenNow: true,
		}

		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if !place.OpenNow {
				t.Errorf("Expected only open places, got closed place: %s", place.Name)
			}
		}
	})

	t.Run("CombinedFilters", func(t *testing.T) {
		req := SearchRequest{
			Category:  "restaurant",
			Lat:       40.7128,
			Lon:       -74.0060,
			Radius:    10.0,
			MinRating: 4.0,
			MaxPrice:  3,
			OpenNow:   true,
		}

		filtered := applySmartFilters(places, req)

		for _, place := range filtered {
			if place.Category != "restaurant" {
				t.Errorf("Expected restaurant, got %s", place.Category)
			}
			if place.Rating < 4.0 {
				t.Errorf("Expected rating >= 4.0, got %.1f", place.Rating)
			}
			if place.PriceLevel > 3 {
				t.Errorf("Expected price <= 3, got %d", place.PriceLevel)
			}
			if !place.OpenNow {
				t.Error("Expected only open places")
			}
		}
	})
}

// TestSortByRelevanceComprehensive tests sorting with various scenarios
func TestSortByRelevanceComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SortByRating", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Low", "restaurant", 0.5, 3.0),
			createTestPlace("2", "High", "restaurant", 0.5, 5.0),
			createTestPlace("3", "Mid", "restaurant", 0.5, 4.0),
		}

		sortByRelevance(places)

		// Should be sorted by rating descending
		if places[0].Rating < places[1].Rating || places[1].Rating < places[2].Rating {
			t.Error("Places not sorted by rating descending")
		}
	})

	t.Run("SortByDistance", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Far", "restaurant", 5.0, 4.0),
			createTestPlace("2", "Near", "restaurant", 0.5, 4.0),
			createTestPlace("3", "Mid", "restaurant", 2.0, 4.0),
		}

		sortByRelevance(places)

		// Should be sorted by distance ascending (when ratings are equal)
		if places[0].Distance > places[1].Distance || places[1].Distance > places[2].Distance {
			t.Error("Places not sorted by distance ascending")
		}
	})

	t.Run("SortByName", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Zebra", "restaurant", 0.5, 4.0),
			createTestPlace("2", "Apple", "restaurant", 0.5, 4.0),
			createTestPlace("3", "Mango", "restaurant", 0.5, 4.0),
		}

		sortByRelevance(places)

		// Should be sorted by name alphabetically (when rating and distance are equal)
		if places[0].Name > places[1].Name || places[1].Name > places[2].Name {
			t.Error("Places not sorted by name alphabetically")
		}
	})

	t.Run("EmptyArray", func(t *testing.T) {
		places := []Place{}
		sortByRelevance(places)
		// Should not panic
		if len(places) != 0 {
			t.Error("Empty array should remain empty")
		}
	})

	t.Run("SingleItem", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Single", "restaurant", 0.5, 4.0),
		}
		sortByRelevance(places)
		// Should not panic
		if len(places) != 1 {
			t.Error("Single item array should remain single")
		}
	})
}

// TestDiscoverHiddenGemsFiltering tests hidden gems discovery filtering
func TestDiscoverHiddenGemsFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	places := []Place{
		createTestPlace("1", "McDonald's", "restaurant", 0.5, 4.0),
		createTestPlace("2", "Starbucks Coffee", "restaurant", 0.6, 4.1),
		createTestPlace("3", "Local Gem", "restaurant", 0.7, 4.5),
		createTestPlace("4", "Another Local", "restaurant", 0.8, 4.0),
	}

	req := SearchRequest{
		Lat:    40.7128,
		Lon:    -74.0060,
		Radius: 10.0,
	}

	gems := discoverHiddenGems(places, req)

	// Should filter out chain stores
	for _, gem := range gems {
		if isChainStore(gem.Name) {
			t.Errorf("Hidden gems should not include chain stores: %s", gem.Name)
		}
	}
}

// TestGetCacheKeyUniqueness tests cache key generation uniqueness
func TestGetCacheKeyUniqueness(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	req1 := SearchRequest{
		Query:    "restaurant",
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	req2 := SearchRequest{
		Query:    "restaurant",
		Lat:      40.7128,
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "grocery", // Different category
	}

	req3 := SearchRequest{
		Query:    "restaurant",
		Lat:      40.7129, // Different lat
		Lon:      -74.0060,
		Radius:   5.0,
		Category: "restaurant",
	}

	key1 := getCacheKey(req1)
	key2 := getCacheKey(req2)
	key3 := getCacheKey(req3)

	if key1 == key2 {
		t.Error("Different categories should produce different cache keys")
	}

	if key1 == key3 {
		t.Error("Different coordinates should produce different cache keys")
	}

	if key2 == key3 {
		t.Error("Different requests should produce different cache keys")
	}
}

// TestIsChainStoreComprehensive tests chain store detection
func TestIsChainStoreComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	chainStores := []string{
		"McDonald's Restaurant",
		"Starbucks Coffee",
		"Subway Sandwiches",
		"Walmart Supercenter",
		"Target Store",
		"CVS Pharmacy",
		"Walgreens",
		"7-Eleven Store",
		"Dunkin' Donuts",
		"Pizza Hut",
	}

	localStores := []string{
		"Joe's Diner",
		"The Coffee Shop",
		"Main Street Pharmacy",
		"Green Garden Market",
		"Mom's Pizza",
	}

	for _, name := range chainStores {
		if !isChainStore(name) {
			t.Errorf("Expected '%s' to be detected as chain store", name)
		}
	}

	for _, name := range localStores {
		if isChainStore(name) {
			t.Errorf("Expected '%s' to NOT be detected as chain store", name)
		}
	}
}

// TestDeduplicateAndLimitEdgeCases tests deduplication edge cases
func TestDeduplicateAndLimitEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AllDuplicates", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Place", "restaurant", 0.5, 4.0),
			createTestPlace("1", "Place", "restaurant", 0.5, 4.0),
			createTestPlace("1", "Place", "restaurant", 0.5, 4.0),
		}

		result := deduplicateAndLimit(places, 10)
		if len(result) != 1 {
			t.Errorf("Expected 1 unique place, got %d", len(result))
		}
	})

	t.Run("LowLimit", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Place 1", "restaurant", 0.5, 4.0),
			createTestPlace("2", "Place 2", "restaurant", 0.6, 4.0),
			createTestPlace("3", "Place 3", "restaurant", 0.7, 4.0),
		}

		result := deduplicateAndLimit(places, 1)
		if len(result) != 1 {
			t.Errorf("Expected 1 place with limit 1, got %d", len(result))
		}
	})

	t.Run("LimitGreaterThanPlaces", func(t *testing.T) {
		places := []Place{
			createTestPlace("1", "Place 1", "restaurant", 0.5, 4.0),
			createTestPlace("2", "Place 2", "restaurant", 0.6, 4.0),
		}

		result := deduplicateAndLimit(places, 100)
		if len(result) != 2 {
			t.Errorf("Expected 2 places, got %d", len(result))
		}
	})
}

// TestEnableCORSMethodHandling tests CORS for all HTTP methods
func TestEnableCORSMethodHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := makeHTTPRequest(enableCORS(healthHandler), HTTPTestRequest{
				Method: method,
				Path:   "/health",
			})

			// Check CORS headers are present
			if w.Header().Get("Access-Control-Allow-Origin") == "" {
				t.Errorf("Missing CORS origin header for %s", method)
			}
		})
	}
}
