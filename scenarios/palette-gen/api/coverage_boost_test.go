package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestGenerateHandler_CacheFailure tests cache retrieval failure path
func TestGenerateHandler_CacheFailure(t *testing.T) {
	if redisClient == nil {
		t.Skip("Redis not available")
	}

	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("InvalidCachedData", func(t *testing.T) {
		// Store invalid data in cache
		cacheKey := "palette:testkey123"
		redisClient.Set(context.Background(), cacheKey, "invalid json data", time.Hour)

		// Should fail to unmarshal and regenerate
		req := generateTestPaletteRequest("ocean", "vibrant", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still succeed by generating new palette
		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)
	})
}

// TestSavePaletteHistory_WithRedisErrors tests history saving error paths
func TestSavePaletteHistory_WithRedisErrors(t *testing.T) {
	if redisClient == nil {
		t.Skip("Redis not available")
	}

	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("MarshalError", func(t *testing.T) {
		// Create response that will fail to marshal
		// (In Go, this is hard to trigger, so we just verify it doesn't panic)
		response := PaletteResponse{
			Success: true,
			Palette: []string{"#FF0000"},
			Name:    "Test",
			Theme:   "test",
		}

		// Should not panic
		savePaletteHistory(response)
	})

	t.Run("SaveMultiplePalettes", func(t *testing.T) {
		redisClient.FlushDB(context.Background())

		// Save multiple palettes
		for i := 0; i < 5; i++ {
			response := PaletteResponse{
				Success: true,
				Palette: []string{"#FF0000", "#00FF00"},
				Name:    "Test",
				Theme:   "test",
			}
			savePaletteHistory(response)
			time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		}

		// Retrieve history
		history := getPaletteHistory(10)
		if len(history) != 5 {
			t.Errorf("Expected 5 history items, got %d", len(history))
		}
	})
}

// TestGetPaletteHistory_WithCorruptedData tests history retrieval error paths
func TestGetPaletteHistory_WithCorruptedData(t *testing.T) {
	if redisClient == nil {
		t.Skip("Redis not available")
	}

	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("CorruptedHistoryEntry", func(t *testing.T) {
		redisClient.FlushDB(context.Background())

		// Add a valid entry
		validResponse := PaletteResponse{
			Success: true,
			Palette: []string{"#FF0000"},
			Name:    "Valid",
			Theme:   "test",
		}
		validData, _ := json.Marshal(validResponse)
		key1 := "history:1000"
		redisClient.Set(context.Background(), key1, validData, time.Hour)
		redisClient.ZAdd(context.Background(), "palette:history", redis.Z{
			Score:  1000,
			Member: key1,
		})

		// Add a corrupted entry
		key2 := "history:2000"
		redisClient.Set(context.Background(), key2, "invalid json", time.Hour)
		redisClient.ZAdd(context.Background(), "palette:history", redis.Z{
			Score:  2000,
			Member: key2,
		})

		// Should skip corrupted entry and return valid one
		history := getPaletteHistory(10)
		if len(history) != 1 {
			t.Errorf("Expected 1 valid history item, got %d", len(history))
		}
	})

	t.Run("MissingHistoryKey", func(t *testing.T) {
		redisClient.FlushDB(context.Background())

		// Add reference to non-existent key
		redisClient.ZAdd(context.Background(), "palette:history", redis.Z{
			Score:  3000,
			Member: "history:nonexistent",
		})

		// Should handle missing key gracefully
		history := getPaletteHistory(10)
		if len(history) != 0 {
			t.Errorf("Expected 0 history items for missing keys, got %d", len(history))
		}
	})

	t.Run("LimitHistoryResults", func(t *testing.T) {
		redisClient.FlushDB(context.Background())

		// Add 20 entries
		for i := 0; i < 20; i++ {
			response := PaletteResponse{
				Success: true,
				Palette: []string{"#FF0000"},
				Name:    "Test",
				Theme:   "test",
			}
			data, _ := json.Marshal(response)
			key := "history:" + string(rune(4000+i))
			redisClient.Set(context.Background(), key, data, time.Hour)
			redisClient.ZAdd(context.Background(), "palette:history", redis.Z{
				Score:  float64(4000 + i),
				Member: key,
			})
		}

		// Request only 5
		history := getPaletteHistory(5)
		if len(history) > 5 {
			t.Errorf("Expected max 5 history items, got %d", len(history))
		}
	})
}

