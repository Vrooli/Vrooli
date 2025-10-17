// +build testing

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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := logger
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() { logger = originalLogger }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "visited-tracker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	if err := initFileStorage(); err != nil {
		os.Chdir(originalWD)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to init file storage: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// TestCampaign provides a pre-configured campaign for testing
type TestCampaign struct {
	Campaign     *Campaign
	TrackedFiles []TrackedFile
	Cleanup      func()
}

// setupTestCampaign creates a test campaign with sample data
func setupTestCampaign(t *testing.T, name string, patterns []string) *TestCampaign {
	if patterns == nil {
		patterns = []string{"*.go"}
	}

	description := fmt.Sprintf("Test campaign: %s", name)
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        name,
		Description: &description,
		Patterns:    patterns,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      "active",
		Metadata:    make(map[string]interface{}),
	}

	// Create some sample tracked files
	now := time.Now()
	trackedFiles := []TrackedFile{
		{
			ID:           uuid.New(),
			FilePath:     "test1.go",
			VisitCount:   5,
			LastModified: now.Add(-2 * time.Hour),
			LastVisited:  &now,
			Deleted:      false,
		},
		{
			ID:           uuid.New(),
			FilePath:     "test2.go",
			VisitCount:   3,
			LastModified: now.Add(-1 * time.Hour),
			LastVisited:  nil,
			Deleted:      false,
		},
		{
			ID:           uuid.New(),
			FilePath:     "test3.js",
			VisitCount:   1,
			LastModified: now.Add(-30 * time.Minute),
			LastVisited:  nil,
			Deleted:      false,
		},
	}

	campaign.TrackedFiles = trackedFiles

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	return &TestCampaign{
		Campaign:     campaign,
		TrackedFiles: trackedFiles,
		Cleanup: func() {
			deleteCampaignFile(campaign.ID)
		},
	}
}

// setupTestCampaignWithFiles creates a test campaign and actual files on disk
func setupTestCampaignWithFiles(t *testing.T, name string, files map[string]string) *TestCampaign {
	// Create the files on disk
	for filename, content := range files {
		if err := ioutil.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Determine patterns from file extensions
	patterns := []string{}
	extMap := make(map[string]bool)
	for filename := range files {
		ext := filepath.Ext(filename)
		if ext != "" && !extMap[ext] {
			patterns = append(patterns, "*"+ext)
			extMap[ext] = true
		}
	}

	testCampaign := setupTestCampaign(t, name, patterns)

	// Update cleanup to remove files
	originalCleanup := testCampaign.Cleanup
	testCampaign.Cleanup = func() {
		for filename := range files {
			os.Remove(filename)
		}
		originalCleanup()
	}

	return testCampaign
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method     string
	Path       string
	Body       interface{}
	URLVars    map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
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
				return nil, fmt.Errorf("failed to marshal request body: %v", err)
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
	return w, nil
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

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
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

// testHandlerWithRequest is a helper for testing handlers with specific requests
func testHandlerWithRequest(t *testing.T, handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, nil)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	handler(w, httpReq)
	return w
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// generateVisitRequest creates a test visit request
func (g *TestDataGenerator) VisitRequest(files []string) VisitRequest {
	return VisitRequest{Files: files}
}

// generateCreateCampaignRequest creates a test campaign creation request
func (g *TestDataGenerator) CreateCampaignRequest(name string, patterns []string) CreateCampaignRequest {
	description := fmt.Sprintf("Test campaign: %s", name)
	return CreateCampaignRequest{
		Name:        name,
		Description: &description,
		Patterns:    patterns,
	}
}

// generateAdjustVisitRequest creates a test adjust visit request
func (g *TestDataGenerator) AdjustVisitRequest(fileID uuid.UUID, action string) AdjustVisitRequest {
	return AdjustVisitRequest{
		FileID: fileID.String(),
		Action: action,
	}
}

// generateStructureSyncRequest creates a test structure sync request
func (g *TestDataGenerator) StructureSyncRequest(patterns []string) StructureSyncRequest {
	return StructureSyncRequest{
		Patterns: patterns,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// Common test scenarios
type TestScenarios struct{}

// testCampaignNotFound tests handlers with non-existent campaign IDs
func (s *TestScenarios) CampaignNotFound(t *testing.T, handler http.HandlerFunc, method, path string) {
	nonExistentID := uuid.New()
	req := HTTPTestRequest{
		Method:  method,
		Path:    strings.Replace(path, "{id}", nonExistentID.String(), 1),
		URLVars: map[string]string{"id": nonExistentID.String()},
	}

	w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, nil)
	httpReq = mux.SetURLVars(httpReq, req.URLVars)

	handler(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}
}

// testInvalidUUID tests handlers with invalid UUID formats
func (s *TestScenarios) InvalidUUID(t *testing.T, handler http.HandlerFunc, method, path string) {
	req := HTTPTestRequest{
		Method:  method,
		Path:    strings.Replace(path, "{id}", "invalid-uuid", 1),
		URLVars: map[string]string{"id": "invalid-uuid"},
	}

	w, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, nil)
	httpReq = mux.SetURLVars(httpReq, req.URLVars)

	handler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}
}

// Global test scenarios instance
var Scenarios = &TestScenarios{}