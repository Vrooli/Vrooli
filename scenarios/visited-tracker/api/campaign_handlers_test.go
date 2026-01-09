package main

import (
	"bytes"
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

func TestListCampaignsHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-list-test")
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

	// Test empty list
	req := httptest.NewRequest("GET", "/api/v1/campaigns", nil)
	w := httptest.NewRecorder()

	listCampaignsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
		t.Logf("Response body: %s", w.Body.String())
	}

	// Check campaigns field - can be null or empty array
	if campaignsValue, exists := response["campaigns"]; !exists {
		t.Error("Response should have campaigns field")
	} else {
		// Handle both nil (null in JSON) and empty array cases
		if campaignsValue == nil {
			// This is the expected behavior for empty results
		} else if campaigns, ok := campaignsValue.([]interface{}); ok {
			if len(campaigns) != 0 {
				t.Errorf("Empty list should return empty campaigns array, got %d campaigns", len(campaigns))
			}
		} else {
			t.Errorf("Campaigns field should be null or array, got %T: %v", campaignsValue, campaignsValue)
		}
	}

	if count, ok := response["count"].(float64); !ok || count != 0 {
		t.Error("Empty list should return count 0")
	}

	// Test with saved campaign that has tracked files (to trigger computation loop)
	now := time.Now()
	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-list-campaign",
		FromAgent: "unit-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    "active",
		Patterns:  []string{"*.go"},
		Metadata:  make(map[string]interface{}),
		TrackedFiles: []TrackedFile{
			{
				ID:           uuid.New(),
				FilePath:     "test.go",
				VisitCount:   2,
				LastVisited:  &now,
				LastModified: now.Add(-1 * time.Hour),
				Deleted:      false,
			},
			{
				ID:           uuid.New(),
				FilePath:     "unvisited.go",
				VisitCount:   0,
				LastModified: now.Add(-2 * time.Hour),
				Deleted:      false,
			},
		},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test list with campaign
	w = httptest.NewRecorder()
	listCampaignsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if campaigns, ok := response["campaigns"].([]interface{}); !ok || len(campaigns) != 1 {
		t.Error("List should return one campaign")
	}

	if count, ok := response["count"].(float64); !ok || count != 1 {
		t.Error("List should return count 1")
	}
}

func TestCreateCampaignHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-create-test")
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

	// Test valid campaign creation
	requestBody := CreateCampaignRequest{
		Name:      "test-create-campaign",
		FromAgent: "unit-test",
		Patterns:  []string{"*.go"},
		Metadata:  map[string]interface{}{"test": "value"},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	// Check response structure
	var campaign Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &campaign); err != nil {
		t.Errorf("Response should be valid campaign JSON: %v", err)
	}

	if campaign.Name != requestBody.Name {
		t.Errorf("Expected name %s, got %s", requestBody.Name, campaign.Name)
	}
	if campaign.FromAgent != requestBody.FromAgent {
		t.Errorf("Expected from_agent %s, got %s", requestBody.FromAgent, campaign.FromAgent)
	}
	if len(campaign.Patterns) != len(requestBody.Patterns) {
		t.Errorf("Expected %d patterns, got %d", len(requestBody.Patterns), len(campaign.Patterns))
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/api/v1/campaigns", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	// Test missing name
	invalidRequest := CreateCampaignRequest{
		FromAgent: "unit-test",
		Patterns:  []string{"*.go"},
	}

	jsonBody, _ = json.Marshal(invalidRequest)
	req = httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing name, got %d", w.Code)
	}

	// Test missing patterns
	invalidRequest = CreateCampaignRequest{
		Name:      "test-campaign",
		FromAgent: "unit-test",
		Patterns:  []string{},
	}

	jsonBody, _ = json.Marshal(invalidRequest)
	req = httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing patterns, got %d", w.Code)
	}

	// Test duplicate name (create same campaign again)
	jsonBody, _ = json.Marshal(requestBody)
	req = httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate name, got %d", w.Code)
	}
}

func TestGetCampaignHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-get-test")
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
		Name:               "test-get-campaign",
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

	// Test getting existing campaign
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response
	var response Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid campaign JSON: %v", err)
	}

	if response.ID != campaign.ID {
		t.Errorf("Expected ID %s, got %s", campaign.ID, response.ID)
	}
	if response.Name != campaign.Name {
		t.Errorf("Expected name %s, got %s", campaign.Name, response.Name)
	}

	// Test invalid UUID
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}
}

func TestDeleteCampaignHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-delete-handler-test")
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
		Name:               "test-delete-handler-campaign",
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

	// Test deleting existing campaign
	req := httptest.NewRequest("DELETE", "/api/v1/campaigns/"+campaign.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if response["deleted"] != true {
		t.Error("Response should indicate campaign was deleted")
	}

	// Verify campaign was actually deleted
	if _, err := loadCampaign(campaign.ID); err == nil {
		t.Error("Campaign should be deleted from storage")
	}

	// Test invalid UUID
	req = httptest.NewRequest("DELETE", "/api/v1/campaigns/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test deleting non-existent campaign (should be idempotent - return 200)
	nonExistentID := uuid.New()
	req = httptest.NewRequest("DELETE", "/api/v1/campaigns/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for idempotent delete of non-existent campaign, got %d", w.Code)
	}
}

func TestCreateCampaignHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test with invalid JSON
	invalidJSON := `{"name": "test", "description": "`
	req := httptest.NewRequest("POST", "/api/v1/campaigns", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	// Test with missing required fields
	emptyRequest := `{}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns", strings.NewReader(emptyRequest))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing fields, got %d", w.Code)
	}

	// Test with empty name
	emptyName := `{"name": "", "description": "test"}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns", strings.NewReader(emptyName))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty name, got %d", w.Code)
	}

	// Test with empty patterns array [REQ:VT-REQ-001]
	emptyPatterns := `{"name": "test-campaign", "patterns": []}`
	req = httptest.NewRequest("POST", "/api/v1/campaigns", strings.NewReader(emptyPatterns))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty patterns, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "At least one file pattern is required") {
		t.Errorf("Expected error message about patterns, got: %s", w.Body.String())
	}
}

func TestGetCampaignHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test missing campaign ID in URL vars
	req := httptest.NewRequest("GET", "/api/v1/campaigns/missing", nil)
	// Don't set URL vars to trigger the missing ID path
	w := httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing campaign ID, got %d", w.Code)
	}

	// Test invalid UUID format
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}
}

func TestDeleteCampaignHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-delete-test")
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

	// Test missing campaign ID in URL vars
	req := httptest.NewRequest("DELETE", "/api/v1/campaigns/missing", nil)
	w := httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing campaign ID, got %d", w.Code)
	}

	// Test invalid UUID format
	req = httptest.NewRequest("DELETE", "/api/v1/campaigns/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign (should succeed - idempotent delete)
	nonExistentID := uuid.New()
	req = httptest.NewRequest("DELETE", "/api/v1/campaigns/"+nonExistentID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for non-existent campaign delete (idempotent), got %d", w.Code)
	}
}

func TestGetCampaignHandlerWithDeletedFiles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-deleted-test")
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

	now := time.Now()
	visited := now.Add(-1 * time.Hour)
	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-deleted-files",
		Patterns:  []string{"*.go"},
		CreatedAt: now,
		UpdatedAt: now,
		Status:    "active",
		Metadata:  make(map[string]interface{}),
		TrackedFiles: []TrackedFile{
			{
				ID:           uuid.New(),
				FilePath:     "active.go",
				VisitCount:   5,
				LastModified: now,
				LastVisited:  &visited,
				Deleted:      false,
			},
			{
				ID:           uuid.New(),
				FilePath:     "deleted.go",
				VisitCount:   3,
				LastModified: now,
				LastVisited:  &visited,
				Deleted:      true,
			},
			{
				ID:           uuid.New(),
				FilePath:     "never-visited.go",
				VisitCount:   0,
				LastModified: now,
				LastVisited:  nil,
				Deleted:      false,
			},
		},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	getCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.VisitedFiles != 1 {
		t.Errorf("Expected 1 visited file (excluding deleted), got %d", response.VisitedFiles)
	}

	if response.TotalFiles != 3 {
		t.Errorf("Expected 3 total files, got %d", response.TotalFiles)
	}
}

func TestCreateCampaignHandlerMetadataInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-metadata-test")
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

	reqBody := CreateCampaignRequest{
		Name:      "test-no-metadata",
		FromAgent: "test-agent",
		Patterns:  []string{"*.go"},
		Metadata:  nil,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
	}

	var response Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Metadata == nil {
		t.Error("Expected metadata to be initialized, got nil")
	}
}

func TestCreateCampaignHandlerAutoSyncTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-autosync-test")
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

	reqBody := CreateCampaignRequest{
		Name:      "test-autosync",
		FromAgent: "test-agent",
		Patterns:  []string{"*.nonexistent"},
		Metadata:  map[string]interface{}{"test": "value"},
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/campaigns", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	createCampaignHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
	}

	var response Campaign
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, exists := response.Metadata["auto_sync_attempted"]; !exists {
		t.Error("Expected auto_sync_attempted metadata field")
	}
}

func TestDeleteCampaignHandlerIdempotent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-delete-test")
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

	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-delete-idempotent",
		Patterns:  []string{"*.go"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "active",
		Metadata:  make(map[string]interface{}),
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/v1/campaigns/"+campaign.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/campaigns/"+campaign.ID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	deleteCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for idempotent delete, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if deleted, ok := response["deleted"].(bool); !ok || !deleted {
		t.Error("Expected deleted=true in response")
	}
}

func TestFindOrCreateCampaignHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-find-or-create-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Test creating new campaign with location and tag
	location := "test-location"
	tag := "test-tag"
	requestBody := CreateCampaignRequest{
		Location: &location,
		Tag:      &tag,
		Patterns: []string{"*.go"},
	}

	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for new campaign, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if created, ok := response["created"].(bool); !ok || !created {
		t.Error("Expected created=true for new campaign")
	}

	campaign, ok := response["campaign"].(map[string]interface{})
	if !ok {
		t.Fatal("Response should have campaign object")
	}

	if name, ok := campaign["name"].(string); !ok || name != "test-location-test-tag" {
		t.Errorf("Expected generated name 'test-location-test-tag', got %v", campaign["name"])
	}

	// Test finding existing campaign with same location+tag
	req = httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for existing campaign, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Response should be valid JSON: %v", err)
	}

	if created, ok := response["created"].(bool); !ok || created {
		t.Error("Expected created=false for existing campaign")
	}

	// Test with explicit name but different location/tag (should create new)
	differentLocation := "different-location"
	differentTag := "different-tag"
	requestWithName := CreateCampaignRequest{
		Name:     "explicit-name",
		Location: &differentLocation,
		Tag:      &differentTag,
		Patterns: []string{"*.js"},
	}

	jsonBody, _ = json.Marshal(requestWithName)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201 for campaign with different location/tag, got %d", w.Code)
	}

	// Test missing patterns with different location/tag to avoid finding existing
	thirdLocation := "third-location"
	thirdTag := "third-tag"
	invalidRequest := CreateCampaignRequest{
		Location: &thirdLocation,
		Tag:      &thirdTag,
		Patterns: []string{},
	}

	jsonBody, _ = json.Marshal(invalidRequest)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing patterns, got %d", w.Code)
	}

	// Test missing name when location/tag not provided
	invalidRequest2 := CreateCampaignRequest{
		Patterns: []string{"*.go"},
	}

	jsonBody, _ = json.Marshal(invalidRequest2)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing name and location/tag, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/api/v1/campaigns/find-or-create", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	findOrCreateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestUpdateCampaignHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-update-campaign-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		t.Fatalf("Failed to create data directory: %v", err)
	}

	// Create test campaign
	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-update-campaign",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{"*.go"},
		Notes:              nil,
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test updating campaign notes
	newNotes := "Updated campaign notes for testing"
	updates := struct {
		Notes *string `json:"notes,omitempty"`
	}{
		Notes: &newNotes,
	}

	jsonBody, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify campaign was updated
	updatedCampaign, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("Failed to load updated campaign: %v", err)
	}

	if updatedCampaign.Notes == nil || *updatedCampaign.Notes != newNotes {
		t.Errorf("Expected notes to be %q, got %v", newNotes, updatedCampaign.Notes)
	}

	if !updatedCampaign.UpdatedAt.After(campaign.UpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/invalid-uuid", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+nonExistentID.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String(), strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	updateCampaignHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}
