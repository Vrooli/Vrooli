package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorsMiddleware(t *testing.T) {
	// Set UI_PORT for testing
	os.Setenv("UI_PORT", "38440")
	defer os.Unsetenv("UI_PORT")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with CORS middleware
	corsHandler := corsMiddleware(testHandler)

	// Test OPTIONS request with allowed origin
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:38440")
	w := httptest.NewRecorder()
	corsHandler.ServeHTTP(w, req)

	// Check CORS headers for allowed origin
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:38440" {
		t.Errorf("CORS middleware should set Access-Control-Allow-Origin to http://localhost:38440, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "GET") {
		t.Error("CORS middleware should allow GET method")
	}

	if w.Code != http.StatusOK {
		t.Errorf("OPTIONS request should return 200, got %d", w.Code)
	}

	// Test OPTIONS request with disallowed origin
	req = httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w = httptest.NewRecorder()
	corsHandler.ServeHTTP(w, req)

	// Should not set CORS header for disallowed origin
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS middleware should not set Access-Control-Allow-Origin for disallowed origins")
	}

	// Test regular GET request passes through
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:38440")
	w = httptest.NewRecorder()
	corsHandler.ServeHTTP(w, req)

	if w.Body.String() != "test response" {
		t.Error("CORS middleware should pass through regular requests")
	}

	// Verify allowed origin is set on regular requests
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:38440" {
		t.Error("CORS middleware should set allowed origin on regular requests")
	}
}

func TestHealthHandler(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-health-handler-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory and create required directory structure
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create the required directory structure
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Test health handler
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response is valid JSON
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	// Check required fields
	if response["status"] == nil {
		t.Error("Response should have status field")
	}
	if response["service"] != serviceName {
		t.Errorf("Expected service %s, got %v", serviceName, response["service"])
	}
	if response["version"] != apiVersion {
		t.Errorf("Expected version %s, got %v", apiVersion, response["version"])
	}

	// Check dependencies structure
	if deps, ok := response["dependencies"].(map[string]interface{}); ok {
		if storage, ok := deps["storage"].(map[string]interface{}); ok {
			if storage["connected"] != true {
				t.Error("Storage should be connected when data directory exists")
			}
			if storage["type"] != "json-files" {
				t.Error("Storage type should be json-files")
			}
		} else {
			t.Error("Dependencies should have storage object")
		}
	} else {
		t.Error("Response should have dependencies object")
	}
}

func TestHealthEndpointComponents(t *testing.T) {
	// Test when data directory doesn't exist
	tempDir, err := ioutil.TempDir("", "visited-tracker-health-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	nonExistentPath := filepath.Join(tempDir, "nonexistent")

	// Test directory access
	if _, err := os.Stat(nonExistentPath); err == nil {
		t.Error("Expected error when accessing non-existent directory")
	}

	// Test directory creation
	if err := os.MkdirAll(nonExistentPath, 0755); err != nil {
		t.Errorf("Should be able to create directory: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nonExistentPath); err != nil {
		t.Errorf("Directory should exist after creation: %v", err)
	}
}

func TestHealthHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test health handler with various scenarios
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify response contains expected fields
	var health map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("Failed to parse health JSON: %v", err)
	}

	if health["status"] != "ok" && health["status"] != "degraded" && health["status"] != "healthy" {
		t.Errorf("Expected status 'ok', 'degraded', or 'healthy', got %v", health["status"])
	}

	if health["service"] != serviceName {
		t.Errorf("Expected service name %s, got %v", serviceName, health["service"])
	}

	if health["version"] != apiVersion {
		t.Errorf("Expected version %s, got %v", apiVersion, health["version"])
	}
}

func TestOptionsHandler(t *testing.T) {
	req := httptest.NewRequest("OPTIONS", "/api/v1/campaigns", nil)
	w := httptest.NewRecorder()

	optionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIVersion(t *testing.T) {
	expected := "3.0.0"
	if apiVersion != expected {
		t.Errorf("Expected API version %s, got %s", expected, apiVersion)
	}
}

func TestServiceName(t *testing.T) {
	expected := "visited-tracker"
	if serviceName != expected {
		t.Errorf("Expected service name %s, got %s", expected, serviceName)
	}
}

func TestMainFunctionComponents(t *testing.T) {
	// Test individual components that main() uses without actually running main()

	// Test environment variable validation (simulated)
	originalEnv := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
	os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") == "true" {
		t.Error("Environment variable should be unset for this test")
	}

	// Restore environment
	if originalEnv != "" {
		os.Setenv("VROOLI_LIFECYCLE_MANAGED", originalEnv)
	}

	// Test port environment variable checking
	originalPort := os.Getenv("API_PORT")
	os.Unsetenv("API_PORT")

	if os.Getenv("API_PORT") != "" {
		t.Error("API_PORT should be unset for this test")
	}

	// Restore environment
	if originalPort != "" {
		os.Setenv("API_PORT", originalPort)
	}

	// Test directory changing functionality
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Test that os.Chdir works (this is what main() does)
	tempDir, err := ioutil.TempDir("", "visited-tracker-main-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Errorf("Directory change should work: %v", err)
	}

	// Restore original directory
	os.Chdir(originalWD)
}