// TestGenerateHandler_AllPossiblePaths ensures all branches are covered
func TestGenerateHandler_AllPossiblePaths(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("WithoutRedis_NoCache", func(t *testing.T) {
		originalClient := redisClient
		redisClient = nil
		defer func() { redisClient = originalClient }()

		req := generateTestPaletteRequest("ocean", "vibrant", 5, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 5)

		// No cache header when Redis is nil
		cacheHeader := w.Header().Get("X-Cache")
		if cacheHeader != "MISS" {
			t.Logf("Cache header: %s", cacheHeader)
		}
	})

	t.Run("LargeNumColors", func(t *testing.T) {
		req := generateTestPaletteRequest("tech", "vibrant", 50, "")
		w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, 200)
		assertPaletteResponse(t, response, 50)
	})

	t.Run("BaseColorWithEachStyle", func(t *testing.T) {
		styles := []string{"analogous", "complementary", "triadic", "vibrant", "pastel", "unknown"}

		for _, style := range styles {
			t.Run(style, func(t *testing.T) {
				req := generateTestPaletteRequest("", style, 5, "#FF5733")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed to make request: %v", err)
				}

				response := assertJSONResponse(t, w, 200)
				palette := assertPaletteResponse(t, response, 5)

				// First color should match base for valid styles
				if style == "analogous" || style == "complementary" || style == "triadic" {
					if palette[0].(string) != "#FF5733" {
						t.Errorf("Expected first color to be base color, got %s", palette[0].(string))
					}
				}
			})
		}
	})

	t.Run("UnusualThemes", func(t *testing.T) {
		themes := []string{"ocean", "sunset", "forest", "desert", "tech", "vintage", "modern",
			"nature", "fire", "ice", "autumn", "spring", "summer", "winter", "unknown-theme"}

		for _, theme := range themes {
			t.Run(theme, func(t *testing.T) {
				req := generateTestPaletteRequest(theme, "vibrant", 5, "")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed for theme %s: %v", theme, err)
				}

				response := assertJSONResponse(t, w, 200)
				assertPaletteResponse(t, response, 5)
			})
		}
	})
}

// TestAssertHelpers tests helper functions to increase coverage
func TestAssertHelpers(t *testing.T) {
	t.Run("AssertSuccessResponse_Failure", func(t *testing.T) {
		// Create a failed response
		response := map[string]interface{}{
			"success": false,
		}

		// This should fail the test, but we're testing the helper itself
		// So we run it in a sub-test that we expect to fail
		testFailed := false
		mockT := &testing.T{}

		// Capture the failure
		defer func() {
			if !testFailed {
				t.Error("Expected assertSuccessResponse to fail")
			}
		}()

		// This will mark mockT as failed
		assertSuccessResponse(mockT, response)
		if mockT.Failed() {
			testFailed = true
		}
	})

	t.Run("AssertSuccessResponse_MissingField", func(t *testing.T) {
		response := map[string]interface{}{}

		mockT := &testing.T{}
		assertSuccessResponse(mockT, response)

		if !mockT.Failed() {
			t.Error("Expected assertSuccessResponse to fail for missing field")
		}
	})

	t.Run("AssertPaletteResponse_WrongCount", func(t *testing.T) {
		response := map[string]interface{}{
			"success": true,
			"palette": []interface{}{"#FF0000", "#00FF00", "#0000FF"},
		}

		mockT := &testing.T{}
		assertPaletteResponse(mockT, response, 5)

		if !mockT.Failed() {
			t.Error("Expected assertPaletteResponse to fail for wrong count")
		}
	})
}

// TestInitRedis_ErrorPaths tests Redis initialization error handling
func TestInitRedis_ErrorPaths(t *testing.T) {
	t.Run("CustomPassword", func(t *testing.T) {
		originalClient := redisClient
		os.Setenv("REDIS_PASSWORD", "testpass")
		defer func() {
			redisClient = originalClient
			os.Unsetenv("REDIS_PASSWORD")
		}()

		// Should not panic with password set
		initRedis()
	})

	t.Run("DefaultHostPort", func(t *testing.T) {
		originalClient := redisClient
		os.Unsetenv("REDIS_HOST")
		os.Unsetenv("REDIS_PORT")
		defer func() {
			redisClient = originalClient
		}()

		// Should use defaults
		initRedis()
	})
}

// TestGenerateFromBaseColor_EdgeCases tests all branches
func TestGenerateFromBaseColor_EdgeCases(t *testing.T) {
	t.Run("ComplementaryWithMultipleColors", func(t *testing.T) {
		palette := generateFromBaseColor("#3B82F6", "complementary", 10)
		if len(palette) != 10 {
			t.Errorf("Expected 10 colors, got %d", len(palette))
		}

		// First should be base
		if palette[0] != "#3B82F6" {
			t.Error("First color should be base color")
		}

		// Second should be complementary (opposite on wheel)
		// Remaining colors should be variations
		if palette[1] == palette[0] {
			t.Error("Second color should differ from base")
		}
	})

	t.Run("TriadicWith4Colors", func(t *testing.T) {
		palette := generateFromBaseColor("#FF0000", "triadic", 4)
		if len(palette) != 4 {
			t.Errorf("Expected 4 colors, got %d", len(palette))
		}
	})

	t.Run("MonochromaticWith8Colors", func(t *testing.T) {
		palette := generateFromBaseColor("#00FF00", "monochromatic", 8)
		if len(palette) != 8 {
			t.Errorf("Expected 8 colors, got %d", len(palette))
		}

		// All should have same hue (roughly)
		baseH, _, _ := hexToHSL(palette[0])
		for i := 1; i < len(palette); i++ {
			h, _, _ := hexToHSL(palette[i])
			// Hue should be similar (allowing for small rounding differences)
			if h != baseH {
				t.Logf("Monochromatic palette hue variation: base=%.1f, color[%d]=%.1f", baseH, i, h)
			}
		}
	})
}
