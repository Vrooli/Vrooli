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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestCoverageHandler verifies campaign file coverage statistics calculation
// [REQ:VT-REQ-002] [REQ:VT-REQ-006]
func TestCoverageHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-coverage-test")
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

	// Create test campaign with some tracked files
	now := time.Now()
	trackedFiles := []TrackedFile{
		{
			ID:             uuid.New(),
			FilePath:       "visited.go",
			VisitCount:     3,
			LastVisited:    &now,
			LastModified:   now.Add(-1 * time.Hour),
			StalenessScore: 10.0,
		},
		{
			ID:             uuid.New(),
			FilePath:       "unvisited.go",
			VisitCount:     0,
			LastVisited:    nil,
			LastModified:   now.Add(-2 * time.Hour),
			StalenessScore: 20.0,
		},
	}

	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-coverage-campaign",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{"*.go"},
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       trackedFiles,
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test coverage handler
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/coverage", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	coverageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	// Check expected fields
	expectedFields := []string{"total_files", "visited_files", "unvisited_files", "coverage_percentage", "average_visits", "average_staleness"}
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Response should have field: %s", field)
		}
	}

	// Check calculated values
	if totalFiles, ok := response["total_files"].(float64); !ok || totalFiles != 2 {
		t.Errorf("Expected 2 total files, got %v", response["total_files"])
	}
	if visitedFiles, ok := response["visited_files"].(float64); !ok || visitedFiles != 1 {
		t.Errorf("Expected 1 visited file, got %v", response["visited_files"])
	}
	if unvisitedFiles, ok := response["unvisited_files"].(float64); !ok || unvisitedFiles != 1 {
		t.Errorf("Expected 1 unvisited file, got %v", response["unvisited_files"])
	}
	if coveragePercentage, ok := response["coverage_percentage"].(float64); !ok || coveragePercentage != 50.0 {
		t.Errorf("Expected 50%% coverage, got %v", response["coverage_percentage"])
	}
}

// TestExportHandler verifies campaign data export functionality
// [REQ:VT-REQ-010]
func TestExportHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-export-test")
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

	// Create test campaign
	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-export-campaign",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{"*.go"},
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test export handler
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response is the campaign data
	var response Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid campaign JSON: %v", err)
	}

	if response.ID != campaign.ID {
		t.Errorf("Expected campaign ID %s, got %s", campaign.ID, response.ID)
	}
	if response.Name != campaign.Name {
		t.Errorf("Expected campaign name %s, got %s", campaign.Name, response.Name)
	}
}

// TestExportHandlerComprehensive verifies comprehensive campaign export scenarios
// including filters, formats, and error handling
// [REQ:VT-REQ-010]
func TestExportHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-export-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWD, _ := os.Getwd()
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	if err := initFileStorage(); err != nil {
		t.Fatalf("Failed to init file storage: %v", err)
	}

	// Create test campaign
	description := "Test campaign for export testing"
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        "test-export-campaign",
		Description: &description,
		CreatedAt:   time.Now(),
	}

	// Add tracked files
	trackedFiles := []TrackedFile{
		{
			ID:           uuid.New(),
			FilePath:     "test1.go",
			VisitCount:   5,
			LastModified: time.Now().Add(-2 * time.Hour),
			Deleted:      false,
		},
		{
			ID:           uuid.New(),
			FilePath:     "test2.js",
			VisitCount:   3,
			LastModified: time.Now().Add(-1 * time.Hour),
			Deleted:      false,
		},
	}

	campaign.TrackedFiles = trackedFiles

	// Save campaign
	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}
	defer deleteCampaignFile(campaign.ID)

	// Test successful export with default parameters
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	// Verify JSON response structure
	var exportData map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &exportData); err != nil {
		t.Fatalf("Failed to parse export JSON: %v", err)
	}

	// Should contain campaign metadata (correct field name from JSON tag)
	if exportData["name"] != campaign.Name {
		t.Errorf("Expected campaign name %s, got %v", campaign.Name, exportData["name"])
	}

	// Should contain tracked files (correct field name from JSON tag)
	files, ok := exportData["tracked_files"].([]interface{})
	if !ok {
		t.Errorf("Expected tracked_files array in export data")
	}

	if len(files) != len(trackedFiles) {
		t.Errorf("Expected %d files in export, got %d", len(trackedFiles), len(files))
	}

	// Test export with include_history parameter
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/export?include_history=true", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for include_history=true, got %d", w.Code)
	}

	// Test export with format parameter
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/export?format=detailed", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for format=detailed, got %d", w.Code)
	}

	// Test export with patterns filter
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/export?patterns=*.go", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for patterns filter, got %d", w.Code)
	}

	// Verify patterns filter worked
	if err := json.Unmarshal(w.Body.Bytes(), &exportData); err != nil {
		t.Fatalf("Failed to parse filtered export JSON: %v", err)
	}

	filteredFiles, ok := exportData["tracked_files"].([]interface{})
	if !ok {
		t.Errorf("Expected tracked_files array in filtered export data")
	}

	// Should only include .go files
	for _, fileInterface := range filteredFiles {
		if file, ok := fileInterface.(map[string]interface{}); ok {
			if filePath, ok := file["file_path"].(string); ok {
				if !strings.HasSuffix(filePath, ".go") {
					t.Errorf("Patterns filter failed: got non-.go file %s", filePath)
				}
			}
		}
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String()+"/export", nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	exportHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}
}

// TestCoverageHandlerErrorPaths verifies coverage handler error handling and edge cases
// [REQ:VT-REQ-002] [REQ:VT-REQ-006]
func TestCoverageHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-coverage-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWD, _ := os.Getwd()
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	if err := initFileStorage(); err != nil {
		t.Fatalf("Failed to init file storage: %v", err)
	}

	// Create test campaign
	description := "Test campaign for coverage error testing"
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        "test-coverage-campaign",
		Description: &description,
		CreatedAt:   time.Now(),
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}
	defer deleteCampaignFile(campaign.ID)

	// Test coverage with group-by parameter
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/coverage?group_by=extension", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	coverageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for group_by extension, got %d", w.Code)
	}

	// Test coverage with patterns parameter
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/coverage?patterns=*.go,*.js", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	coverageHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for patterns filter, got %d", w.Code)
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid/coverage", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	coverageHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String()+"/coverage", nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	coverageHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}
}
