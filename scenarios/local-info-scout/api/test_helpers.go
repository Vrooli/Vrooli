package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Redirect log output to discard to keep test output clean
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	RedisClient  *redis.Client
	OriginalEnv  map[string]string
	Cleanup      func()
}

// setupTestEnvironment creates an isolated test environment with proper cleanup
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Save original environment variables
	originalEnv := map[string]string{
		"REDIS_HOST":      os.Getenv("REDIS_HOST"),
		"REDIS_PORT":      os.Getenv("REDIS_PORT"),
		"POSTGRES_HOST":   os.Getenv("POSTGRES_HOST"),
		"POSTGRES_PORT":   os.Getenv("POSTGRES_PORT"),
		"POSTGRES_USER":   os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_DB":     os.Getenv("POSTGRES_DB"),
		"OLLAMA_HOST":     os.Getenv("OLLAMA_HOST"),
		"SEARXNG_HOST":    os.Getenv("SEARXNG_HOST"),
	}

	// Set test environment variables
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("POSTGRES_DB", "local_info_scout_test")
	os.Setenv("OLLAMA_HOST", "http://localhost:11434")
	os.Setenv("SEARXNG_HOST", "http://localhost:8280")

	// Try to create a test Redis client
	redisAddr := "localhost:6379"
	testRedis := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "",
		DB:       15, // Use test database
	})

	return &TestEnvironment{
		RedisClient: testRedis,
		OriginalEnv: originalEnv,
		Cleanup: func() {
			// Clean up test Redis data
			if testRedis != nil {
				ctx := context.Background()
				if err := testRedis.FlushDB(ctx).Err(); err == nil {
					testRedis.Close()
				}
			}

			// Restore original environment
			for key, value := range originalEnv {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonData, _ := json.Marshal(v)
			bodyReader = bytes.NewBuffer(jsonData)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set Content-Type if not already set and body is present
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)
	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertJSONArrayResponse validates a JSON array response
func assertJSONArrayResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result []interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode JSON array response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	if expectedMessage != "" {
		body := w.Body.String()
		if body != expectedMessage && body != expectedMessage+"\n" {
			t.Errorf("Expected error message '%s', got '%s'", expectedMessage, body)
		}
	}
}

// assertPlacesResponse validates a places array response
func assertPlacesResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []Place {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var places []Place
	if err := json.NewDecoder(w.Body).Decode(&places); err != nil {
		t.Fatalf("Failed to decode places response: %v. Body: %s", err, w.Body.String())
	}

	return places
}

// assertStringArrayResponse validates a string array response
func assertStringArrayResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []string {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result []string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode string array response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertPlaceResponse validates a single place response
func assertPlaceResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) Place {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var place Place
	if err := json.NewDecoder(w.Body).Decode(&place); err != nil {
		t.Fatalf("Failed to decode place response: %v. Body: %s", err, w.Body.String())
	}

	return place
}

// assertHeaderPresent checks if a header is present
func assertHeaderPresent(t *testing.T, w *httptest.ResponseRecorder, headerName string) {
	t.Helper()

	if w.Header().Get(headerName) == "" {
		t.Errorf("Expected header '%s' to be present", headerName)
	}
}

// assertHeaderValue checks if a header has a specific value
func assertHeaderValue(t *testing.T, w *httptest.ResponseRecorder, headerName, expectedValue string) {
	t.Helper()

	actual := w.Header().Get(headerName)
	if actual != expectedValue {
		t.Errorf("Expected header '%s' to be '%s', got '%s'", headerName, expectedValue, actual)
	}
}

// createTestSearchRequest creates a test search request
func createTestSearchRequest(query string, lat, lon, radius float64) SearchRequest {
	return SearchRequest{
		Query:  query,
		Lat:    lat,
		Lon:    lon,
		Radius: radius,
	}
}

// createTestPlace creates a test place
func createTestPlace(id, name, category string, distance, rating float64) Place {
	return Place{
		ID:          id,
		Name:        name,
		Address:     "123 Test St",
		Category:    category,
		Distance:    distance,
		Rating:      rating,
		PriceLevel:  2,
		OpenNow:     true,
		Description: "Test place",
	}
}
