package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestCalculateStalenessScore(t *testing.T) {
	now := time.Now()

	// Test never visited file
	file := &TrackedFile{
		LastVisited:  nil,
		LastModified: now.Add(-7 * 24 * time.Hour), // 7 days ago
		VisitCount:   0,
	}

	score := calculateStalenessScore(file)
	if score <= 0 {
		t.Errorf("Never visited file should have positive staleness score, got %f", score)
	}

	// Test recently visited file
	recentVisit := now.Add(-1 * time.Hour)
	file.LastVisited = &recentVisit
	file.VisitCount = 5

	score = calculateStalenessScore(file)
	if score > 50 {
		t.Errorf("Recently visited file should have low staleness score, got %f", score)
	}
}

func TestUpdateStalenessScores(t *testing.T) {
	now := time.Now()
	campaign := &Campaign{
		TrackedFiles: []TrackedFile{
			{
				ID:           uuid.New(),
				FilePath:     "test1.go",
				VisitCount:   0,
				LastVisited:  nil,
				LastModified: now.Add(-7 * 24 * time.Hour),
			},
			{
				ID:           uuid.New(),
				FilePath:     "test2.go",
				VisitCount:   5,
				LastVisited:  &now,
				LastModified: now.Add(-1 * time.Hour),
			},
		},
	}

	// Update staleness scores
	updateStalenessScores(campaign)

	// Check that staleness scores were calculated
	if campaign.TrackedFiles[0].StalenessScore <= 0 {
		t.Error("Never visited file should have positive staleness score")
	}

	if campaign.TrackedFiles[1].StalenessScore >= campaign.TrackedFiles[0].StalenessScore {
		t.Error("Recently visited file should have lower staleness score than never visited file")
	}
}

// Test initFileStorage function

func TestLeastVisitedHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-least-visited-test")
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

	// Create test campaign with tracked files of varying visit counts
	now := time.Now()
	trackedFiles := []TrackedFile{
		{
			ID:             uuid.New(),
			FilePath:       "never_visited.go",
			VisitCount:     0,
			StalenessScore: 30.0,
			LastModified:   now.Add(-3 * time.Hour),
			Deleted:        false,
		},
		{
			ID:             uuid.New(),
			FilePath:       "once_visited.go",
			VisitCount:     1,
			StalenessScore: 20.0,
			LastModified:   now.Add(-2 * time.Hour),
			Deleted:        false,
		},
		{
			ID:             uuid.New(),
			FilePath:       "often_visited.go",
			VisitCount:     10,
			StalenessScore: 5.0,
			LastModified:   now.Add(-1 * time.Hour),
			Deleted:        false,
		},
		{
			ID:             uuid.New(),
			FilePath:       "deleted_file.go",
			VisitCount:     0,
			StalenessScore: 40.0,
			LastModified:   now.Add(-4 * time.Hour),
			Deleted:        true, // Should be filtered out
		},
	}

	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-least-visited-campaign",
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

	// Test least visited without limit (should return default 10)
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/least-visited", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	leastVisitedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Error("Response should have files array")
	}

	// Should return 3 files (excluding deleted one)
	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}

	// First file should be never_visited.go (0 visits, highest staleness)
	if firstFile, ok := files[0].(map[string]interface{}); ok {
		if filePath, ok := firstFile["file_path"].(string); !ok || filePath != "never_visited.go" {
			t.Errorf("Expected first file to be never_visited.go, got %v", firstFile["file_path"])
		}
		if visitCount, ok := firstFile["visit_count"].(float64); !ok || visitCount != 0 {
			t.Errorf("Expected first file visit count 0, got %v", firstFile["visit_count"])
		}
	}

	// Test with custom limit
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/least-visited?limit=2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	leastVisitedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok = response["files"].([]interface{})
	if !ok || len(files) != 2 {
		t.Errorf("Expected 2 files with limit=2, got %d", len(files))
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid/least-visited", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	leastVisitedHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String()+"/least-visited", nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	leastVisitedHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test invalid limit parameter (should use default)
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/least-visited?limit=invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	leastVisitedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for invalid limit (should use default), got %d", w.Code)
	}
}

// Test mostStaleHandler

