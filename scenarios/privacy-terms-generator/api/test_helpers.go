package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	// Redirect log output to a test buffer
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "privacy-terms-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request) {
	var bodyReader *bytes.Reader

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

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	return w, httpReq
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	body := w.Body.String()
	if expectedMessage != "" && !strings.Contains(body, expectedMessage) {
		t.Errorf("Expected response to contain '%s', got '%s'", expectedMessage, body)
	}
}

// assertContentType validates the Content-Type header
func assertContentType(t *testing.T, w *httptest.ResponseRecorder, expectedType string) {
	contentType := w.Header().Get("Content-Type")
	if contentType != expectedType {
		t.Errorf("Expected Content-Type '%s', got '%s'", expectedType, contentType)
	}
}

// assertResponseContains validates that response contains specific text
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, expectedText string) {
	body := w.Body.String()
	if !strings.Contains(body, expectedText) {
		t.Errorf("Expected response to contain '%s', got '%s'", expectedText, body)
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// GenerateRequest creates a test generation request
func (g *TestDataGenerator) GenerateRequest(businessName, docType string, jurisdictions []string) GenerateRequest {
	return GenerateRequest{
		BusinessName:  businessName,
		DocumentType:  docType,
		Jurisdictions: jurisdictions,
		BusinessType:  "SaaS",
		Format:        "markdown",
	}
}

// GenerateRequestWithEmail creates a test generation request with email
func (g *TestDataGenerator) GenerateRequestWithEmail(businessName, docType, email string, jurisdictions []string) GenerateRequest {
	return GenerateRequest{
		BusinessName:  businessName,
		DocumentType:  docType,
		Jurisdictions: jurisdictions,
		Email:         email,
		Format:        "markdown",
	}
}

// SearchRequest creates a test search request
func (g *TestDataGenerator) SearchClauseRequest(query string, limit int) SearchRequest {
	return SearchRequest{
		Query: query,
		Limit: limit,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
