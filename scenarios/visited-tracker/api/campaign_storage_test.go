package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCampaignStorage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test campaign
	campaign := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-campaign",
		FromAgent:          "unit-test",
		Description:        strPtr("Test campaign for unit testing"),
		Patterns:           []string{"*.go"},
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	// Note: We can't easily override dataDir (const), so we'll use tempDir directly for testing

	// Mock getCampaignPath to use temp directory
	testPath := filepath.Join(tempDir, campaign.ID.String()+".json")

	// Manually save to test path
	data, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal campaign: %v", err)
	}

	if err := os.WriteFile(testPath, data, 0644); err != nil {
		t.Fatalf("Failed to write test campaign file: %v", err)
	}

	// Test loading
	loadedData, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read test campaign file: %v", err)
	}

	var loadedCampaign Campaign
	if err := json.Unmarshal(loadedData, &loadedCampaign); err != nil {
		t.Fatalf("Failed to unmarshal campaign: %v", err)
	}

	// Verify loaded data matches original
	if loadedCampaign.ID != campaign.ID {
		t.Errorf("Expected ID %s, got %s", campaign.ID, loadedCampaign.ID)
	}
	if loadedCampaign.Name != campaign.Name {
		t.Errorf("Expected name %s, got %s", campaign.Name, loadedCampaign.Name)
	}
}

func TestInitFileStorage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-init-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Initialize logger for testing (initFileStorage uses global logger)
	cleanup := setupTestLogger()
	defer cleanup()

	// Test initFileStorage
	if err := initFileStorage(); err != nil {
		t.Errorf("initFileStorage should succeed: %v", err)
	}

	// Verify directory was created
	expectedPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Data directory should be created at: %s", expectedPath)
	}

	// Test calling it again (should not fail)
	if err := initFileStorage(); err != nil {
		t.Errorf("initFileStorage should succeed on second call: %v", err)
	}
}

// Test saveCampaign function

func TestSaveCampaign(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-save-test")
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
		Name:               "test-save-campaign",
		FromAgent:          "unit-test",
		Description:        strPtr("Test campaign for save testing"),
		Patterns:           []string{"*.go"},
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	// Test saving campaign
	if err := saveCampaign(campaign); err != nil {
		t.Errorf("saveCampaign should succeed: %v", err)
	}

	// Verify file was created
	expectedPath := getCampaignPath(campaign.ID)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Campaign file should be created at: %s", expectedPath)
	}

	// Verify file contents
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Errorf("Should be able to read saved file: %v", err)
	}

	var loaded Campaign
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Errorf("Saved file should contain valid JSON: %v", err)
	}

	// Verify data matches
	if loaded.ID != campaign.ID {
		t.Errorf("Expected ID %s, got %s", campaign.ID, loaded.ID)
	}
	if loaded.Name != campaign.Name {
		t.Errorf("Expected name %s, got %s", campaign.Name, loaded.Name)
	}
}

// Test loadCampaign function

func TestLoadCampaign(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-load-test")
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
		Name:               "test-load-campaign",
		FromAgent:          "unit-test",
		Description:        strPtr("Test campaign for load testing"),
		Patterns:           []string{"*.go"},
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	// Save campaign first
	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Test loading campaign
	loaded, err := loadCampaign(campaign.ID)
	if err != nil {
		t.Errorf("loadCampaign should succeed: %v", err)
	}

	// Verify loaded data
	if loaded.ID != campaign.ID {
		t.Errorf("Expected ID %s, got %s", campaign.ID, loaded.ID)
	}
	if loaded.Name != campaign.Name {
		t.Errorf("Expected name %s, got %s", campaign.Name, loaded.Name)
	}

	// Test loading non-existent campaign
	nonExistentID := uuid.New()
	_, err = loadCampaign(nonExistentID)
	if err == nil {
		t.Error("loadCampaign should fail for non-existent campaign")
	}
	if !strings.Contains(err.Error(), "campaign not found") {
		t.Errorf("Expected 'campaign not found' error, got: %v", err)
	}
}

// Test loadAllCampaigns function

func TestLoadAllCampaigns(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-load-all-test")
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

	// Test loading from empty directory
	campaigns, err := loadAllCampaigns()
	if err != nil {
		t.Errorf("loadAllCampaigns should succeed on empty directory: %v", err)
	}
	if len(campaigns) != 0 {
		t.Errorf("Expected 0 campaigns, got %d", len(campaigns))
	}

	// Create and save multiple test campaigns
	campaign1 := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-load-all-1",
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

	campaign2 := &Campaign{
		ID:                 uuid.New(),
		Name:               "test-load-all-2",
		FromAgent:          "unit-test",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Patterns:           []string{"*.js"},
		Metadata:           make(map[string]interface{}),
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}

	// Save campaigns
	if err := saveCampaign(campaign1); err != nil {
		t.Fatalf("Failed to save campaign1: %v", err)
	}
	if err := saveCampaign(campaign2); err != nil {
		t.Fatalf("Failed to save campaign2: %v", err)
	}

	// Test loading all campaigns
	campaigns, err = loadAllCampaigns()
	if err != nil {
		t.Errorf("loadAllCampaigns should succeed: %v", err)
	}
	if len(campaigns) != 2 {
		t.Errorf("Expected 2 campaigns, got %d", len(campaigns))
	}

	// Verify campaigns were loaded correctly
	found1, found2 := false, false
	for _, c := range campaigns {
		if c.ID == campaign1.ID {
			found1 = true
			if c.Name != campaign1.Name {
				t.Errorf("Campaign1 name mismatch: expected %s, got %s", campaign1.Name, c.Name)
			}
		}
		if c.ID == campaign2.ID {
			found2 = true
			if c.Name != campaign2.Name {
				t.Errorf("Campaign2 name mismatch: expected %s, got %s", campaign2.Name, c.Name)
			}
		}
	}

	if !found1 {
		t.Error("Campaign1 not found in loaded campaigns")
	}
	if !found2 {
		t.Error("Campaign2 not found in loaded campaigns")
	}
}

