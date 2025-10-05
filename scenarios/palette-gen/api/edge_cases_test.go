package main

import (
	"testing"
)

// TestGenerateFromBaseColor tests all base color generation modes
func TestGenerateFromBaseColor(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Analogous", func(t *testing.T) {
		palette := generateFromBaseColor("#3B82F6", "analogous", 5)
		if len(palette) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(palette))
		}
		if palette[0] != "#3B82F6" {
			t.Errorf("Expected first color to be base color, got %s", palette[0])
		}
	})

	t.Run("Complementary", func(t *testing.T) {
		palette := generateFromBaseColor("#FF0000", "complementary", 5)
		if len(palette) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(palette))
		}
		if palette[0] != "#FF0000" {
			t.Errorf("Expected first color to be base color, got %s", palette[0])
		}
	})

	t.Run("Triadic", func(t *testing.T) {
		palette := generateFromBaseColor("#00FF00", "triadic", 5)
		if len(palette) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(palette))
		}
	})

	t.Run("Monochromatic_Default", func(t *testing.T) {
		// Any other style defaults to monochromatic
		palette := generateFromBaseColor("#0000FF", "monochromatic", 5)
		if len(palette) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(palette))
		}
	})

	t.Run("UnknownStyle_DefaultsToMonochromatic", func(t *testing.T) {
		palette := generateFromBaseColor("#FF00FF", "unknown", 5)
		if len(palette) != 5 {
			t.Errorf("Expected 5 colors, got %d", len(palette))
		}
	})
}

// TestGetThemeHue tests theme hue mapping
func TestGetThemeHue(t *testing.T) {
	testCases := []struct {
		theme       string
		expectedHue float64
	}{
		{"ocean", 200},
		{"sunset", 25},
		{"forest", 120},
		{"desert", 40},
		{"tech", 210},
		{"vintage", 30},
		{"modern", 240},
		{"nature", 90},
		{"fire", 15},
		{"ice", 195},
		{"autumn", 35},
		{"spring", 150},
		{"summer", 60},
		{"winter", 200},
		{"unknown", 210}, // default
	}

	for _, tc := range testCases {
		t.Run(tc.theme, func(t *testing.T) {
			hue := getThemeHue(tc.theme)
			if hue != tc.expectedHue {
				t.Errorf("Expected hue %.0f for theme %s, got %.0f", tc.expectedHue, tc.theme, hue)
			}
		})
	}
}

// TestGenerateFromStyle tests all style variations
func TestGenerateFromStyle(t *testing.T) {
	testCases := []struct {
		style     string
		numColors int
	}{
		{"vibrant", 5},
		{"pastel", 7},
		{"dark", 3},
		{"minimal", 5},
		{"earthy", 5},
		{"unknown", 5}, // defaults to harmonious
	}

	for _, tc := range testCases {
		t.Run(tc.style, func(t *testing.T) {
			palette := generateFromStyle(200, tc.style, tc.numColors)
			if len(palette) != tc.numColors {
				t.Errorf("Expected %d colors, got %d", tc.numColors, len(palette))
			}

			// Verify all colors are valid hex
			for i, color := range palette {
				if len(color) != 7 || color[0] != '#' {
					t.Errorf("Color %d has invalid format: %s", i, color)
				}
			}
		})
	}
}

// TestSavePaletteHistory tests history saving
func TestSavePaletteHistory(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("WithRedis", func(t *testing.T) {
		if redisClient == nil {
			t.Skip("Redis not available")
		}

		response := PaletteResponse{
			Success: true,
			Palette: []string{"#FF0000", "#00FF00", "#0000FF"},
			Name:    "Test Palette",
			Theme:   "test",
		}

		// Should not panic even if it fails
		savePaletteHistory(response)
	})

	t.Run("WithoutRedis", func(t *testing.T) {
		originalClient := redisClient
		redisClient = nil
		defer func() { redisClient = originalClient }()

		response := PaletteResponse{
			Success: true,
			Palette: []string{"#FF0000"},
			Name:    "Test",
			Theme:   "test",
		}

		// Should not panic when Redis is nil
		savePaletteHistory(response)
	})
}

// TestGetPaletteHistory tests history retrieval
func TestGetPaletteHistory(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("WithoutRedis", func(t *testing.T) {
		originalClient := redisClient
		redisClient = nil
		defer func() { redisClient = originalClient }()

		history := getPaletteHistory(10)
		if len(history) != 0 {
			t.Errorf("Expected empty history without Redis, got %d items", len(history))
		}
	})

	t.Run("WithRedis_EmptyHistory", func(t *testing.T) {
		if redisClient == nil {
			t.Skip("Redis not available")
		}

		history := getPaletteHistory(10)
		// Should return empty array, not fail
		if history == nil {
			t.Error("Expected non-nil history array")
		}
	})
}

