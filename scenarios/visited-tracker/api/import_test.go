package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// [REQ:VT-REQ-010] Test importHandler - Campaign Import Functionality
func TestImportHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-import-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory and initialize file storage
	originalWD, _ := os.Getwd()
	defer os.Chdir(originalWD)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	if err := initFileStorage(); err != nil {
		t.Fatalf("Failed to init file storage: %v", err)
	}

	// Create test campaign data to import
	now := time.Now()
	importCampaign := Campaign{
		Name:        "test-import-campaign",
		FromAgent:   "test-agent",
		Description: stringPtr("Test campaign for import testing"),
		Patterns:    []string{"*.go", "*.js"},
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
		TrackedFiles: []TrackedFile{
			{
				FilePath:    "test1.go",
				VisitCount:  5,
				LastVisited: &now,
			},
			{
				FilePath:    "test2.js",
				VisitCount:  3,
			},
		},
		Visits: []Visit{},
		Metadata: map[string]interface{}{
			"test": "data",
		},
	}

	// Test 1: Import new campaign
	t.Run("ImportNewCampaign", func(t *testing.T) {
		body, _ := json.Marshal(importCampaign)
		req := httptest.NewRequest("POST", "/api/v1/campaigns/import", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		importHandler(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["message"] != "Campaign imported successfully" {
			t.Errorf("Expected success message, got %v", response["message"])
		}

		// Verify campaign was saved
		campaignData := response["campaign"].(map[string]interface{})
		if campaignData["name"] != importCampaign.Name {
			t.Errorf("Expected name %s, got %v", importCampaign.Name, campaignData["name"])
		}

		// Verify metadata includes imported_at timestamp
		metadata := campaignData["metadata"].(map[string]interface{})
		if _, exists := metadata["imported_at"]; !exists {
			t.Error("Expected imported_at timestamp in metadata")
		}
	})

	// Test 2: Import with missing required fields
	t.Run("ImportMissingName", func(t *testing.T) {
		invalidCampaign := Campaign{
			Patterns: []string{"*.go"},
		}
		body, _ := json.Marshal(invalidCampaign)
		req := httptest.NewRequest("POST", "/api/v1/campaigns/import", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		importHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test 3: Import with missing patterns
	t.Run("ImportMissingPatterns", func(t *testing.T) {
		invalidCampaign := Campaign{
			Name: "test-campaign",
		}
		body, _ := json.Marshal(invalidCampaign)
		req := httptest.NewRequest("POST", "/api/v1/campaigns/import", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		importHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test 4: Import with invalid JSON
	t.Run("ImportInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/campaigns/import", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		importHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	// Test 5: Merge mode - merge with existing campaign
	t.Run("ImportMergeMode", func(t *testing.T) {
		// Create existing campaign
		existingCampaign := Campaign{
			ID:          uuid.New(),
			Name:        "test-merge-campaign",
			FromAgent:   "test-agent",
			Patterns:    []string{"*.go"},
			Status:      "active",
			CreatedAt:   now,
			UpdatedAt:   now,
			TrackedFiles: []TrackedFile{
				{
					ID:          uuid.New(),
					FilePath:    "existing.go",
					VisitCount:  10,
					LastVisited: &now,
				},
			},
			Metadata: map[string]interface{}{},
		}
		saveCampaign(&existingCampaign)

		// Import campaign with same name but new files
		mergeTime := now.Add(time.Hour)
		mergeCampaign := Campaign{
			Name:     "test-merge-campaign",
			Patterns: []string{"*.go", "*.ts"},
			TrackedFiles: []TrackedFile{
				{
					FilePath:    "existing.go",
					VisitCount:  5, // Lower than existing - should keep existing
					LastVisited: &now,
				},
				{
					FilePath:    "new.go",
					VisitCount:  3,
					LastVisited: &mergeTime,
				},
			},
		}

		body, _ := json.Marshal(mergeCampaign)
		req := httptest.NewRequest("POST", "/api/v1/campaigns/import?merge=true", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		importHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["message"] != "Campaign merged successfully" {
			t.Errorf("Expected merge message, got %v", response["message"])
		}

		// Verify merged campaign has both files
		campaignData := response["campaign"].(map[string]interface{})
		files := campaignData["tracked_files"].([]interface{})
		if len(files) != 2 {
			t.Errorf("Expected 2 files after merge, got %d", len(files))
		}

		// Verify patterns were updated
		patterns := campaignData["patterns"].([]interface{})
		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns after merge, got %d", len(patterns))
		}
	})
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}
