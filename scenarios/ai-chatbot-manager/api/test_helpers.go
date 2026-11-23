package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// Global logger variable
var logger *Logger

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	if logger == nil {
		logger = NewLogger()
	}
	originalLogger := logger
	logger = NewLogger()
	logger.SetOutput(io.Discard) // Silence logs during tests
	return func() { logger = originalLogger }
}

// TestDatabase manages test database connection
type TestDatabase struct {
	db      *Database
	Cleanup func()
}

// TestChatbot provides a pre-configured chatbot for testing
type TestChatbot struct {
	Chatbot *Chatbot
	Cleanup func()
}

// TestTenant provides a pre-configured tenant for testing
type TestTenant struct {
	Tenant  *Tenant
	Cleanup func()
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
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
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
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
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
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
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
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected error field in response")
		return
	}

	if expectedErrorMessage != "" && !strings.Contains(errorMsg.(string), expectedErrorMessage) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMessage, errorMsg)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ChatRequest creates a test chat request
func (g *TestDataGenerator) ChatRequest(message, sessionID string) ChatRequest {
	return ChatRequest{
		Message:   message,
		SessionID: sessionID,
		Context:   map[string]interface{}{},
	}
}

// CreateChatbotRequest creates a test chatbot creation request
func (g *TestDataGenerator) CreateChatbotRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":           name,
		"description":    fmt.Sprintf("Test chatbot: %s", name),
		"personality":    "friendly and helpful",
		"knowledge_base": "Test knowledge base",
		"model_config": map[string]interface{}{
			"model":       "llama2",
			"temperature": 0.7,
		},
		"is_active": true,
	}
}

// CreateTenantRequest creates a test tenant creation request
func (g *TestDataGenerator) CreateTenantRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"slug":        strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		"description": fmt.Sprintf("Test tenant: %s", name),
		"plan":        "pro",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
