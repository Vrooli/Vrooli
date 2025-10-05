// +build testing

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Suppress logs during tests unless TEST_VERBOSE is set
	if os.Getenv("TEST_VERBOSE") != "true" {
		log.SetOutput(io.Discard)
		return func() {
			log.SetOutput(os.Stdout)
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server  *Server
	Cleanup func()
}

// setupTestServer creates an isolated test server
func setupTestServer(t *testing.T) *TestEnvironment {
	// Set environment variables for testing
	os.Setenv("API_PORT", "0") // Random port
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")

	server := NewServer()

	return &TestEnvironment{
		Server: server,
		Cleanup: func() {
			if server.db != nil {
				server.db.Close()
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(server *Server, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("failed to marshal request body: %v", err))
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	for key, expectedValue := range expectedFields {
		actualValue, exists := response[key]
		if !exists {
			t.Errorf("Expected field '%s' not found in response", key)
			continue
		}

		if expectedValue != nil && actualValue != expectedValue {
			t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
		}
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// Error responses might be plain text from http.Error
	body := strings.TrimSpace(w.Body.String())
	if body == "" {
		t.Error("Expected error message in response body")
	}
}

// assertHealthResponse validates health check responses
func assertHealthResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus string) HealthResponse {
	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse health response: %v", err)
	}

	if response.Status != expectedStatus {
		t.Errorf("Expected health status %s, got %s", expectedStatus, response.Status)
	}

	if response.Service != "smart-shopping-assistant" {
		t.Errorf("Expected service name 'smart-shopping-assistant', got %s", response.Service)
	}

	return response
}

// mockAuthContext creates a context with authenticated user for testing
func mockAuthContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ShoppingResearchRequest creates a test shopping research request
func (g *TestDataGenerator) ShoppingResearchRequest(query string, budgetMax float64) ShoppingResearchRequest {
	return ShoppingResearchRequest{
		ProfileID:           "test-user-1",
		Query:               query,
		BudgetMax:           budgetMax,
		IncludeAlternatives: true,
	}
}

// PatternAnalysisRequest creates a test pattern analysis request
func (g *TestDataGenerator) PatternAnalysisRequest(profileID string) PatternAnalysisRequest {
	return PatternAnalysisRequest{
		ProfileID: profileID,
		Timeframe: "30d",
	}
}

// PriceAlertRequest creates a test price alert request
func (g *TestDataGenerator) PriceAlertRequest(productID string, targetPrice float64) map[string]interface{} {
	return map[string]interface{}{
		"profile_id":  "test-user-1",
		"product_id":  productID,
		"target_price": targetPrice,
		"alert_type":  "below_target",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
