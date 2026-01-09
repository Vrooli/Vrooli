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

func TestUpdateFileNotesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-update-file-notes-test")
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

	// Create test campaign with tracked file
	fileID := uuid.New()
	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-file-notes-campaign",
		FromAgent: "unit-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    "active",
		Patterns:  []string{"*.go"},
		Metadata:  make(map[string]interface{}),
		TrackedFiles: []TrackedFile{
			{
				ID:           fileID,
				FilePath:     "test.go",
				AbsolutePath: "/tmp/test.go",
				VisitCount:   0,
				Notes:        nil,
				Metadata:     make(map[string]interface{}),
			},
		},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test updating file notes
	newNotes := "File-level notes for test.go"
	updates := struct {
		Notes *string `json:"notes,omitempty"`
	}{
		Notes: &newNotes,
	}

	jsonBody, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/notes", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w := httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file notes were updated
	updatedCampaign, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("Failed to load updated campaign: %v", err)
	}

	if len(updatedCampaign.TrackedFiles) == 0 {
		t.Fatal("Campaign should have tracked files")
	}

	if updatedCampaign.TrackedFiles[0].Notes == nil || *updatedCampaign.TrackedFiles[0].Notes != newNotes {
		t.Errorf("Expected file notes to be %q, got %v", newNotes, updatedCampaign.TrackedFiles[0].Notes)
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/invalid-uuid/files/"+fileID.String()+"/notes", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid", "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid campaign UUID, got %d", w.Code)
	}

	// Test invalid file ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/invalid-uuid/notes", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": "invalid-uuid"})
	w = httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+nonExistentID.String()+"/files/"+fileID.String()+"/notes", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test non-existent file
	nonExistentFileID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+nonExistentFileID.String()+"/notes", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": nonExistentFileID.String()})
	w = httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/notes", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFileNotesHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestUpdateFilePriorityHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-update-file-priority-test")
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

	// Create test campaign with tracked file
	fileID := uuid.New()
	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-file-priority-campaign",
		FromAgent: "unit-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    "active",
		Patterns:  []string{"*.go"},
		Metadata:  make(map[string]interface{}),
		TrackedFiles: []TrackedFile{
			{
				ID:             fileID,
				FilePath:       "test.go",
				AbsolutePath:   "/tmp/test.go",
				VisitCount:     0,
				PriorityWeight: 1.0,
				LastModified:   time.Now().UTC(),
				Metadata:       make(map[string]interface{}),
			},
		},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test updating file priority
	updates := struct {
		PriorityWeight float64 `json:"priority_weight"`
	}{
		PriorityWeight: 2.5,
	}

	jsonBody, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/priority", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w := httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file priority was updated
	updatedCampaign, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("Failed to load updated campaign: %v", err)
	}

	if len(updatedCampaign.TrackedFiles) == 0 {
		t.Fatal("Campaign should have tracked files")
	}

	if updatedCampaign.TrackedFiles[0].PriorityWeight != 2.5 {
		t.Errorf("Expected priority weight to be 2.5, got %f", updatedCampaign.TrackedFiles[0].PriorityWeight)
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/invalid-uuid/files/"+fileID.String()+"/priority", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid", "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid campaign UUID, got %d", w.Code)
	}

	// Test invalid file ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/invalid-uuid/priority", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": "invalid-uuid"})
	w = httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+nonExistentID.String()+"/files/"+fileID.String()+"/priority", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test non-existent file
	nonExistentFileID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+nonExistentFileID.String()+"/priority", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": nonExistentFileID.String()})
	w = httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/priority", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	updateFilePriorityHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestToggleFileExclusionHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tempDir, err := ioutil.TempDir("", "visited-tracker-toggle-exclusion-test")
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

	// Create test campaign with tracked file
	fileID := uuid.New()
	campaign := &Campaign{
		ID:        uuid.New(),
		Name:      "test-file-exclusion-campaign",
		FromAgent: "unit-test",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    "active",
		Patterns:  []string{"*.go"},
		Metadata:  make(map[string]interface{}),
		TrackedFiles: []TrackedFile{
			{
				ID:           fileID,
				FilePath:     "test.go",
				AbsolutePath: "/tmp/test.go",
				VisitCount:   0,
				Excluded:     false,
				Metadata:     make(map[string]interface{}),
			},
		},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test excluding file
	updates := struct {
		Excluded bool `json:"excluded"`
	}{
		Excluded: true,
	}

	jsonBody, _ := json.Marshal(updates)
	req := httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w := httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was excluded
	updatedCampaign, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("Failed to load updated campaign: %v", err)
	}

	if len(updatedCampaign.TrackedFiles) == 0 {
		t.Fatal("Campaign should have tracked files")
	}

	if !updatedCampaign.TrackedFiles[0].Excluded {
		t.Error("File should be excluded")
	}

	// Test un-excluding file
	updates.Excluded = false
	jsonBody, _ = json.Marshal(updates)
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify file was un-excluded
	updatedCampaign, err = loadCampaign(campaign.ID)
	if err != nil {
		t.Fatalf("Failed to load updated campaign: %v", err)
	}

	if updatedCampaign.TrackedFiles[0].Excluded {
		t.Error("File should not be excluded")
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/invalid-uuid/files/"+fileID.String()+"/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid", "file_id": fileID.String()})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid campaign UUID, got %d", w.Code)
	}

	// Test invalid file ID
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/invalid-uuid/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": "invalid-uuid"})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid file UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+nonExistentID.String()+"/files/"+fileID.String()+"/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test non-existent file
	nonExistentFileID := uuid.New()
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+nonExistentFileID.String()+"/exclude", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": nonExistentFileID.String()})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent file, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("PATCH", "/api/v1/campaigns/"+campaign.ID.String()+"/files/"+fileID.String()+"/exclude", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String(), "file_id": fileID.String()})
	w = httptest.NewRecorder()

	toggleFileExclusionHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}

}