// TestGenerateCacheKey tests cache key generation
func TestGenerateCacheKey(t *testing.T) {
	t.Run("SameRequest_SameKey", func(t *testing.T) {
		req1 := PaletteRequest{
			Theme:     "ocean",
			Style:     "vibrant",
			NumColors: 5,
			BaseColor: "",
		}

		req2 := PaletteRequest{
			Theme:     "ocean",
			Style:     "vibrant",
			NumColors: 5,
			BaseColor: "",
		}

		key1 := generateCacheKey(req1)
		key2 := generateCacheKey(req2)

		if key1 != key2 {
			t.Errorf("Expected same cache keys for identical requests")
		}
	})

	t.Run("DifferentRequest_DifferentKey", func(t *testing.T) {
		req1 := PaletteRequest{
			Theme:     "ocean",
			Style:     "vibrant",
			NumColors: 5,
		}

		req2 := PaletteRequest{
			Theme:     "forest",
			Style:     "vibrant",
			NumColors: 5,
		}

		key1 := generateCacheKey(req1)
		key2 := generateCacheKey(req2)

		if key1 == key2 {
			t.Errorf("Expected different cache keys for different requests")
		}
	})

	t.Run("WithBaseColor", func(t *testing.T) {
		req := PaletteRequest{
			Theme:     "ocean",
			Style:     "vibrant",
			NumColors: 5,
			BaseColor: "#FF0000",
		}

		key := generateCacheKey(req)
		if key == "" || len(key) < 10 {
			t.Error("Expected valid cache key")
		}
	})
}

// TestExportPalette tests all export formats in detail
func TestExportPalette(t *testing.T) {
	palette := []interface{}{"#FF0000", "#00FF00", "#0000FF"}

	t.Run("CSS_Format", func(t *testing.T) {
		result := exportPalette(palette, "css")
		if result == "" {
			t.Error("Expected CSS export")
		}
		if !contains(result, ":root") {
			t.Error("CSS should contain :root")
		}
		if !contains(result, "--color-1") {
			t.Error("CSS should contain color variables")
		}
	})

	t.Run("JSON_Format", func(t *testing.T) {
		result := exportPalette(palette, "json")
		if result == "" {
			t.Error("Expected JSON export")
		}
		if !contains(result, "[") || !contains(result, "]") {
			t.Error("JSON should be an array")
		}
	})

	t.Run("SCSS_Format", func(t *testing.T) {
		result := exportPalette(palette, "scss")
		if result == "" {
			t.Error("Expected SCSS export")
		}
		if !contains(result, "$color-") {
			t.Error("SCSS should contain $ variables")
		}
	})

	t.Run("Unknown_Format", func(t *testing.T) {
		result := exportPalette(palette, "unknown")
		if result != "" {
			t.Error("Unknown format should return empty string")
		}
	})
}

// TestGetSuggestionsForUseCase tests fallback suggestions
func TestGetSuggestionsForUseCase(t *testing.T) {
	suggestions := getSuggestionsForUseCase("any use case")

	if len(suggestions) == 0 {
		t.Error("Expected fallback suggestions")
	}

	// Verify structure of suggestions
	for i, suggestion := range suggestions {
		sug := suggestion

		if _, ok := sug["name"]; !ok {
			t.Errorf("Suggestion %d missing name", i)
		}

		if _, ok := sug["colors"]; !ok {
			t.Errorf("Suggestion %d missing colors", i)
		}

		if _, ok := sug["description"]; !ok {
			t.Errorf("Suggestion %d missing description", i)
		}

		// Verify colors are valid
		if colors, ok := sug["colors"].([]string); ok {
			if len(colors) != 5 {
				t.Errorf("Expected 5 colors in suggestion %d, got %d", i, len(colors))
			}
		}
	}
}

// TestCalculateHueRange tests hue range calculation
func TestCalculateHueRange(t *testing.T) {
	t.Run("EmptyArray", func(t *testing.T) {
		hueRange := calculateHueRange([]float64{})
		if hueRange != 0 {
			t.Errorf("Expected 0 for empty array, got %.2f", hueRange)
		}
	})

	t.Run("SingleHue", func(t *testing.T) {
		hueRange := calculateHueRange([]float64{180})
		if hueRange != 0 {
			t.Errorf("Expected 0 for single hue, got %.2f", hueRange)
		}
	})

	t.Run("MultipleHues", func(t *testing.T) {
		hueRange := calculateHueRange([]float64{0, 90, 180, 270})
		if hueRange != 270 {
			t.Errorf("Expected 270, got %.2f", hueRange)
		}
	})
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestGenerateHandler_EdgeCases tests additional edge cases
func TestGenerateHandler_EdgeCases(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("ZeroColors_DefaultsTo5", func(t *testing.T) {
		req := generateTestPaletteRequest("ocean", "vibrant", 0, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		palette := assertPaletteResponse(t, response, 5)

		if len(palette) != 5 {
			t.Errorf("Expected default 5 colors when num_colors is 0")
		}
	})

	t.Run("EmptyStyle_DefaultsToVibrant", func(t *testing.T) {
		req := generateTestPaletteRequest("tech", "", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})

	t.Run("AllBaseColorStyles", func(t *testing.T) {
		styles := []string{"analogous", "complementary", "triadic", "monochromatic"}

		for _, style := range styles {
			t.Run(style, func(t *testing.T) {
				req := generateTestPaletteRequest("", style, 5, "#3B82F6")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				response := assertJSONResponse(t, w, 200)
				palette := assertPaletteResponse(t, response, 5)

				if palette[0].(string) != "#3B82F6" {
					t.Errorf("Expected first color to be base color")
				}
			})
		}
	})
}