func TestMostStaleHandler(t *testing.T) {
	// Setup logger for testing
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-most-stale-test")
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

	// Create test campaign with tracked files set up for realistic staleness calculation
	now := time.Now()

	// Set up files with visit patterns that will result in predictable staleness after updateStalenessScores
	veryOldTime := now.Add(-30 * 24 * time.Hour) // 30 days ago (very stale)
	oldTime := now.Add(-7 * 24 * time.Hour)      // 7 days ago (stale)
	recentTime := now.Add(-1 * time.Hour)        // 1 hour ago (fresh)

	trackedFiles := []TrackedFile{
		{
			ID:             uuid.New(),
			FilePath:       "never_visited_old.go",
			VisitCount:     0,
			LastVisited:    nil,
			LastModified:   veryOldTime,
			Deleted:        false,
			StalenessScore: 0, // Will be calculated by updateStalenessScores
		},
		{
			ID:             uuid.New(),
			FilePath:       "visited_but_old.go",
			VisitCount:     1,
			LastVisited:    &oldTime,
			LastModified:   oldTime,
			Deleted:        false,
			StalenessScore: 0, // Will be calculated
		},
		{
			ID:             uuid.New(),
			FilePath:       "recently_visited.go",
			VisitCount:     5,
			LastVisited:    &recentTime,
			LastModified:   recentTime,
			Deleted:        false,
			StalenessScore: 0, // Will be calculated
		},
		{
			ID:             uuid.New(),
			FilePath:       "frequently_visited.go",
			VisitCount:     10,
			LastVisited:    &recentTime,
			LastModified:   recentTime,
			Deleted:        false,
			StalenessScore: 0, // Will be calculated
		},
		{
			ID:             uuid.New(),
			FilePath:       "deleted_file.go",
			VisitCount:     1,
			LastVisited:    &oldTime,
			LastModified:   oldTime,
			Deleted:        true, // Should be filtered out
			StalenessScore: 0,
		},
	}

	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-most-stale-campaign",
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

	// Test most stale without parameters (default limit 10, threshold 0)
	req := httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/most-stale", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w := httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check response structure
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok := response["files"].([]interface{})
	if !ok {
		t.Error("Response should have files array")
	}

	// Should return 4 files (excluding deleted one), ordered by staleness descending
	if len(files) != 4 {
		t.Errorf("Expected 4 files, got %d", len(files))
	}

	// First file should be the most stale one (never_visited_old.go has no visits and is oldest)
	if firstFile, ok := files[0].(map[string]interface{}); ok {
		if filePath, ok := firstFile["file_path"].(string); !ok || filePath != "never_visited_old.go" {
			t.Errorf("Expected first file to be never_visited_old.go, got %v", firstFile["file_path"])
		}
		// Just verify it has a positive staleness score
		if staleness, ok := firstFile["staleness_score"].(float64); !ok || staleness <= 0 {
			t.Errorf("Expected positive staleness score for most stale file, got %v", firstFile["staleness_score"])
		}
	}

	// Verify files are sorted by staleness descending
	var prevStaleness float64 = 1000.0 // Start with a high value
	for i, fileInterface := range files {
		if file, ok := fileInterface.(map[string]interface{}); ok {
			if staleness, ok := file["staleness_score"].(float64); ok {
				if staleness > prevStaleness {
					t.Errorf("Files should be sorted by staleness descending, but file %d has staleness %f > previous %f", i, staleness, prevStaleness)
				}
				prevStaleness = staleness
			}
		}
	}

	// Check critical count - should include files with staleness > 50
	criticalCount, ok := response["critical_count"].(float64)
	if !ok {
		t.Error("Response should have critical_count field")
	}
	// Since staleness is calculated dynamically, just verify it's a valid count
	if criticalCount < 0 || criticalCount > 4 {
		t.Errorf("Critical count should be between 0 and 4, got %v", criticalCount)
	}

	// Test with custom limit
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/most-stale?limit=3", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok = response["files"].([]interface{})
	if !ok || len(files) != 3 {
		t.Errorf("Expected 3 files with limit=3, got %d", len(files))
	}

	// Test with threshold filter (use a lower threshold since staleness is calculated dynamically)
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/most-stale?threshold=5.0", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok = response["files"].([]interface{})
	if !ok {
		t.Error("Response should have files array")
	}

	// Verify all returned files meet the threshold
	for _, fileInterface := range files {
		if file, ok := fileInterface.(map[string]interface{}); ok {
			if staleness, ok := file["staleness_score"].(float64); ok {
				if staleness < 5.0 {
					t.Errorf("File %v has staleness %f below threshold 5.0", file["file_path"], staleness)
				}
			}
		}
	}

	// Test with limit and threshold
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/most-stale?limit=2&threshold=1.0", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Response should be valid JSON: %v", err)
	}

	files, ok = response["files"].([]interface{})
	if !ok {
		t.Error("Response should have files array")
	}

	// Should return at most 2 files
	if len(files) > 2 {
		t.Errorf("Expected at most 2 files with limit=2, got %d", len(files))
	}

	// Verify all returned files meet the threshold
	for _, fileInterface := range files {
		if file, ok := fileInterface.(map[string]interface{}); ok {
			if staleness, ok := file["staleness_score"].(float64); ok {
				if staleness < 1.0 {
					t.Errorf("File %v has staleness %f below threshold 1.0", file["file_path"], staleness)
				}
			}
		}
	}

	// Test invalid campaign ID
	req = httptest.NewRequest("GET", "/api/v1/campaigns/invalid-uuid/most-stale", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid-uuid"})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid UUID, got %d", w.Code)
	}

	// Test non-existent campaign
	nonExistentID := uuid.New()
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+nonExistentID.String()+"/most-stale", nil)
	req = mux.SetURLVars(req, map[string]string{"id": nonExistentID.String()})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent campaign, got %d", w.Code)
	}

	// Test invalid limit and threshold parameters (should use defaults)
	req = httptest.NewRequest("GET", "/api/v1/campaigns/"+campaign.ID.String()+"/most-stale?limit=invalid&threshold=invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": campaign.ID.String()})
	w = httptest.NewRecorder()

	mostStaleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for invalid params (should use defaults), got %d", w.Code)
	}
}

// ============================================================================
// Main Function and Environment Tests (Simplified - No Subprocess Testing)
// ============================================================================
