
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

// setupTestLogger initializes a test logger
func setupTestLogger() func() {
	originalFlags := log.Flags()
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return func() { log.SetFlags(originalFlags) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	DataDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "job-pipeline-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	testDataDir := filepath.Join(tempDir, "data")

	// Create state directories
	states := []string{"pending", "researching", "evaluated", "approved", "building", "completed", "archive"}
	for _, state := range states {
		stateDir := filepath.Join(testDataDir, state)
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create state dir %s: %v", state, err)
		}
	}

	// Override dataDir for tests
	oldDataDir := dataDir
	dataDir = testDataDir

	return &TestEnvironment{
		TempDir:    tempDir,
		DataDir:    testDataDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			dataDir = oldDataDir
			os.RemoveAll(tempDir)
		},
	}
}

// TestJob provides a pre-configured job for testing
type TestJob struct {
	Job     *Job
	Cleanup func()
}

// setupTestJob creates a test job with sample data
func setupTestJob(t *testing.T, state string) *TestJob {
	jobID := fmt.Sprintf("TEST-JOB-%s", uuid.New().String()[:8])

	job := &Job{
		ID:          jobID,
		Source:      "manual",
		Title:       "Test Job Title",
		Description: "This is a test job description for testing purposes.",
		State:       state,
		Budget: Budget{
			Min:      100.0,
			Max:      500.0,
			Currency: "USD",
		},
		SkillsRequired: []string{"Go", "Testing", "API Development"},
		Timeline:       "2 weeks",
		Metadata: Metadata{
			ImportedAt:      time.Now().UTC().Format(time.RFC3339),
			JobType:         "contract",
			ExperienceLevel: "intermediate",
		},
		History: []HistoryEntry{
			{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				State:     state,
				Action:    "created",
				Notes:     "Test job created",
			},
		},
	}

	if err := saveJob(job); err != nil {
		t.Fatalf("Failed to save test job: %v", err)
	}

	return &TestJob{
		Job: job,
		Cleanup: func() {
			deleteJobFile(jobID, state)
		},
	}
}

// setupTestJobWithResearch creates a test job with research report
func setupTestJobWithResearch(t *testing.T, state string) *TestJob {
	testJob := setupTestJob(t, state)

	testJob.Job.ResearchReport = &ResearchReport{
		ID:                uuid.New().String(),
		Evaluation:        "RECOMMENDED",
		FeasibilityScore:  0.85,
		ExistingScenarios: []string{"ecosystem-manager", "research-assistant"},
		RequiredScenarios: []string{"new-custom-scenario"},
		EstimatedHours:    40,
		TechnicalAnalysis: "This job is feasible with our existing capabilities.",
		CreatedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	if err := saveJob(testJob.Job); err != nil {
		t.Fatalf("Failed to save job with research: %v", err)
	}

	return testJob
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
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	body := strings.TrimSpace(w.Body.String())
	if body == "" {
		t.Error("Expected error message in response body")
	}
}

// deleteJobFile removes a job file from disk
func deleteJobFile(jobID, state string) {
	filename := filepath.Join(dataDir, state, jobID+".yaml")
	os.Remove(filename)
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// ImportRequest creates a test import request
func (g *TestDataGenerator) ImportRequest(source, data string) ImportRequest {
	return ImportRequest{
		Source: source,
		Data:   data,
	}
}

// ResearchRequest creates a test research request
func (g *TestDataGenerator) ResearchRequest(jobID string) ResearchRequest {
	return ResearchRequest{
		JobID: jobID,
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// Common test scenarios
type TestScenarios struct{}

// JobNotFound tests handlers with non-existent job IDs
func (s *TestScenarios) JobNotFound(t *testing.T, handler http.HandlerFunc, method, path string) {
	nonExistentID := "NON-EXISTENT-JOB-ID"
	req := HTTPTestRequest{
		Method:  method,
		Path:    strings.Replace(path, "{id}", nonExistentID, 1),
		URLVars: map[string]string{"id": nonExistentID},
	}

	w, httpReq := makeHTTPRequest(req)
	handler(w, httpReq)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent job, got %d", w.Code)
	}
}

// InvalidSource tests import handler with invalid source
func (s *TestScenarios) InvalidSource(t *testing.T, handler http.HandlerFunc) {
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/jobs/import",
		Body: ImportRequest{
			Source: "invalid-source",
			Data:   "test data",
		},
	}

	w, httpReq := makeHTTPRequest(req)
	handler(w, httpReq)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid source, got %d", w.Code)
	}
}

// Global test scenarios instance
var Scenarios = &TestScenarios{}
