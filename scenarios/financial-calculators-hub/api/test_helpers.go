package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes quiet test logging
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// HTTPTestRequest defines a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(req HTTPTestRequest, handler http.HandlerFunc) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader

	if req.Body != nil {
		if bodyStr, ok := req.Body.(string); ok {
			bodyReader = bytes.NewBufferString(bodyStr)
		} else {
			bodyBytes, err := json.Marshal(req.Body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewBuffer(bodyBytes)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)

	return w, nil
}

// assertJSONResponse validates that a response is valid JSON with expected status
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" && w.Body.Len() > 0 {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	if w.Body.Len() > 0 {
		var result map[string]interface{}
		bodyBytes := w.Body.Bytes()
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			t.Errorf("Failed to decode JSON response: %v", err)
			t.Logf("Response body: %s", string(bodyBytes))
		}
	}
}

// assertErrorResponse validates that a response contains an error message
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
		t.Logf("Response body: %s", w.Body.String())
		return
	}

	// Only check for error field if there's a body
	if w.Body.Len() > 0 {
		var result map[string]interface{}
		bodyBytes := w.Body.Bytes()
		if err := json.Unmarshal(bodyBytes, &result); err == nil {
			if _, ok := result["error"]; !ok {
				t.Error("Expected error field in response")
			}
		}
	}
}

// decodeJSONResponse decodes a JSON response into the provided interface
func decodeJSONResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()

	// Create a new reader from the body bytes to avoid EOF issues
	bodyBytes := w.Body.Bytes()
	if err := json.Unmarshal(bodyBytes, v); err != nil {
		t.Fatalf("Failed to decode JSON response: %v\nBody: %s", err, string(bodyBytes))
	}
}

// setupTestEnvironment prepares environment for testing
func setupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Set required environment variables
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "8080")

	// Disable database for most tests (unless explicitly testing DB features)
	originalDB := db
	db = nil

	return func() {
		db = originalDB
	}
}

// assertFloatEqual checks if two floats are approximately equal
func assertFloatEqual(t *testing.T, actual, expected, delta float64, field string) {
	t.Helper()

	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}

	if diff > delta {
		t.Errorf("%s: expected %v, got %v (delta: %v)", field, expected, actual, delta)
	}
}