func TestLoadAllCampaignsDirectoryMissing(t *testing.T) {
	teardownLogger := setupTestLogger()
	defer teardownLogger()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to determine working directory: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalWd)

	campaigns, err := loadAllCampaigns()
	if err != nil {
		t.Fatalf("Expected no error when campaigns directory missing, got %v", err)
	}
	if len(campaigns) != 0 {
		t.Fatalf("Expected 0 campaigns, got %d", len(campaigns))
	}
}

func TestLoadAllCampaignsSkipsInvalidFiles(t *testing.T) {
	teardownLogger := setupTestLogger()
	defer teardownLogger()

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to determine working directory: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalWd)

	campaignsDir := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(campaignsDir, 0o755); err != nil {
		t.Fatalf("Failed to create campaigns directory: %v", err)
	}

	// Invalid JSON file should be ignored gracefully
	if err := os.WriteFile(filepath.Join(campaignsDir, "invalid.json"), []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid campaign file: %v", err)
	}

	validCampaign := Campaign{
		ID:        uuid.New(),
		Name:      "valid-campaign",
		FromAgent: "unit-test",
		Patterns:  []string{"**/*.go"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Status:    "active",
	}

	data, err := json.Marshal(validCampaign)
	if err != nil {
		t.Fatalf("Failed to marshal valid campaign: %v", err)
	}
	if err := os.WriteFile(filepath.Join(campaignsDir, "valid.json"), data, 0o644); err != nil {
		t.Fatalf("Failed to write valid campaign file: %v", err)
	}

	campaigns, err := loadAllCampaigns()
	if err != nil {
		t.Fatalf("Expected no error when loading campaigns, got %v", err)
	}
	if len(campaigns) != 1 {
		t.Fatalf("Expected 1 valid campaign, got %d", len(campaigns))
	}
	if campaigns[0].Name != validCampaign.Name {
		t.Fatalf("Expected campaign name %q, got %q", validCampaign.Name, campaigns[0].Name)
	}
}

// Test deleteCampaignFile function

func TestDeleteCampaignFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "visited-tracker-delete-test")
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
		Name:               "test-delete-campaign",
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

	// Save campaign first
	if err := saveCampaign(campaign); err != nil {
		t.Fatalf("Failed to save test campaign: %v", err)
	}

	// Verify file exists
	filePath := getCampaignPath(campaign.ID)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Campaign file should exist before deletion")
	}

	// Test deleting campaign file
	if err := deleteCampaignFile(campaign.ID); err != nil {
		t.Errorf("deleteCampaignFile should succeed: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Campaign file should be deleted")
	}

	// Test deleting non-existent file (should not error)
	if err := deleteCampaignFile(uuid.New()); err != nil {
		t.Errorf("deleteCampaignFile should not error for non-existent file: %v", err)
	}
}

// Test health handler

func TestLoadAllCampaignsErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Test by changing to a directory where the data dir doesn't exist
	originalWD, _ := os.Getwd()
	testDir := "/tmp/visited-tracker-test-no-data"
	os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	os.Chdir(testDir)
	defer os.Chdir(originalWD)

	campaigns, err := loadAllCampaigns()

	// Should handle missing directory gracefully
	if err != nil {
		t.Errorf("loadAllCampaigns should handle missing directory gracefully, got error: %v", err)
	}

	if len(campaigns) != 0 {
		t.Errorf("Expected 0 campaigns from non-existent directory, got %d", len(campaigns))
	}
}

func TestGetCampaignPath(t *testing.T) {
	campaignID := uuid.New()
	path := getCampaignPath(campaignID)

	expectedSuffix := campaignID.String() + ".json"
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("Campaign path should end with %s, got %s", expectedSuffix, path)
	}

	if !strings.Contains(path, "scenarios/visited-tracker/data/campaigns") {
		t.Errorf("Campaign path should contain correct directory structure, got %s", path)
	}
}

func TestGetFileLock(t *testing.T) {
	filename := "test_file.json"

	lock1 := getFileLock(filename)
	lock2 := getFileLock(filename)

	// Should return the same lock instance for the same filename
	if lock1 != lock2 {
		t.Error("getFileLock should return the same lock instance for the same filename")
	}

	// Test different filenames get different locks
	lock3 := getFileLock("different_file.json")
	if lock1 == lock3 {
		t.Error("Different filenames should get different lock instances")
	}
}
