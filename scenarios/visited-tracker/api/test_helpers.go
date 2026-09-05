package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func setTestStorageRoot(t *testing.T, root string) {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", filepath.Join(root, "storage"))
}

func initTestStorageRoot(t *testing.T, root string) {
	t.Helper()
	if logger == nil {
		t.Cleanup(setupTestLogger())
	}
	setTestStorageRoot(t, root)
	if err := initFileStorage(); err != nil {
		t.Fatalf("Failed to init file storage: %v", err)
	}
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalLogger := logger
	logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
	return func() { logger = originalLogger }
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

	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

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

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// VisitRequest creates a test visit request
func (g *TestDataGenerator) VisitRequest(files []string) VisitRequest {
	return VisitRequest{Files: VisitFiles{Paths: files}}
}

// CreateCampaignRequest creates a test campaign creation request
func (g *TestDataGenerator) CreateCampaignRequest(name string, patterns []string) CreateCampaignRequest {
	description := fmt.Sprintf("Test campaign: %s", name)
	return CreateCampaignRequest{
		Name:        name,
		Description: &description,
		Patterns:    patterns,
	}
}

// AdjustVisitRequest creates a test adjust visit request
func (g *TestDataGenerator) AdjustVisitRequest(fileID uuid.UUID, action string) AdjustVisitRequest {
	return AdjustVisitRequest{
		FileID: fileID.String(),
		Action: action,
	}
}

// StructureSyncRequest creates a test structure sync request
func (g *TestDataGenerator) StructureSyncRequest(patterns []string) StructureSyncRequest {
	return StructureSyncRequest{
		Patterns: patterns,
	}
}

// TestData is the global test data generator.
var TestData = &TestDataGenerator{}

// TestScenarios provides common test scenarios.
type TestScenarios struct{}

// CampaignNotFound tests handlers with non-existent campaign IDs
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

// InvalidUUID tests handlers with invalid UUID formats
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

// Scenarios is the global test scenarios instance.
var Scenarios = &TestScenarios{}
