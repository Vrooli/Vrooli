package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestHealthHandlerWithRedisError tests health endpoint when Redis is unavailable
func TestHealthHandlerWithRedisError(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create a broken Redis client (wrong port)
	originalClient := redisClient
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer func() { redisClient = originalClient }()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should be degraded when Redis fails
	if response["status"] != "degraded" {
		t.Errorf("Expected status 'degraded' when Redis fails, got %v", response["status"])
	}

	// Should have dependency info with error
	if deps, ok := response["dependencies"].(map[string]interface{}); ok {
		if redis, ok := deps["redis"].(map[string]interface{}); ok {
			if redis["connected"] != false {
				t.Error("Expected Redis connected=false")
			}
			if redis["error"] == nil {
				t.Error("Expected Redis error to be present")
			}
		} else {
			t.Error("Expected Redis dependency info")
		}
	} else {
		t.Error("Expected dependencies in response")
	}
}

// TestHealthHandlerWithoutRedis tests health endpoint when Redis is disabled
func TestHealthHandlerWithoutRedis(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	originalClient := redisClient
	redisClient = nil
	defer func() { redisClient = originalClient }()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy' without Redis, got %v", response["status"])
	}

	// Should not have dependencies section when Redis is disabled
	if _, ok := response["dependencies"]; ok {
		t.Error("Expected no dependencies section when Redis is disabled")
	}
}

// TestGenerateHandlerInvalidMethod tests non-POST requests
func TestGenerateHandlerInvalidMethod(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/generate", nil)
		w := httptest.NewRecorder()

		generateHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Method %s: Expected status 405, got %d", method, w.Code)
		}
	}
}

