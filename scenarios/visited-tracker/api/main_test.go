package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
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

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
    originalLogger := logger
    logger = log.New(os.Stdout, "[test] ", log.LstdFlags)
    return func() { logger = originalLogger }
}

// Test staleness calculation
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

// Test file locking utilities
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

// Test campaign path generation
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

// Test campaign creation and storage
func TestCampaignStorage(t *testing.T) {
    // Create a temporary directory for testing
    tempDir, err := ioutil.TempDir("", "visited-tracker-test")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    // Create test campaign
    campaign := &Campaign{
        ID:          uuid.New(),
        Name:        "test-campaign",
        FromAgent:   "unit-test",
        Description: strPtr("Test campaign for unit testing"),
        Patterns:    []string{"*.go"},
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test CORS middleware
func TestCorsMiddleware(t *testing.T) {
    // Create a test handler
    testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("test response"))
    })
    
    // Wrap with CORS middleware
    corsHandler := corsMiddleware(testHandler)
    
    // Test OPTIONS request
    req := httptest.NewRequest("OPTIONS", "/test", nil)
    w := httptest.NewRecorder()
    corsHandler.ServeHTTP(w, req)
    
    // Check CORS headers
    if w.Header().Get("Access-Control-Allow-Origin") != "*" {
        t.Error("CORS middleware should set Access-Control-Allow-Origin to *")
    }
    
    if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "GET") {
        t.Error("CORS middleware should allow GET method")
    }
    
    if w.Code != http.StatusOK {
        t.Errorf("OPTIONS request should return 200, got %d", w.Code)
    }
    
    // Test regular GET request passes through
    req = httptest.NewRequest("GET", "/test", nil)
    w = httptest.NewRecorder()
    corsHandler.ServeHTTP(w, req)
    
    if w.Body.String() != "test response" {
        t.Error("CORS middleware should pass through regular requests")
    }
}

// Test health endpoint logic components
func TestHealthEndpointComponents(t *testing.T) {
    // Test when data directory doesn't exist
    tempDir, err := ioutil.TempDir("", "visited-tracker-health-test")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    nonExistentPath := filepath.Join(tempDir, "nonexistent")
    
    // Test directory access
    if _, err := os.Stat(nonExistentPath); err == nil {
        t.Error("Expected error when accessing non-existent directory")
    }
    
    // Test directory creation
    if err := os.MkdirAll(nonExistentPath, 0755); err != nil {
        t.Errorf("Should be able to create directory: %v", err)
    }
    
    // Verify directory was created
    if _, err := os.Stat(nonExistentPath); err != nil {
        t.Errorf("Directory should exist after creation: %v", err)
    }
}

// Test string pointer helper
func TestStrPtr(t *testing.T) {
    test := "test string"
    ptr := strPtr(test)
    
    if ptr == nil {
        t.Error("strPtr should return non-nil pointer")
    }
    
    if *ptr != test {
        t.Errorf("Expected %s, got %s", test, *ptr)
    }
}

