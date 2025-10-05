package main

import (
	"testing"
	"time"
)

// TestPerformance_PaletteGeneration tests palette generation performance
func TestPerformance_PaletteGeneration(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []PerformanceTestCase{
		{
			Name: "GeneratePalette_Vibrant_5Colors",
			Setup: func(t *testing.T) {
				// No setup needed
			},
			Execute: func(t *testing.T) {
				req := generateTestPaletteRequest("ocean", "vibrant", 5, "")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed to generate palette: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    500 * time.Millisecond,
			IterationCount: 10,
			Description:    "Generate 5-color vibrant palette",
		},
		{
			Name: "GeneratePalette_WithBaseColor",
			Setup: func(t *testing.T) {
				// No setup needed
			},
			Execute: func(t *testing.T) {
				req := generateTestPaletteRequest("", "analogous", 5, "#3B82F6")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed to generate palette: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    500 * time.Millisecond,
			IterationCount: 10,
			Description:    "Generate palette from base color",
		},
		{
			Name: "GeneratePalette_LargeCount",
			Setup: func(t *testing.T) {
				// No setup needed
			},
			Execute: func(t *testing.T) {
				req := generateTestPaletteRequest("tech", "vibrant", 20, "")
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil {
					t.Fatalf("Failed to generate palette: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    1 * time.Second,
			IterationCount: 5,
			Description:    "Generate 20-color palette",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// TestPerformance_AccessibilityCheck tests accessibility checking performance
func TestPerformance_AccessibilityCheck(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []PerformanceTestCase{
		{
			Name: "AccessibilityCheck_ContrastRatio",
			Execute: func(t *testing.T) {
				req := generateTestAccessibilityRequest("#000000", "#FFFFFF")
				w, err := makeHTTPRequest("POST", "/accessibility", req, accessibilityHandler)
				if err != nil {
					t.Fatalf("Failed to check accessibility: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    100 * time.Millisecond,
			IterationCount: 50,
			Description:    "Calculate contrast ratio",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// TestPerformance_HarmonyAnalysis tests harmony analysis performance
func TestPerformance_HarmonyAnalysis(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []PerformanceTestCase{
		{
			Name: "HarmonyAnalysis_3Colors",
			Execute: func(t *testing.T) {
				req := generateTestHarmonyRequest([]string{"#FF0000", "#00FF00", "#0000FF"})
				w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
				if err != nil {
					t.Fatalf("Failed to analyze harmony: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    100 * time.Millisecond,
			IterationCount: 20,
			Description:    "Analyze 3-color harmony",
		},
		{
			Name: "HarmonyAnalysis_10Colors",
			Execute: func(t *testing.T) {
				colors := []string{
					"#FF0000", "#FF7F00", "#FFFF00", "#00FF00", "#0000FF",
					"#4B0082", "#9400D3", "#FF1493", "#00CED1", "#FFD700",
				}
				req := generateTestHarmonyRequest(colors)
				w, err := makeHTTPRequest("POST", "/harmony", req, harmonyHandler)
				if err != nil {
					t.Fatalf("Failed to analyze harmony: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    200 * time.Millisecond,
			IterationCount: 10,
			Description:    "Analyze 10-color harmony",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// TestPerformance_ColorblindSimulation tests colorblind simulation performance
func TestPerformance_ColorblindSimulation(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []PerformanceTestCase{
		{
			Name: "ColorblindSimulation_Protanopia",
			Execute: func(t *testing.T) {
				req := generateTestColorblindRequest([]string{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FF00FF"}, "protanopia")
				w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
				if err != nil {
					t.Fatalf("Failed to simulate colorblindness: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    100 * time.Millisecond,
			IterationCount: 20,
			Description:    "Simulate protanopia",
		},
		{
			Name: "ColorblindSimulation_AllTypes",
			Execute: func(t *testing.T) {
				colors := []string{"#FF0000", "#00FF00", "#0000FF"}
				types := []string{"protanopia", "deuteranopia", "tritanopia"}

				for _, cbType := range types {
					req := generateTestColorblindRequest(colors, cbType)
					w, err := makeHTTPRequest("POST", "/colorblind", req, colorblindHandler)
					if err != nil {
						t.Fatalf("Failed to simulate %s: %v", cbType, err)
					}
					if w.Code != 200 {
						t.Fatalf("Expected status 200 for %s, got %d", cbType, w.Code)
					}
				}
			},
			MaxDuration:    300 * time.Millisecond,
			IterationCount: 5,
			Description:    "Simulate all colorblind types",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// TestPerformance_ExportFormats tests export performance
func TestPerformance_ExportFormats(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	palette := []interface{}{"#FF0000", "#00FF00", "#0000FF", "#FFFF00", "#FF00FF"}

	testCases := []PerformanceTestCase{
		{
			Name: "Export_CSS",
			Execute: func(t *testing.T) {
				req := map[string]interface{}{
					"palette": palette,
					"format":  "css",
				}
				w, err := makeHTTPRequest("POST", "/export", req, exportHandler)
				if err != nil {
					t.Fatalf("Failed to export: %v", err)
				}
				if w.Code != 200 {
					t.Fatalf("Expected status 200, got %d", w.Code)
				}
			},
			MaxDuration:    50 * time.Millisecond,
			IterationCount: 50,
			Description:    "Export to CSS",
		},
		{
			Name: "Export_AllFormats",
			Execute: func(t *testing.T) {
				formats := []string{"css", "json", "scss"}

				for _, format := range formats {
					req := map[string]interface{}{
						"palette": palette,
						"format":  format,
					}
					w, err := makeHTTPRequest("POST", "/export", req, exportHandler)
					if err != nil {
						t.Fatalf("Failed to export %s: %v", format, err)
					}
					if w.Code != 200 {
						t.Fatalf("Expected status 200 for %s, got %d", format, w.Code)
					}
				}
			},
			MaxDuration:    100 * time.Millisecond,
			IterationCount: 20,
			Description:    "Export all formats",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// TestPerformance_CacheHitRate tests Redis cache performance
func TestPerformance_CacheHitRate(t *testing.T) {
	if redisClient == nil {
		t.Skip("Redis not available, skipping cache performance test")
	}

	cleanup := setupTestEnvironment(t)
	defer cleanup()

	t.Run("CachePerformance", func(t *testing.T) {
		req := generateTestPaletteRequest("ocean", "vibrant", 5, "")

		// First request - cache miss
		duration1 := measureExecutionTime(t, "CacheMiss", func() {
			w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
			if err != nil || w.Code != 200 {
				t.Fatalf("Failed to generate palette: %v", err)
			}
		})

		// Subsequent requests - cache hit
		var totalDuration time.Duration
		iterations := 10

		for i := 0; i < iterations; i++ {
			duration := measureExecutionTime(t, "CacheHit", func() {
				w, err := makeHTTPRequest("POST", "/generate", req, generateHandler)
				if err != nil || w.Code != 200 {
					t.Fatalf("Failed to generate palette: %v", err)
				}
			})
			totalDuration += duration
		}

		avgCacheHitDuration := totalDuration / time.Duration(iterations)

		t.Logf("Cache miss duration: %v", duration1)
		t.Logf("Avg cache hit duration: %v", avgCacheHitDuration)

		// Cache hits should be faster (or at least not slower)
		if avgCacheHitDuration > duration1*2 {
			t.Logf("Warning: Cache hits are slower than expected")
		}
	})
}

// TestPerformance_ColorConversion tests color conversion utility performance
func TestPerformance_ColorConversion(t *testing.T) {
	testCases := []PerformanceTestCase{
		{
			Name: "HexToRGB_Conversion",
			Execute: func(t *testing.T) {
				for i := 0; i < 100; i++ {
					hexToRGB("#3B82F6")
				}
			},
			MaxDuration:    10 * time.Millisecond,
			IterationCount: 10,
			Description:    "Convert hex to RGB 100 times",
		},
		{
			Name: "HSLToHex_Conversion",
			Execute: func(t *testing.T) {
				for i := 0; i < 100; i++ {
					hslToHex(210, 85, 60)
				}
			},
			MaxDuration:    20 * time.Millisecond,
			IterationCount: 10,
			Description:    "Convert HSL to hex 100 times",
		},
		{
			Name: "HexToHSL_Conversion",
			Execute: func(t *testing.T) {
				for i := 0; i < 100; i++ {
					hexToHSL("#3B82F6")
				}
			},
			MaxDuration:    30 * time.Millisecond,
			IterationCount: 10,
			Description:    "Convert hex to HSL 100 times",
		},
	}

	for _, tc := range testCases {
		RunPerformanceTest(t, tc)
	}
}

// BenchmarkPaletteGeneration benchmarks palette generation
func BenchmarkPaletteGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generatePalette("ocean", "vibrant", 5, "")
	}
}

// BenchmarkContrastCalculation benchmarks contrast ratio calculation
func BenchmarkContrastCalculation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateContrastRatio("#000000", "#FFFFFF")
	}
}

// BenchmarkHarmonyAnalysis benchmarks harmony analysis
func BenchmarkHarmonyAnalysis(b *testing.B) {
	colors := []string{"#FF0000", "#00FF00", "#0000FF"}
	for i := 0; i < b.N; i++ {
		analyzeColorHarmony(colors)
	}
}

// BenchmarkColorblindSimulation benchmarks colorblind simulation
func BenchmarkColorblindSimulation(b *testing.B) {
	colors := []string{"#FF0000", "#00FF00", "#0000FF"}
	for i := 0; i < b.N; i++ {
		simulateColorblindness(colors, "protanopia")
	}
}