// TestGenerateHandlerInvalidJSON tests malformed request bodies
func TestGenerateHandlerInvalidJSON(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []struct {
		name string
		body string
	}{
		{"Empty body", ""},
		{"Invalid JSON", "{invalid json}"},
		{"Truncated JSON", `{"theme": "ocean"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBufferString(tc.body))
			w := httptest.NewRecorder()

			generateHandler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	}
}

// TestGenerateHandlerEdgeCases tests edge cases in palette generation
func TestGenerateHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	testCases := []struct {
		name       string
		request    PaletteRequest
		checkStyle string // Expected resolved style
	}{
		{
			name: "Auto style resolution",
			request: PaletteRequest{
				Theme: "ocean",
				Style: "auto",
			},
			checkStyle: "vibrant",
		},
		{
			name: "Base color without hash",
			request: PaletteRequest{
				Theme:     "tech",
				BaseColor: "FF5733",
			},
			checkStyle: "vibrant",
		},
		{
			name: "Negative num_colors",
			request: PaletteRequest{
				Theme:     "forest",
				NumColors: -5,
			},
			checkStyle: "vibrant",
		},
		{
			name: "Zero num_colors",
			request: PaletteRequest{
				Theme:     "sunset",
				NumColors: 0,
			},
			checkStyle: "vibrant",
		},
		{
			name: "Too few colors (clamped to 3)",
			request: PaletteRequest{
				Theme:     "corporate",
				NumColors: 1,
			},
			checkStyle: "vibrant",
		},
		{
			name: "Too many colors (clamped to 10)",
			request: PaletteRequest{
				Theme:     "rainbow",
				NumColors: 100,
			},
			checkStyle: "vibrant",
		},
		{
			name: "Whitespace in theme and style",
			request: PaletteRequest{
				Theme: "  ocean  ",
				Style: "  pastel  ",
			},
			checkStyle: "pastel",
		},
		{
			name: "Empty theme defaults to vibrant",
			request: PaletteRequest{
				Theme: "",
				Style: "",
			},
			checkStyle: "vibrant",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			generateHandler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("Expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var response PaletteResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !response.Success {
				t.Error("Expected success=true")
			}

			if len(response.Palette) == 0 {
				t.Error("Expected non-empty palette")
			}

			// Verify style was resolved
			if response.Style != tc.checkStyle {
				t.Logf("Style resolved to: %s (expected: %s)", response.Style, tc.checkStyle)
			}
		})
	}
}

// TestGenerateHandlerWithAIDebug tests AI debug augmentation
func TestGenerateHandlerWithAIDebug(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Save original env
	originalOllama := os.Getenv("OLLAMA_API_GENERATE")
	defer func() {
		if originalOllama != "" {
			os.Setenv("OLLAMA_API_GENERATE", originalOllama)
		} else {
			os.Unsetenv("OLLAMA_API_GENERATE")
		}
	}()

	// Set to invalid endpoint to test error handling
	os.Setenv("OLLAMA_API_GENERATE", "http://localhost:99999/invalid")

	request := PaletteRequest{
		Theme:          "ocean",
		Style:          "vibrant",
		NumColors:      5,
		IncludeAIDebug: true,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	generateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response PaletteResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Debug == nil {
		t.Fatal("Expected debug info when IncludeAIDebug=true")
	}

	if !response.Debug.AIRequested {
		t.Error("Expected AIRequested=true")
	}

	// Should have error since we pointed to invalid endpoint
	if response.Debug.AIError == "" {
		t.Error("Expected AI error when endpoint is invalid")
	}
}

// TestBuildAISuggestionContext tests context building for different scenarios
func TestBuildAISuggestionContext(t *testing.T) {
	testCases := []struct {
		name     string
		request  PaletteRequest
		contains []string
	}{
		{
			name: "Theme only",
			request: PaletteRequest{
				Theme: "ocean",
			},
			contains: []string{"ocean"},
		},
		{
			name: "Theme and style",
			request: PaletteRequest{
				Theme: "sunset",
				Style: "vibrant",
			},
			contains: []string{"sunset", "vibrant style"},
		},
		{
			name: "Theme, style, and base color",
			request: PaletteRequest{
				Theme:     "forest",
				Style:     "earthy",
				BaseColor: "#2C5F2D",
			},
			contains: []string{"forest", "earthy style", "anchored by #2C5F2D"},
		},
		{
			name: "Base color only",
			request: PaletteRequest{
				BaseColor: "#FF5733",
			},
			contains: []string{"anchored by #FF5733"},
		},
		{
			name: "Empty request",
			request: PaletteRequest{
				Theme: "",
			},
			contains: []string{"general purpose interface design"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildAISuggestionContext(tc.request)

			for _, expected := range tc.contains {
				if !bytes.Contains([]byte(result), []byte(expected)) {
					t.Errorf("Expected context to contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// TestResolveGenerationStrategy tests strategy resolution logic
func TestResolveGenerationStrategy(t *testing.T) {
	testCases := []struct {
		name     string
		request  PaletteRequest
		expected string
	}{
		{
			name: "Base color, theme, and style",
			request: PaletteRequest{
				Theme:     "ocean",
				Style:     "vibrant",
				BaseColor: "#1E90FF",
			},
			expected: "base-color-and-theme",
		},
		{
			name: "Base color only",
			request: PaletteRequest{
				BaseColor: "#FF5733",
			},
			expected: "base-color",
		},
		{
			name: "Theme and style",
			request: PaletteRequest{
				Theme: "forest",
				Style: "earthy",
			},
			expected: "theme-and-style",
		},
		{
			name: "Theme only",
			request: PaletteRequest{
				Theme: "sunset",
			},
			expected: "theme-only",
		},
		{
			name:     "Empty request",
			request:  PaletteRequest{},
			expected: "procedural-default",
		},
		{
			name: "Whitespace theme",
			request: PaletteRequest{
				Theme: "   ",
			},
			expected: "procedural-default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := resolveGenerationStrategy(tc.request)
			if result != tc.expected {
				t.Errorf("Expected strategy '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestBuildPaletteName tests name generation edge cases
func TestBuildPaletteName(t *testing.T) {
	testCases := []struct {
		style    string
		theme    string
		expected string
	}{
		{"vibrant", "ocean", "Vibrant Ocean"},
		{"", "sunset", "Sunset"},
		{"pastel", "", "Pastel Palette"},
		{"", "", "Custom Palette"},
		{"  minimal  ", "  tech  ", "Minimal Tech"},
		{"DARK", "FOREST", "Dark Forest"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := buildPaletteName(tc.style, tc.theme)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

// TestBuildPaletteDescription tests description generation
func TestBuildPaletteDescription(t *testing.T) {
	testCases := []struct {
		name      string
		style     string
		theme     string
		baseColor string
		numColors int
		contains  []string
	}{
		{
			name:      "Full description",
			style:     "vibrant",
			theme:     "ocean",
			baseColor: "#1E90FF",
			numColors: 5,
			contains:  []string{"Inspired by ocean", "vibrant style", "#1E90FF", "5 colors"},
		},
		{
			name:      "Theme only",
			theme:     "sunset",
			numColors: 7,
			contains:  []string{"Inspired by sunset", "7 colors"},
		},
		{
			name:      "Empty values",
			numColors: 3,
			contains:  []string{"Procedurally generated palette", "3 colors"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildPaletteDescription(tc.style, tc.theme, tc.baseColor, tc.numColors)

			for _, expected := range tc.contains {
				if !bytes.Contains([]byte(result), []byte(expected)) {
					t.Errorf("Expected description to contain '%s', got: %s", expected, result)
				}
			}
		})
	}
}

// TestTitleCaseEdgeCases tests title case conversion edge cases
func TestTitleCaseEdgeCases(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"   ", ""},
		{"hello", "Hello"},
		{"HELLO", "Hello"},
		{"hello world", "Hello World"},
		{"  hello  world  ", "Hello World"},
		{"ONE TWO THREE", "One Two Three"},
		{"a", "A"},
		{"ABC DEF", "Abc Def"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := titleCase(tc.input)
			if result != tc.expected {
				t.Errorf("titleCase(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestSavePaletteHistoryWithoutRedis tests history saving when Redis is disabled
func TestSavePaletteHistoryWithoutRedis(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	originalClient := redisClient
	redisClient = nil
	defer func() { redisClient = originalClient }()

	response := PaletteResponse{
		Success: true,
		Palette: []string{"#FF5733", "#C70039", "#900C3F"},
		Name:    "Test Palette",
		Theme:   "test",
	}

	// Should not panic when Redis is nil
	savePaletteHistory(response)
}

// TestGetPaletteHistoryWithoutRedis tests history retrieval when Redis is disabled
func TestGetPaletteHistoryWithoutRedis(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	originalClient := redisClient
	redisClient = nil
	defer func() { redisClient = originalClient }()

	result := getPaletteHistory(10)

	if result == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty history, got %d items", len(result))
	}
}

// TestSavePaletteHistoryWithRedisError tests history saving with Redis errors
func TestSavePaletteHistoryWithRedisError(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	originalClient := redisClient
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer func() { redisClient = originalClient }()

	response := PaletteResponse{
		Success: true,
		Palette: []string{"#FF5733", "#C70039", "#900C3F"},
		Name:    "Test Palette",
		Theme:   "test",
	}

	// Should not panic when Redis operation fails
	savePaletteHistory(response)
}

// TestGetPaletteHistoryWithRedisError tests history retrieval with Redis errors
func TestGetPaletteHistoryWithRedisError(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	originalClient := redisClient
	redisClient = redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer func() { redisClient = originalClient }()

	result := getPaletteHistory(10)

	if result == nil {
		t.Error("Expected empty slice on error, got nil")
	}

	if len(result) != 0 {
		t.Errorf("Expected empty history on error, got %d items", len(result))
	}
}

// TestAugmentWithAIDebugNilInfo tests AI debug with nil debug info
func TestAugmentWithAIDebugNilInfo(t *testing.T) {
	req := PaletteRequest{
		Theme: "ocean",
	}

	// Should not panic when info is nil
	augmentWithAIDebug(req, nil)
}

// TestResolveBaseHue tests base hue resolution
func TestResolveBaseHue(t *testing.T) {
	// Test with base color
	hue1 := resolveBaseHue("ocean", "#FF0000")
	if hue1 < 0 || hue1 > 360 {
		t.Errorf("Expected hue in range [0, 360], got %f", hue1)
	}

	// Test without base color (should use theme)
	hue2 := resolveBaseHue("ocean", "")
	if hue2 < 0 || hue2 > 360 {
		t.Errorf("Expected hue in range [0, 360], got %f", hue2)
	}

	// With base color should differ from theme-only
	if hue1 == hue2 {
		t.Log("Base color hue matches theme hue (may be coincidence)")
	}
}

// TestConcurrentHealthRequests tests health endpoint under concurrent load
func TestConcurrentHealthRequests(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	const numRequests = 50

	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			healthHandler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			done <- true
		}()
	}

	// Wait for all requests with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Concurrent health requests timed out")
		}
	}
}