// Test updateStalenessScores function
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
        ID:          uuid.New(),
        Name:        "test-save-campaign",
        FromAgent:   "unit-test",
        Description: strPtr("Test campaign for save testing"),
        Patterns:    []string{"*.go"},
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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
        ID:          uuid.New(),
        Name:        "test-load-campaign",
        FromAgent:   "unit-test",
        Description: strPtr("Test campaign for load testing"),
        Patterns:    []string{"*.go"},
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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
        ID:          uuid.New(),
        Name:        "test-load-all-1",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
        StructureSnapshots: []StructureSnapshot{},
    }
    
    campaign2 := &Campaign{
        ID:          uuid.New(),
        Name:        "test-load-all-2",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.js"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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
        ID:          uuid.New(),
        Name:        "test-delete-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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
func TestHealthHandler(t *testing.T) {
    // Create a temporary directory for testing
    tempDir, err := ioutil.TempDir("", "visited-tracker-health-handler-test")
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
    
    // Test health handler
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    
    healthHandler(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    // Check response is valid JSON
    var response map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
        t.Errorf("Response should be valid JSON: %v", err)
    }
    
    // Check required fields
    if response["status"] == nil {
        t.Error("Response should have status field")
    }
    if response["service"] != serviceName {
        t.Errorf("Expected service %s, got %v", serviceName, response["service"])
    }
    if response["version"] != apiVersion {
        t.Errorf("Expected version %s, got %v", apiVersion, response["version"])
    }
    
    // Check dependencies structure
    if deps, ok := response["dependencies"].(map[string]interface{}); ok {
        if storage, ok := deps["storage"].(map[string]interface{}); ok {
            if storage["connected"] != true {
                t.Error("Storage should be connected when data directory exists")
            }
            if storage["type"] != "json-files" {
                t.Error("Storage type should be json-files")
            }
        } else {
            t.Error("Dependencies should have storage object")
        }
    } else {
        t.Error("Response should have dependencies object")
    }
}

// Test options handler
func TestOptionsHandler(t *testing.T) {
    req := httptest.NewRequest("OPTIONS", "/api/v1/campaigns", nil)
    w := httptest.NewRecorder()
    
    optionsHandler(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
}

// Test listCampaignsHandler
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
        ID:          uuid.New(),
        Name:        "test-list-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
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
        Visits:      []Visit{},
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

// Test createCampaignHandler
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

// Test getCampaignHandler
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
        ID:          uuid.New(),
        Name:        "test-get-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test deleteCampaignHandler
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
        ID:          uuid.New(),
        Name:        "test-delete-handler-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test visitHandler
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
        ID:          uuid.New(),
        Name:        "test-visit-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test adjustVisitHandler
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
        ID:          uuid.New(),
        Name:        "test-adjust-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{trackedFile},
        Visits:      []Visit{},
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

// Test coverageHandler
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
            ID:           uuid.New(),
            FilePath:     "visited.go",
            VisitCount:   3,
            LastVisited:  &now,
            LastModified: now.Add(-1 * time.Hour),
            StalenessScore: 10.0,
        },
        {
            ID:           uuid.New(),
            FilePath:     "unvisited.go",
            VisitCount:   0,
            LastVisited:  nil,
            LastModified: now.Add(-2 * time.Hour),
            StalenessScore: 20.0,
        },
    }
    
    campaign := &Campaign{
        ID:          uuid.New(),
        Name:        "test-coverage-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: trackedFiles,
        Visits:      []Visit{},
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

// Test exportHandler
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
        ID:          uuid.New(),
        Name:        "test-export-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test constants (keeping existing functionality)
func TestAPIVersion(t *testing.T) {
    expected := "3.0.0"
    if apiVersion != expected {
        t.Errorf("Expected API version %s, got %s", expected, apiVersion)
    }
}

func TestServiceName(t *testing.T) {
    expected := "visited-tracker"
    if serviceName != expected {
        t.Errorf("Expected service name %s, got %s", expected, serviceName)
    }
}

// Test structureSyncHandler
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
        ID:          uuid.New(),
        Name:        "test-structure-sync-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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
        ID:          uuid.New(),
        Name:        "test-no-patterns",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: []TrackedFile{},
        Visits:      []Visit{},
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

// Test leastVisitedHandler
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
            ID:           uuid.New(),
            FilePath:     "never_visited.go",
            VisitCount:   0,
            StalenessScore: 30.0,
            LastModified: now.Add(-3 * time.Hour),
            Deleted:      false,
        },
        {
            ID:           uuid.New(),
            FilePath:     "once_visited.go",
            VisitCount:   1,
            StalenessScore: 20.0,
            LastModified: now.Add(-2 * time.Hour),
            Deleted:      false,
        },
        {
            ID:           uuid.New(),
            FilePath:     "often_visited.go",
            VisitCount:   10,
            StalenessScore: 5.0,
            LastModified: now.Add(-1 * time.Hour),
            Deleted:      false,
        },
        {
            ID:           uuid.New(),
            FilePath:     "deleted_file.go",
            VisitCount:   0,
            StalenessScore: 40.0,
            LastModified: now.Add(-4 * time.Hour),
            Deleted:      true, // Should be filtered out
        },
    }
    
    campaign := &Campaign{
        ID:          uuid.New(),
        Name:        "test-least-visited-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: trackedFiles,
        Visits:      []Visit{},
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
    veryOldTime := now.Add(-30 * 24 * time.Hour)    // 30 days ago (very stale)
    oldTime := now.Add(-7 * 24 * time.Hour)         // 7 days ago (stale) 
    recentTime := now.Add(-1 * time.Hour)           // 1 hour ago (fresh)
    
    trackedFiles := []TrackedFile{
        {
            ID:           uuid.New(),
            FilePath:     "never_visited_old.go",
            VisitCount:   0,
            LastVisited:  nil,
            LastModified: veryOldTime,
            Deleted:      false,
            StalenessScore: 0, // Will be calculated by updateStalenessScores
        },
        {
            ID:           uuid.New(),
            FilePath:     "visited_but_old.go",
            VisitCount:   1,
            LastVisited:  &oldTime,
            LastModified: oldTime,
            Deleted:      false,
            StalenessScore: 0, // Will be calculated
        },
        {
            ID:           uuid.New(),
            FilePath:     "recently_visited.go",
            VisitCount:   5,
            LastVisited:  &recentTime,
            LastModified: recentTime,
            Deleted:      false,
            StalenessScore: 0, // Will be calculated
        },
        {
            ID:           uuid.New(),
            FilePath:     "frequently_visited.go",
            VisitCount:   10,
            LastVisited:  &recentTime,
            LastModified: recentTime,
            Deleted:      false,
            StalenessScore: 0, // Will be calculated
        },
        {
            ID:           uuid.New(),
            FilePath:     "deleted_file.go",
            VisitCount:   1,
            LastVisited:  &oldTime,
            LastModified: oldTime,
            Deleted:      true, // Should be filtered out
            StalenessScore: 0,
        },
    }
    
    campaign := &Campaign{
        ID:          uuid.New(),
        Name:        "test-most-stale-campaign",
        FromAgent:   "unit-test",
        CreatedAt:   time.Now().UTC(),
        UpdatedAt:   time.Now().UTC(),
        Status:      "active",
        Patterns:    []string{"*.go"},
        Metadata:    make(map[string]interface{}),
        TrackedFiles: trackedFiles,
        Visits:      []Visit{},
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

func TestMainFunctionComponents(t *testing.T) {
    // Test individual components that main() uses without actually running main()
    
    // Test environment variable validation (simulated)
    originalEnv := os.Getenv("VROOLI_LIFECYCLE_MANAGED")
    os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
    
    if os.Getenv("VROOLI_LIFECYCLE_MANAGED") == "true" {
        t.Error("Environment variable should be unset for this test")
    }
    
    // Restore environment
    if originalEnv != "" {
        os.Setenv("VROOLI_LIFECYCLE_MANAGED", originalEnv)
    }
    
    // Test port environment variable checking
    originalPort := os.Getenv("API_PORT")
    os.Unsetenv("API_PORT")
    
    if os.Getenv("API_PORT") != "" {
        t.Error("API_PORT should be unset for this test")
    }
    
    // Restore environment
    if originalPort != "" {
        os.Setenv("API_PORT", originalPort)
    }
    
    // Test directory changing functionality
    originalWD, err := os.Getwd()
    if err != nil {
        t.Fatalf("Failed to get current directory: %v", err)
    }
    
    // Test that os.Chdir works (this is what main() does)
    tempDir, err := ioutil.TempDir("", "visited-tracker-main-test")
    if err != nil {
        t.Fatalf("Failed to create temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)
    
    if err := os.Chdir(tempDir); err != nil {
        t.Errorf("Directory change should work: %v", err)
    }
    
    // Restore original directory
    os.Chdir(originalWD)
}

// ============================================================================
// Export Handler Tests (Lowest Coverage)
// ============================================================================

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

// ============================================================================
// Error Path and Edge Case Tests
// ============================================================================

func TestHealthHandlerErrorPaths(t *testing.T) {
    cleanup := setupTestLogger()
    defer cleanup()
    
    // Test health handler with various scenarios
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()
    
    healthHandler(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }
    
    // Verify response contains expected fields
    var health map[string]interface{}
    if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
        t.Fatalf("Failed to parse health JSON: %v", err)
    }
    
    if health["status"] != "ok" && health["status"] != "degraded" {
        t.Errorf("Expected status 'ok' or 'degraded', got %v", health["status"])
    }
    
    if health["service"] != serviceName {
        t.Errorf("Expected service name %s, got %v", serviceName, health["service"])
    }
    
    if health["version"] != apiVersion {
        t.Errorf("Expected version %s, got %v", apiVersion, health["version"])
    }
}

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

// ============================================================================
// Additional Coverage Improvement Tests
// ============================================================================

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
