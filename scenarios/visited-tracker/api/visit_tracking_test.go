package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestVisitHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-visit-test")
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

	// Create and save test campaign
	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-visit-campaign",
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

	// Test visit request
	visitReq := VisitRequest{
		Files:   []string{"test.go", "main.go"},
		Context: strPtr("test-visit"),
		Agent:   strPtr("unit-test"),
	}

	jsonBody, _ := json.Marshal(visitReq)
	req := httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if recorded, ok := response["recorded"].(float64); !ok || recorded != 2 {
		t.Errorf("Expected 2 visits recorded, got %v", response["recorded"])
	}

	// Verify campaign was updated
	updatedCampaign, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Errorf("Should be able to load updated campaign: %v", err)
	}

	if len(updatedCampaign.TrackedFiles) != 2 {
		t.Errorf("Expected 2 tracked files, got %d", len(updatedCampaign.TrackedFiles))
	}
	if len(updatedCampaign.Visits) != 2 {
		t.Errorf("Expected 2 visits, got %d", len(updatedCampaign.Visits))
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("POST", "/api/v1/campaigns/invalid-uuid/visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+nonExistentID.String()+"/visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test empty files
	emptyReq := VisitRequest{
		Files: []string{},
	}
	jsonBody, _ = json.Marshal(emptyReq)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty files, got %d", w.Code)
	}
}

func TestAdjustVisitHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-adjust-test")
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

	// Create test campaign with tracked file
	trackedFile := TrackedFile{
		ID:           uuid.New(),
		FilePath:     "test.go",
		AbsolutePath: "/tmp/test.go",
		VisitCount:   5,
		FirstSeen:    time.Now().UTC(),
		LastModified: time.Now().UTC(),
		SizeBytes:    100,
		Deleted:      false,
		Metadata:     make(map[string]interface{}),
	}

	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-adjust-campaign",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{"*.go"},
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{trackedFile},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test increment visit
	adjustReq := AdjustVisitRequest{
		FileID: trackedFile.ID.String(),
		Action: "increment",
	}

	jsonBody, _ := json.Marshal(adjustReq)
	req := httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if count, ok := response["visit_count"].(float64); !ok || count != 6 {
		t.Errorf("Expected visit count 6, got %v", response["visit_count"])
	}

	// Test decrement visit
	adjustReq.Action = "decrement"
	jsonBody, _ = json.Marshal(adjustReq)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test invalid action
	adjustReq.Action = "invalid"
	jsonBody, _ = json.Marshal(adjustReq)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid action, got %d", w.Code)
	}

	// Test invalid file ID
	adjustReq.Action = "increment"
	adjustReq.FileID = "invalid-uuid"
	jsonBody, _ = json.Marshal(adjustReq)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file ID, got %d", w.Code)
	}
}

func TestVisitHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-visit-error-test")
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
	description := "Test campaign for visit error testing"
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        "test-visit-error-campaign",
		Description: &description,
		CreatedAt:   time.Now(),
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}
	defer deleteCampaignFile(campaign.ID)

	// Test with invalid JSON
	invalidJSON := `{"file_paths": ["test.go"], "context": "`
	req := httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/visit", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	// Test with empty file paths
	emptyPaths := `{"file_paths": [], "context": "general"}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/visit", strings.NewReader(emptyPaths))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty file paths, got %d", w.Code)
	}

	// Test with missing file_paths field
	missingPaths := `{"context": "general"}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/visit", strings.NewReader(missingPaths))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	visitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing file_paths, got %d", w.Code)
	}
}

func TestAdjustVisitHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-adjust-visit-error-test")
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
	description := "Test campaign for adjust visit error testing"
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        "test-adjust-visit-error-campaign",
		Description: &description,
		CreatedAt:   time.Now(),
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}
	defer deleteCampaignFile(campaign.ID)

	// Test with invalid JSON (malformed JSON)
	invalidJSON := `{"file_id": "invalid", "action": "`
	req := httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	// Test with invalid file ID
	invalidFileID := `{"file_id": "invalid-uuid", "adjustment": 5}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", strings.NewReader(invalidFileID))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file ID, got %d", w.Code)
	}

	// Test with non-existent file ID (with correct JSON structure)
	nonExistentFileID := uuid.New()
	nonExistentFile := fmt.Sprintf(`{"file_id": "%s", "action": "increment"}`, nonExistentFileID.String())
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/adjust-visit", strings.NewReader(nonExistentFile))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	adjustVisitHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file ID, got %d", w.Code)
	}
}
