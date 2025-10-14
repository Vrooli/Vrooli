package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	// Redirect log output to avoid test noise
	oldOutput := log.Writer()
	log.SetOutput(io.Discard)

	return func() {
		log.SetOutput(oldOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	BackupDir  string
	OriginalWD string
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := os.MkdirTemp("", "data-backup-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create backup directory structure
	backupDir := filepath.Join(tempDir, "data", "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create backup dir: %v", err)
	}

	// Create subdirectories
	for _, dir := range []string{"postgres", "files", "minio"} {
		if err := os.MkdirAll(filepath.Join(backupDir, dir), 0755); err != nil {
			os.RemoveAll(tempDir)
			t.Fatalf("Failed to create %s dir: %v", dir, err)
		}
	}

	// Change to temp directory for tests
	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		BackupDir:  backupDir,
		OriginalWD: originalWD,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// makeHTTPRequest creates a test HTTP request with JSON body
func makeHTTPRequest(method, url string, body interface{}) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			panic(fmt.Sprintf("Failed to marshal request body: %v", err))
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req := httptest.NewRequest(method, url, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// assertJSONResponse validates the JSON response from a handler
func assertJSONResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, validate func(map[string]interface{}) bool) map[string]interface{} {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode JSON response: %v. Body: %s", err, rr.Body.String())
		return nil
	}

	if validate != nil && !validate(response) {
		t.Errorf("Response validation failed. Response: %+v", response)
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if rr.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, rr.Code, rr.Body.String())
		return
	}

	bodyStr := rr.Body.String()
	if expectedError != "" && bodyStr != expectedError+"\n" {
		t.Errorf("Expected error %q, got %q", expectedError, bodyStr)
	}
}

// createMockBackupFile creates a mock backup file for testing
func createMockBackupFile(t *testing.T, backupDir, jobID, backupType string) string {
	t.Helper()

	timestamp := time.Now().Format("20060102_150405")
	var filename string
	var subdir string

	switch backupType {
	case "postgres":
		subdir = "postgres"
		filename = fmt.Sprintf("%s_%s.sql.gz", jobID, timestamp)
	case "files":
		subdir = "files"
		filename = fmt.Sprintf("%s_%s.tar.gz", jobID, timestamp)
	case "minio":
		subdir = "minio"
		filename = fmt.Sprintf("%s_%s.tar.gz", jobID, timestamp)
	default:
		t.Fatalf("Unknown backup type: %s", backupType)
	}

	filePath := filepath.Join(backupDir, subdir, filename)
	if err := os.WriteFile(filePath, []byte("mock backup data"), 0644); err != nil {
		t.Fatalf("Failed to create mock backup file: %v", err)
	}

	return filePath
}

// assertFileExists checks if a file exists
func assertFileExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Expected file to exist: %s", path)
	}
}

// assertFileNotExists checks if a file does not exist
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Errorf("Expected file to not exist: %s", path)
	}
}

// setTestEnv sets environment variables for testing and returns cleanup function
func setTestEnv(t *testing.T, envVars map[string]string) func() {
	t.Helper()

	originalEnv := make(map[string]string)

	for key, value := range envVars {
		originalEnv[key] = os.Getenv(key)
		if value == "" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, value)
		}
	}

	return func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}
}

// createTestRouter creates a mux router with all API routes for testing
func createTestRouter() *mux.Router {
	r := mux.NewRouter()

	// Health endpoint
	r.HandleFunc("/health", handleHealth).Methods("GET")

	// API v1 routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Backup endpoints
	api.HandleFunc("/backup/create", handleBackupCreate).Methods("POST")
	api.HandleFunc("/backup/status", handleBackupStatus).Methods("GET")
	api.HandleFunc("/backup/list", handleBackupList).Methods("GET")
	api.HandleFunc("/backup/verify/{id}", handleBackupVerify).Methods("POST")

	// Restore endpoints
	api.HandleFunc("/restore/create", handleRestoreCreate).Methods("POST")
	api.HandleFunc("/restore/status/{id}", handleRestoreStatus).Methods("GET")

	// Schedule endpoints
	api.HandleFunc("/schedules", handleScheduleList).Methods("GET")
	api.HandleFunc("/schedules", handleScheduleCreate).Methods("POST")
	api.HandleFunc("/schedules/{id}", handleScheduleUpdate).Methods("PUT")
	api.HandleFunc("/schedules/{id}", handleScheduleDelete).Methods("DELETE")

	// Compliance endpoints
	api.HandleFunc("/compliance/report", handleComplianceReport).Methods("GET")
	api.HandleFunc("/compliance/scan", handleComplianceScan).Methods("POST")
	api.HandleFunc("/compliance/issue/{id}/fix", handleComplianceFix).Methods("POST")

	// Visited tracker integration
	api.HandleFunc("/visited/record", handleVisitedRecord).Methods("POST")
	api.HandleFunc("/visited/next", handleVisitedNext).Methods("GET")

	// Maintenance orchestrator integration
	api.HandleFunc("/maintenance/status", handleMaintenanceStatus).Methods("GET")
	api.HandleFunc("/maintenance/task", handleMaintenanceTask).Methods("POST")
	api.HandleFunc("/maintenance/agent/toggle", handleMaintenanceAgentToggle).Methods("POST")

	return r
}

// executeRequest executes a request against the router and returns the response recorder
func executeRequest(router *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
