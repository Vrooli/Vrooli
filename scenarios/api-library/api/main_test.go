package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_DB", "test_api_library")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("QDRANT_HOST", "localhost")
	os.Setenv("QDRANT_PORT", "6333")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal("Failed to decode response body")
	}

	if response["status"] != "healthy" {
		t.Errorf("handler returned wrong status: got %v want %v",
			response["status"], "healthy")
	}

	if response["service"] != "api-library" {
		t.Errorf("handler returned wrong service name: got %v want %v",
			response["service"], "api-library")
	}
}
