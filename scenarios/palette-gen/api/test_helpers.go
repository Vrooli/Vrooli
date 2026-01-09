package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := logger

	// Suppress logs during tests unless VERBOSE_TESTS is set
	if os.Getenv("VERBOSE_TESTS") != "true" {
		logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	} else {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
	slog.SetDefault(logger)

	return func() {
		logger = originalLogger
		if logger != nil {
			slog.SetDefault(logger)
		}
	}
}

// setupTestRedis creates a test Redis client with mock if Redis not available
func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	originalClient := redisClient

	// Try to connect to test Redis
	testClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use a different DB for testing
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := testClient.Ping(ctx).Result()
	if err != nil {
		// Redis not available, use nil client
		redisClient = nil
		return nil, func() {
			redisClient = originalClient
		}
	}

	// Clear test database
	testClient.FlushDB(context.Background())

	redisClient = testClient

	return testClient, func() {
		testClient.FlushDB(context.Background())
		testClient.Close()
		redisClient = originalClient
	}
}

// makeHTTPRequest creates and executes an HTTP request
func makeHTTPRequest(method, path string, body interface{}, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	handler(w, req)

	return w, nil
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// assertSuccessResponse validates successful response with success field
func assertSuccessResponse(t *testing.T, response map[string]interface{}) {
	t.Helper()

	success, ok := response["success"].(bool)
	if !ok {
		t.Error("Response missing 'success' field")
		return
	}

	if !success {
		t.Errorf("Expected success=true, got false. Response: %v", response)
	}
}

// assertPaletteResponse validates palette generation response
func assertPaletteResponse(t *testing.T, response map[string]interface{}, expectedNumColors int) []interface{} {
	t.Helper()

	assertSuccessResponse(t, response)

	palette, ok := response["palette"].([]interface{})
	if !ok {
		t.Fatal("Response missing 'palette' field")
	}

	if len(palette) != expectedNumColors {
		t.Errorf("Expected %d colors, got %d", expectedNumColors, len(palette))
	}

	// Validate each color is a hex string
	for i, color := range palette {
		colorStr, ok := color.(string)
		if !ok {
			t.Errorf("Color at index %d is not a string", i)
			continue
		}
		if len(colorStr) != 7 || colorStr[0] != '#' {
			t.Errorf("Color at index %d has invalid hex format: %s", i, colorStr)
		}
	}

	return palette
}

// assertCacheHeader validates cache hit/miss headers
func assertCacheHeader(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	t.Helper()

	cacheHeader := w.Header().Get("X-Cache")
	if cacheHeader != expected {
		t.Errorf("Expected X-Cache header '%s', got '%s'", expected, cacheHeader)
	}
}

// generateTestPaletteRequest creates a test palette request
func generateTestPaletteRequest(theme, style string, numColors int, baseColor string) PaletteRequest {
	return PaletteRequest{
		Theme:     theme,
		Style:     style,
		NumColors: numColors,
		BaseColor: baseColor,
	}
}

// generateTestAccessibilityRequest creates a test accessibility request
func generateTestAccessibilityRequest(fg, bg string) AccessibilityRequest {
	return AccessibilityRequest{
		Foreground: fg,
		Background: bg,
	}
}

// generateTestHarmonyRequest creates a test harmony request
func generateTestHarmonyRequest(colors []string) HarmonyRequest {
	return HarmonyRequest{
		Colors: colors,
	}
}

// generateTestColorblindRequest creates a test colorblind request
func generateTestColorblindRequest(colors []string, cbType string) ColorblindRequest {
	return ColorblindRequest{
		Colors: colors,
		Type:   cbType,
	}
}

// setupTestEnvironment initializes full test environment
func setupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Set lifecycle env var
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "18888")

	cleanupLogger := setupTestLogger()
	_, cleanupRedis := setupTestRedis(t)

	return func() {
		cleanupRedis()
		cleanupLogger()
	}
}

// measureExecutionTime measures function execution time
func measureExecutionTime(t *testing.T, name string, fn func()) time.Duration {
	t.Helper()

	start := time.Now()
	fn()
	duration := time.Since(start)

	t.Logf("%s completed in %v", name, duration)
	return duration
}

// assertExecutionTime validates execution time is within threshold
func assertExecutionTime(t *testing.T, duration time.Duration, maxDuration time.Duration, operation string) {
	t.Helper()

	if duration > maxDuration {
		t.Errorf("%s took %v, expected < %v", operation, duration, maxDuration)
	}
}
