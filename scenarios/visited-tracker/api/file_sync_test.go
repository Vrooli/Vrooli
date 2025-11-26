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

func TestStructureSyncHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-structure-sync-test")
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

	// Create test files for syncing
	testFiles := []string{"test1.go", "test2.go", "test3.js"}
	for _, file := range testFiles {
		content := fmt.Sprintf("// Test content for %s\npackage main\n", file)
		if err := ioutil.WriteFile(file, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Create test campaign with patterns
	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-structure-sync-campaign",
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

	// Test successful structure sync with campaign patterns
	syncReq := StructureSyncRequest{}

	jsonBody, _ := json.Marshal(syncReq)
	req := httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/structure-sync", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	if added, ok := response["added"].(float64); !ok || added < 1 {
		t.Errorf("Expected at least 1 file added, got %v", response["added"])
	}

	// Test sync with custom patterns
	syncReqWithPatterns := StructureSyncRequest{
		Patterns: []string{"*.js"},
	}

	jsonBody, _ = json.Marshal(syncReqWithPatterns)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/structure-sync", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("POST", "/api/v1/campaigns/invalid-uuid/structure-sync", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+nonExistentID.String()+"/structure-sync", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaign.ID.String()+"/structure-sync", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

	// Test campaign with no patterns (should fail)
	campaignNoPatterns := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-no-patterns",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{},
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaignNoPatterns); err != nil {
		t.Fatalf("Failed to save campaign with no patterns: %v", err)
	}

	syncReq = StructureSyncRequest{}
	jsonBody, _ = json.Marshal(syncReq)
	req = httptest.NewRequest("POST", "/api/v1/campaigns/"+campaignNoPatterns.ID.String()+"/structure-sync", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaignNoPatterns.ID.String()})
	w = httptest.NewRecorder()

	structureSyncHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for campaign with no patterns, got %d", w.Code)
	}
}

func TestSyncCampaignFilesErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup test environment
	tempDir, err := ioutil.TempDir("", "visited-tracker-sync-test")
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
	description := "Test campaign for sync testing"
	campaign := &Campaign{
		ID:          uuid.New(),
		Name:        "test-sync-campaign",
		Description: &description,
		CreatedAt:   time.Now(),
		Patterns:    []string{"*.go"},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save campaign: %v", err)
	}
	defer deleteCampaignFile(campaign.ID)

	// Test sync with campaign patterns (should use campaign defaults)
	_, err = syncCampaignFiles(campaign, campaign.Patterns)
	if err != nil && !strings.Contains(err.Error(), "no patterns specified") {
		t.Errorf("Sync with campaign patterns should work: %v", err)
	}

	// Test sync with specific patterns
	_, err = syncCampaignFiles(campaign, []string{"*.go"})
	if err != nil {
		t.Errorf("Sync with go patterns should work: %v", err)
	}

	// Test sync with multiple patterns
	_, err = syncCampaignFiles(campaign, []string{"*.js", "*.ts"})
	if err != nil {
		t.Errorf("Sync with multiple patterns should work: %v", err)
	}

	// Test error path - sync with empty patterns
	_, err = syncCampaignFiles(campaign, []string{})
	if err == nil {
		t.Error("Sync with empty patterns should return error")
	}
}
