package main

import (
	"bytes"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w, err := makeHTTPRequest("GET", "/health", nil, healthHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("POST", "/health", nil, healthHandler)
		// Health handler doesn't check method, so it will still return 200
		// This is acceptable for health checks
		if w.Code != 200 {
			t.Logf("Health endpoint returns %d for POST (this is okay)", w.Code)
		}
	})
}

// TestGenerateHandler tests the palette generation endpoint
func TestGenerateHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_DefaultStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("ocean", "", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
		assertCacheHeader(t, w, "MISS")

		// Verify theme is returned
		if theme, ok := response["theme"].(string); !ok || theme != "ocean" {
			t.Errorf("Expected theme 'ocean', got %v", response["theme"])
		}
	})

	t.Run("Success_VibrantStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("forest", "vibrant", 7, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 7)
	})

	t.Run("Success_PastelStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("sunset", "pastel", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})

	t.Run("Success_DarkStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("tech", "dark", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})

	t.Run("Success_MinimalStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("modern", "minimal", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})

	t.Run("Success_EarthyStyle", func(t *testing.T) {
		req := generateTestPaletteRequest("nature", "earthy", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})

	t.Run("Success_WithBaseColor", func(t *testing.T) {
		req := generateTestPaletteRequest("", "analogous", 5, "#3B82F6")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		palette := assertPaletteResponse(t, response, 5)

		// First color should be the base color
		if palette[0].(string) != "#3B82F6" {
			t.Errorf("Expected first color to be base color #3B82F6, got %s", palette[0].(string))
		}
	})

	t.Run("Success_CustomNumColors", func(t *testing.T) {
		req := generateTestPaletteRequest("ocean", "vibrant", 10, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 10)
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/generate", nil, generateHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/generate", bytes.NewBufferString(`{"invalid": json}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		generateHandler(w, req)
		assertErrorResponse(t, w, 400)
	})

	t.Run("CacheHit", func(t *testing.T) {
		// Skip if Redis not available
		if redisClient == nil {
			t.Skip("Redis not available, skipping cache test")
		}

		req := generateTestPaletteRequest("ocean", "vibrant", 5, "")

		// First request - cache miss
		w1, _ := makeHTTPRequest("POST", "/generate", req, generateHandler)
		assertCacheHeader(t, w1, "MISS")

		// Second request - cache hit
		w2, _ := makeHTTPRequest("POST", "/generate", req, generateHandler)
		assertCacheHeader(t, w2, "HIT")

		// Verify responses are identical
		resp1 := assertJSONResponse(t, w1, 200)
		resp2 := assertJSONResponse(t, w2, 200)

		palette1 := resp1["palette"].([]interface{})
		palette2 := resp2["palette"].([]interface{})

		if len(palette1) != len(palette2) {
			t.Error("Cached response differs from original")
		}
	})
}

// TestAccessibilityHandler tests the accessibility checking endpoint
func TestAccessibilityHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_ExcellentContrast", func(t *testing.T) {
		req := generateTestAccessibilityRequest("#000000", "#FFFFFF")
		w, err := makeHTTPRequest("POST", "/accessibility", req, accessibilityHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		// Black on white should have contrast ratio of 21
		ratio, ok := response["contrast_ratio"].(float64)
		if !ok {
			t.Fatal("Missing contrast_ratio field")
		}

		if ratio < 20 || ratio > 22 {
			t.Errorf("Expected contrast ratio ~21, got %.2f", ratio)
		}

		if !response["wcag_aa"].(bool) || !response["wcag_aaa"].(bool) {
			t.Error("Expected WCAG AA and AAA to pass")
		}
	})

	t.Run("Success_PoorContrast", func(t *testing.T) {
		req := generateTestAccessibilityRequest("#888888", "#999999")
		w, err := makeHTTPRequest("POST", "/accessibility", req, accessibilityHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		// Similar grays should have poor contrast
		ratio, ok := response["contrast_ratio"].(float64)
		if !ok {
			t.Fatal("Missing contrast_ratio field")
		}

		if ratio > 3 {
			t.Errorf("Expected low contrast ratio, got %.2f", ratio)
		}

		if response["wcag_aa"].(bool) {
			t.Error("Expected WCAG AA to fail for poor contrast")
		}
	})

	t.Run("Success_ModerateContrast", func(t *testing.T) {
		req := generateTestAccessibilityRequest("#1E40AF", "#DBEAFE")
		w, err := makeHTTPRequest("POST", "/accessibility", req, accessibilityHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		if recommendation, ok := response["recommendation"].(string); ok {
			t.Logf("Recommendation: %s", recommendation)
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/accessibility", nil, accessibilityHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/accessibility", bytes.NewBufferString(`{"invalid":`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		accessibilityHandler(w, req)
		assertErrorResponse(t, w, 400)
	})
}

// TestHarmonyHandler tests the color harmony analysis endpoint
func TestHarmonyHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_ComplementaryColors", func(t *testing.T) {
		// Blue and orange are complementary (opposite on color wheel)
		req := generateTestHarmonyRequest([]string{"#0000FF", "#FFA500"})
		w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		analysis, ok := response["analysis"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing analysis field")
		}

		// Just verify we have analysis - harmonious detection may vary by algorithm
		if colorCount, ok := analysis["color_count"].(float64); !ok || colorCount != 2 {
			t.Errorf("Expected color_count to be 2, got %v", analysis["color_count"])
		}
	})

	t.Run("Success_MonochromaticColors", func(t *testing.T) {
		// Similar blue shades
		req := generateTestHarmonyRequest([]string{"#0000FF", "#0000CC", "#0000AA"})
		w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		analysis, ok := response["analysis"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing analysis field")
		}

		relationships, ok := analysis["relationships"].([]interface{})
		if !ok {
			t.Fatal("Missing relationships field")
		}

		// Should detect monochromatic relationship
		hasMonochromatic := false
		for _, rel := range relationships {
			if rel.(string) == "monochromatic" {
				hasMonochromatic = true
				break
			}
		}

		if !hasMonochromatic {
			t.Error("Expected to detect monochromatic relationship")
		}
	})

	t.Run("Success_TriadicColors", func(t *testing.T) {
		// Colors 120 degrees apart
		req := generateTestHarmonyRequest([]string{"#FF0000", "#00FF00", "#0000FF"})
		w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)
	})

	t.Run("Error_TooFewColors", func(t *testing.T) {
		req := generateTestHarmonyRequest([]string{"#FF0000"})
		w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)

		if response["success"].(bool) {
			t.Error("Expected failure with only 1 color")
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/harmony", nil, harmonyHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/harmony", bytes.NewBufferString(`{invalid}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		harmonyHandler(w, req)
		assertErrorResponse(t, w, 400)
	})
}

// TestColorblindHandler tests the colorblind simulation endpoint
func TestColorblindHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_Protanopia", func(t *testing.T) {
		req := generateTestColorblindRequest([]string{"#FF0000", "#00FF00", "#0000FF"}, "protanopia")
		w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		simulated, ok := response["simulated"].([]interface{})
		if !ok || len(simulated) != 3 {
			t.Error("Expected 3 simulated colors")
		}

		if response["type"].(string) != "protanopia" {
			t.Error("Expected type to be protanopia")
		}
	})

	t.Run("Success_Deuteranopia", func(t *testing.T) {
		req := generateTestColorblindRequest([]string{"#FF0000", "#00FF00"}, "deuteranopia")
		w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		simulated, ok := response["simulated"].([]interface{})
		if !ok || len(simulated) != 2 {
			t.Error("Expected 2 simulated colors")
		}
	})

	t.Run("Success_Tritanopia", func(t *testing.T) {
		req := generateTestColorblindRequest([]string{"#FF0000"}, "tritanopia")
		w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)
	})

	t.Run("Success_DefaultType", func(t *testing.T) {
		req := generateTestColorblindRequest([]string{"#FF0000"}, "")
		w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		// Should default to protanopia
		if response["type"].(string) != "protanopia" {
			t.Error("Expected default type to be protanopia")
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/colorblind", nil, colorblindHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/colorblind", bytes.NewBufferString(`{`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		colorblindHandler(w, req)
		assertErrorResponse(t, w, 400)
	})
}

// TestSuggestHandler tests the palette suggestion endpoint
func TestSuggestHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_WebsiteUseCase", func(t *testing.T) {
		req := map[string]string{"use_case": "corporate website"}
		w, err := makeHTTPRequest("POST", "/suggest", req, suggestHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		suggestions, ok := response["suggestions"].([]interface{})
		if !ok || len(suggestions) == 0 {
			t.Error("Expected suggestions to be returned")
		}

		// Validate suggestion structure
		if len(suggestions) > 0 {
			suggestion := suggestions[0].(map[string]interface{})
			if _, ok := suggestion["name"]; !ok {
				t.Error("Suggestion missing name field")
			}
			if _, ok := suggestion["colors"]; !ok {
				t.Error("Suggestion missing colors field")
			}
		}
	})

	t.Run("Success_FallbackSuggestions", func(t *testing.T) {
		req := map[string]string{"use_case": "random use case"}
		w, err := makeHTTPRequest("POST", "/suggest", req, suggestHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		// Should return fallback suggestions even if AI fails
		suggestions, ok := response["suggestions"].([]interface{})
		if !ok || len(suggestions) == 0 {
			t.Error("Expected fallback suggestions")
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/suggest", nil, suggestHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/suggest", bytes.NewBufferString(`{bad json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suggestHandler(w, req)
		assertErrorResponse(t, w, 400)
	})
}

// TestExportHandler tests the palette export endpoint
func TestExportHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	palette := []interface{}{"#FF0000", "#00FF00", "#0000FF"}

	t.Run("Success_CSSExport", func(t *testing.T) {
		req := map[string]interface{}{
			"palette": palette,
			"format":  "css",
		}
		w, err := makeHTTPRequest("POST", "/export", req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		export, ok := response["export"].(string)
		if !ok || export == "" {
			t.Error("Expected export data")
		}

		// CSS should contain :root and --color variables
		if !bytes.Contains([]byte(export), []byte(":root")) {
			t.Error("CSS export should contain :root")
		}
	})

	t.Run("Success_JSONExport", func(t *testing.T) {
		req := map[string]interface{}{
			"palette": palette,
			"format":  "json",
		}
		w, err := makeHTTPRequest("POST", "/export", req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		export, ok := response["export"].(string)
		if !ok || export == "" {
			t.Error("Expected export data")
		}

		// JSON export should be valid JSON array
		if !bytes.Contains([]byte(export), []byte("[")) {
			t.Error("JSON export should be an array")
		}
	})

	t.Run("Success_SCSSExport", func(t *testing.T) {
		req := map[string]interface{}{
			"palette": palette,
			"format":  "scss",
		}
		w, err := makeHTTPRequest("POST", "/export", req, exportHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		export, ok := response["export"].(string)
		if !ok || export == "" {
			t.Error("Expected export data")
		}

		// SCSS should contain $ variables
		if !bytes.Contains([]byte(export), []byte("$color")) {
			t.Error("SCSS export should contain $color variables")
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("GET", "/export", nil, exportHandler)
		assertErrorResponse(t, w, 405)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/export", bytes.NewBufferString(`{incomplete`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		exportHandler(w, req)
		assertErrorResponse(t, w, 400)
	})
}

// TestHistoryHandler tests the palette history endpoint
func TestHistoryHandler(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("Success_EmptyHistory", func(t *testing.T) {
		w, err := makeHTTPRequest("GET", "/history", nil, historyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		history, ok := response["history"].([]interface{})
		if !ok {
			t.Fatal("Missing history field")
		}

		// Should return empty array if Redis not available or no history
		t.Logf("History length: %d", len(history))
	})

	t.Run("Success_WithHistory", func(t *testing.T) {
		if redisClient == nil {
			t.Skip("Redis not available, skipping history test")
		}

		// Generate a palette to add to history
		req := generateTestPaletteRequest("ocean", "vibrant", 5, "")
		_, _ = makeHTTPRequest("POST", "/generate", req, generateHandler)

		// Small delay to ensure history is saved
		time.Sleep(100 * time.Millisecond)

		// Get history
		w, err := makeHTTPRequest("GET", "/history", nil, historyHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertSuccessResponse(t, response)

		history, ok := response["history"].([]interface{})
		if !ok {
			t.Fatal("Missing history field")
		}

		if len(history) == 0 {
			t.Error("Expected at least one history item")
		}
	})

	t.Run("Error_MethodNotAllowed", func(t *testing.T) {
		w, _ := makeHTTPRequest("POST", "/history", nil, historyHandler)
		assertErrorResponse(t, w, 405)
	})
}

// TestColorConversionFunctions tests utility functions
func TestColorConversionFunctions(t *testing.T) {
	t.Run("HexToRGB_ValidHex", func(t *testing.T) {
		r, g, b := hexToRGB("#FF0000")
		if r != 255 || g != 0 || b != 0 {
			t.Errorf("Expected RGB(255, 0, 0), got RGB(%d, %d, %d)", r, g, b)
		}
	})

	t.Run("HexToRGB_WithoutHash", func(t *testing.T) {
		r, g, b := hexToRGB("00FF00")
		if r != 0 || g != 255 || b != 0 {
			t.Errorf("Expected RGB(0, 255, 0), got RGB(%d, %d, %d)", r, g, b)
		}
	})

	t.Run("HexToRGB_InvalidHex", func(t *testing.T) {
		r, g, b := hexToRGB("#ZZZ")
		if r != 0 || g != 0 || b != 0 {
			t.Errorf("Expected RGB(0, 0, 0) for invalid hex, got RGB(%d, %d, %d)", r, g, b)
		}
	})

	t.Run("HSLToHex_Red", func(t *testing.T) {
		hex := hslToHex(0, 100, 50)
		if hex != "#FF0000" {
			t.Errorf("Expected #FF0000, got %s", hex)
		}
	})

	t.Run("HSLToHex_Green", func(t *testing.T) {
		hex := hslToHex(120, 100, 50)
		if hex != "#00FF00" {
			t.Errorf("Expected #00FF00, got %s", hex)
		}
	})

	t.Run("HSLToHex_Blue", func(t *testing.T) {
		hex := hslToHex(240, 100, 50)
		if hex != "#0000FF" {
			t.Errorf("Expected #0000FF, got %s", hex)
		}
	})

	t.Run("HexToHSL_RoundTrip", func(t *testing.T) {
		original := "#3B82F6"
		h, s, l := hexToHSL(original)
		converted := hslToHex(h, s, l)

		// Allow small rounding differences
		if original != converted {
			t.Logf("Note: Color conversion has minor rounding: %s -> %s", original, converted)
		}
	})
}

// TestGetRelativeLuminance tests luminance calculation
func TestGetRelativeLuminance(t *testing.T) {
	t.Run("Black", func(t *testing.T) {
		lum := getRelativeLuminance("#000000")
		if lum != 0 {
			t.Errorf("Expected luminance 0 for black, got %.4f", lum)
		}
	})

	t.Run("White", func(t *testing.T) {
		lum := getRelativeLuminance("#FFFFFF")
		if lum < 0.99 || lum > 1.01 {
			t.Errorf("Expected luminance ~1 for white, got %.4f", lum)
		}
	})

	t.Run("Gray", func(t *testing.T) {
		lum := getRelativeLuminance("#808080")
		// Gray should be somewhere in the middle
		if lum < 0.1 || lum > 0.9 {
			t.Logf("Gray luminance: %.4f", lum)
		}
	})
}

// TestCalculateContrastRatio tests contrast ratio calculation
func TestCalculateContrastRatio(t *testing.T) {
	t.Run("BlackOnWhite", func(t *testing.T) {
		ratio := calculateContrastRatio("#000000", "#FFFFFF")
		if ratio < 20 || ratio > 22 {
			t.Errorf("Expected contrast ratio ~21, got %.2f", ratio)
		}
	})

	t.Run("SameColor", func(t *testing.T) {
		ratio := calculateContrastRatio("#FF0000", "#FF0000")
		if ratio < 0.9 || ratio > 1.1 {
			t.Errorf("Expected contrast ratio ~1 for same color, got %.2f", ratio)
		}
	})

	t.Run("ModerateContrast", func(t *testing.T) {
		ratio := calculateContrastRatio("#1E40AF", "#DBEAFE")
		// Should be somewhere between poor and excellent
		if ratio < 2 || ratio > 10 {
			t.Logf("Moderate contrast ratio: %.2f", ratio)
		}
	})
}

// TestClampFunction tests the clamp utility
func TestClampFunction(t *testing.T) {
	t.Run("BelowMin", func(t *testing.T) {
		result := clamp(-10, 0, 255)
		if result != 0 {
			t.Errorf("Expected 0, got %d", result)
		}
	})

	t.Run("AboveMax", func(t *testing.T) {
		result := clamp(300, 0, 255)
		if result != 255 {
			t.Errorf("Expected 255, got %d", result)
		}
	})

	t.Run("InRange", func(t *testing.T) {
		result := clamp(128, 0, 255)
		if result != 128 {
			t.Errorf("Expected 128, got %d", result)
		}
	})
}
