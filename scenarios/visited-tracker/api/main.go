package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	apiVersion  = "3.0.0"
	serviceName = "visited-tracker"
	dataDir     = "data/campaigns"
)

var (
	logger    *log.Logger
	fileLocks = make(map[string]*sync.RWMutex)
	locksLock = sync.RWMutex{}
)

// Models with JSON file storage support
type Campaign struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	FromAgent       string                 `json:"from_agent"`
	Description     *string                `json:"description,omitempty"`
	Patterns        []string               `json:"patterns"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Status          string                 `json:"status"`
	Metadata        map[string]interface{} `json:"metadata"`
	TrackedFiles    []TrackedFile          `json:"tracked_files"`
	Visits          []Visit                `json:"visits"`
	StructureSnapshots []StructureSnapshot `json:"structure_snapshots"`
	// Computed fields (not stored)
	TotalFiles      int     `json:"total_files,omitempty"`
	VisitedFiles    int     `json:"visited_files,omitempty"`  
	CoveragePercent float64 `json:"coverage_percent,omitempty"`
}

type TrackedFile struct {
	ID             uuid.UUID              `json:"id"`
	FilePath       string                 `json:"file_path"`
	AbsolutePath   string                 `json:"absolute_path"`
	VisitCount     int                    `json:"visit_count"`
	FirstSeen      time.Time              `json:"first_seen"`
	LastVisited    *time.Time             `json:"last_visited,omitempty"`
	LastModified   time.Time              `json:"last_modified"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	SizeBytes      int64                  `json:"size_bytes"`
	StalenessScore float64                `json:"staleness_score"`
	Deleted        bool                   `json:"deleted"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Visit struct {
	ID             uuid.UUID              `json:"id"`
	FileID         uuid.UUID              `json:"file_id"`
	Timestamp      time.Time              `json:"timestamp"`
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	DurationMs     *int                   `json:"duration_ms,omitempty"`
	Findings       map[string]interface{} `json:"findings,omitempty"`
}

type StructureSnapshot struct {
	ID           uuid.UUID              `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	TotalFiles   int                    `json:"total_files"`
	NewFiles     []string               `json:"new_files"`
	DeletedFiles []string               `json:"deleted_files"`
	MovedFiles   map[string]string      `json:"moved_files"`
	SnapshotData map[string]interface{} `json:"snapshot_data"`
}

// Request/Response types
type CreateCampaignRequest struct {
	Name        string                 `json:"name"`
	FromAgent   string                 `json:"from_agent"`
	Description *string                `json:"description,omitempty"`
	Patterns    []string               `json:"patterns"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type VisitRequest struct {
	Files          interface{}            `json:"files"` // Can be []string or []FileVisit
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type FileVisit struct {
	Path    string  `json:"path"`
	Context *string `json:"context,omitempty"`
}

type StructureSyncRequest struct {
	Files         []string               `json:"files,omitempty"`
	Structure     map[string]interface{} `json:"structure,omitempty"`
	Patterns      []string               `json:"patterns,omitempty"`
	RemoveDeleted bool                   `json:"remove_deleted,omitempty"`
}

type SyncResult struct {
	Added      int                    `json:"added"`
	Moved      int                    `json:"moved"`
	Removed    int                    `json:"removed"`
	SnapshotID uuid.UUID              `json:"snapshot_id"`
	Total      int                    `json:"total"`
}

type AdjustVisitRequest struct {
	FileID string `json:"file_id"`
	Action string `json:"action"` // "increment" or "decrement"
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start visited-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Change working directory to project root for file pattern resolution
	// API runs from scenarios/visited-tracker/api/, so go up 3 levels to project root
	if err := os.Chdir("../../../"); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger = log.New(os.Stdout, "[visited-tracker] ", log.LstdFlags)

	// Log current working directory for transparency
	if cwd, err := os.Getwd(); err == nil {
		logger.Printf("📁 Working directory: %s", cwd)
		logger.Printf("💡 File patterns will be resolved relative to this directory")
	}

	// Initialize JSON file storage
	if err := initFileStorage(); err != nil {
		logger.Fatalf("File storage initialization failed: %v", err)
	}

	// Get port from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("❌ API_PORT environment variable is required")
	}

	// Setup router
	r := mux.NewRouter()
	
	// Apply CORS middleware first
	r.Use(corsMiddleware)

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Health endpoint (outside versioning for simplicity)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Campaign management endpoints
	v1.HandleFunc("/campaigns", listCampaignsHandler).Methods("GET")
	v1.HandleFunc("/campaigns", createCampaignHandler).Methods("POST")
	v1.HandleFunc("/campaigns", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}", getCampaignHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}", deleteCampaignHandler).Methods("DELETE")
	v1.HandleFunc("/campaigns/{id}", optionsHandler).Methods("OPTIONS")

	// Campaign-specific visit tracking endpoints
	v1.HandleFunc("/campaigns/{id}/visit", visitHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/visit", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/adjust-visit", adjustVisitHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/adjust-visit", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/structure/sync", structureSyncHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/structure/sync", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/prioritize/least-visited", leastVisitedHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/prioritize/most-stale", mostStaleHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/coverage", coverageHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/export", exportHandler).Methods("GET")

	logger.Printf("🚀 %s API v%s starting on port %s", serviceName, apiVersion, port)
	logger.Printf("📊 Endpoints available at http://localhost:%s/api/v1", port)
	logger.Printf("💾 Data stored in JSON files at: %s", filepath.Join("scenarios", "visited-tracker", dataDir))

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func initFileStorage() error {
	// Ensure data directory exists
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	
	logger.Printf("✅ JSON file storage initialized at: %s", dataPath)
	return nil
}

// File locking utilities for concurrent access
func getFileLock(filename string) *sync.RWMutex {
	locksLock.Lock()
	defer locksLock.Unlock()
	
	if lock, exists := fileLocks[filename]; exists {
		return lock
	}
	
	lock := &sync.RWMutex{}
	fileLocks[filename] = lock
	return lock
}

func getCampaignPath(campaignID uuid.UUID) string {
	return filepath.Join("scenarios", "visited-tracker", dataDir, campaignID.String()+".json")
}

// Campaign storage operations
func saveCampaign(campaign *Campaign) error {
	campaign.UpdatedAt = time.Now().UTC()
	filePath := getCampaignPath(campaign.ID)
	
	lock := getFileLock(filePath)
	lock.Lock()
	defer lock.Unlock()
	
	data, err := json.MarshalIndent(campaign, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal campaign: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write campaign file: %w", err)
	}
	
	return nil
}

func loadCampaign(campaignID uuid.UUID) (*Campaign, error) {
	filePath := getCampaignPath(campaignID)
	
	lock := getFileLock(filePath)
	lock.RLock()
	defer lock.RUnlock()
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("campaign not found")
		}
		return nil, fmt.Errorf("failed to read campaign file: %w", err)
	}
	
	var campaign Campaign
	if err := json.Unmarshal(data, &campaign); err != nil {
		return nil, fmt.Errorf("failed to unmarshal campaign: %w", err)
	}
	
	return &campaign, nil
}

func loadAllCampaigns() ([]Campaign, error) {
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	
	var campaigns []Campaign
	
	// Check if the directory exists first
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// Directory doesn't exist, return empty slice without error
		return campaigns, nil
	}
	
	err := filepath.WalkDir(dataPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Printf("⚠️ Failed to read campaign file %s: %v", path, err)
			return nil // Continue with other files
		}
		
		var campaign Campaign
		if err := json.Unmarshal(data, &campaign); err != nil {
			logger.Printf("⚠️ Failed to unmarshal campaign file %s: %v", path, err)
			return nil // Continue with other files
		}
		
		campaigns = append(campaigns, campaign)
		return nil
	})
	
	return campaigns, err
}

func deleteCampaignFile(campaignID uuid.UUID) error {
	filePath := getCampaignPath(campaignID)
	
	lock := getFileLock(filePath)
	lock.Lock()
	defer lock.Unlock()
	
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete campaign file: %w", err)
	}
	
	return nil
}

// Staleness calculation
func calculateStalenessScore(file *TrackedFile) float64 {
	if file.LastVisited == nil {
		// Never visited files get high staleness based on age
		daysSinceModified := time.Since(file.LastModified).Hours() / 24.0
		return math.Min(100.0, daysSinceModified*2.0)
	}
	
	daysSinceVisit := time.Since(*file.LastVisited).Hours() / 24.0
	daysSinceModified := math.Abs(file.LastVisited.Sub(file.LastModified).Hours() / 24.0)
	
	// Estimate modifications (simplified: assume 1 mod per week if modified after visit)
	modificationsEstimate := 0.0
	if file.LastModified.After(*file.LastVisited) {
		modificationsEstimate = math.Max(1.0, math.Floor(daysSinceModified/7.0))
	}
	
	// Calculate staleness: (modifications × days_since_visit) / (visit_count + 1)
	staleness := (modificationsEstimate * daysSinceVisit) / float64(file.VisitCount+1)
	
	return math.Min(100.0, staleness)
}

// Update staleness scores for all files in campaign
func updateStalenessScores(campaign *Campaign) {
	for i := range campaign.TrackedFiles {
		campaign.TrackedFiles[i].StalenessScore = calculateStalenessScore(&campaign.TrackedFiles[i])
	}
}

// Sync files for a campaign using the specified patterns
func syncCampaignFiles(campaign *Campaign, patterns []string) (*SyncResult, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns specified")
	}
	
	// Log the current working directory and patterns for debugging
	cwd, _ := os.Getwd()
	logger.Printf("🔍 Syncing files for campaign '%s' from directory: %s", campaign.Name, cwd)
	logger.Printf("🔍 Using patterns: %v", patterns)
	
	// Find files matching patterns
	var foundFiles []string
	
	for _, pattern := range patterns {
		logger.Printf("🔍 Processing pattern: %s", pattern)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			logger.Printf("⚠️ Pattern glob failed for %s: %v", pattern, err)
			continue
		}
		
		logger.Printf("🔍 Pattern '%s' found %d matches: %v", pattern, len(matches), matches)
		
		for _, match := range matches {
			// Skip directories
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				foundFiles = append(foundFiles, match)
				logger.Printf("✅ Added file: %s", match)
			} else if err == nil && info.IsDir() {
				logger.Printf("⏭️ Skipped directory: %s", match)
			} else {
				logger.Printf("⚠️ Could not stat file %s: %v", match, err)
			}
		}
	}
	
	logger.Printf("🔍 Total files found before deduplication: %d", len(foundFiles))
	
	// Deduplicate
	fileSet := make(map[string]bool)
	var uniqueFiles []string
	for _, file := range foundFiles {
		abs, _ := filepath.Abs(file)
		if !fileSet[abs] {
			fileSet[abs] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}
	
	logger.Printf("🔍 Unique files after deduplication: %d", len(uniqueFiles))
	
	addedCount := 0
	
	// Add new files to tracked files
	for _, filePath := range uniqueFiles {
		absolutePath, _ := filepath.Abs(filePath)
		
		// Check if already tracked
		found := false
		for _, tracked := range campaign.TrackedFiles {
			if tracked.AbsolutePath == absolutePath {
				found = true
				break
			}
		}
		
		if !found {
			// Get file info
			fileInfo, err := os.Stat(absolutePath)
			var size int64
			var modTime time.Time
			if err == nil {
				size = fileInfo.Size()
				modTime = fileInfo.ModTime()
			} else {
				modTime = time.Now()
				logger.Printf("⚠️ Could not get file info for %s: %v", absolutePath, err)
			}
			
			// Calculate relative path from project root
			relPath, err := filepath.Rel(cwd, absolutePath)
			if err != nil {
				relPath = filePath
				logger.Printf("⚠️ Could not calculate relative path for %s: %v", absolutePath, err)
			}
			
			newFile := TrackedFile{
				ID:           uuid.New(),
				FilePath:     relPath,
				AbsolutePath: absolutePath,
				VisitCount:   0,
				FirstSeen:    time.Now().UTC(),
				LastModified: modTime.UTC(),
				SizeBytes:    size,
				Deleted:      false,
				Metadata:     make(map[string]interface{}),
			}
			
			campaign.TrackedFiles = append(campaign.TrackedFiles, newFile)
			addedCount++
			logger.Printf("➕ Added tracked file: %s (rel: %s)", absolutePath, relPath)
		} else {
			logger.Printf("⏭️ File already tracked: %s", absolutePath)
		}
	}
	
	logger.Printf("📊 Sync results: %d files added, %d total tracked files", addedCount, len(campaign.TrackedFiles))
	
	// Create structure snapshot
	snapshot := StructureSnapshot{
		ID:           uuid.New(),
		Timestamp:    time.Now().UTC(),
		TotalFiles:   len(campaign.TrackedFiles),
		NewFiles:     []string{}, // Could be enhanced to track what was added
		DeletedFiles: []string{}, // Could be enhanced to track what was removed
		MovedFiles:   make(map[string]string),
		SnapshotData: make(map[string]interface{}),
	}
	
	campaign.StructureSnapshots = append(campaign.StructureSnapshots, snapshot)
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	return &SyncResult{
		Added:      addedCount,
		Moved:      0,
		Removed:    0,
		SnapshotID: snapshot.ID,
		Total:      len(campaign.TrackedFiles),
	}, nil
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	// Check file storage health
	dataPath := filepath.Join("scenarios", "visited-tracker", dataDir)
	storageHealthy := true
	var storageError map[string]interface{}
	
	// Test if we can read the data directory
	if _, err := os.Stat(dataPath); err != nil {
		storageHealthy = false
		storageError = map[string]interface{}{
			"code":      "STORAGE_ACCESS_ERROR",
			"message":   fmt.Sprintf("Cannot access data directory: %v", err),
			"category":  "resource",
			"retryable": true,
		}
	}
	
	// Overall service status
	status := "healthy"
	if !storageHealthy {
		status = "degraded"
	}
	
	healthResponse := map[string]interface{}{
		"status":    status,
		"service":   serviceName,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"readiness": true, // Service is ready to accept requests
		"version":   apiVersion,
		"dependencies": map[string]interface{}{
			"storage": map[string]interface{}{
				"connected": storageHealthy,
				"type":      "json-files",
				"path":      dataPath,
			},
		},
		"metrics": map[string]interface{}{
			"uptime_seconds": time.Since(time.Now().Add(-time.Minute)).Seconds(), // Simplified uptime
		},
	}
	
	// Add storage error if present
	if storageError != nil {
		healthResponse["dependencies"].(map[string]interface{})["storage"].(map[string]interface{})["error"] = storageError
	} else {
		healthResponse["dependencies"].(map[string]interface{})["storage"].(map[string]interface{})["error"] = nil
	}
	
	json.NewEncoder(w).Encode(healthResponse)
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers are already set by corsMiddleware
	w.WriteHeader(http.StatusOK)
}

// Campaign handlers
func listCampaignsHandler(w http.ResponseWriter, r *http.Request) {
	campaigns, err := loadAllCampaigns()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaigns: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Calculate computed fields for each campaign
	for i := range campaigns {
		campaign := &campaigns[i]
		updateStalenessScores(campaign)
		
		totalFiles := len(campaign.TrackedFiles)
		visitedFiles := 0
		for _, file := range campaign.TrackedFiles {
			if !file.Deleted && file.VisitCount > 0 {
				visitedFiles++
			}
		}
		
		campaign.TotalFiles = totalFiles
		campaign.VisitedFiles = visitedFiles
		if totalFiles > 0 {
			campaign.CoveragePercent = float64(visitedFiles) / float64(totalFiles) * 100.0
		}
	}
	
	// Sort by created_at desc
	sort.Slice(campaigns, func(i, j int) bool {
		return campaigns[i].CreatedAt.After(campaigns[j].CreatedAt)
	})
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"campaigns": campaigns,
		"count":     len(campaigns),
	})
}

func createCampaignHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	// Validation
	if req.Name == "" {
		http.Error(w, `{"error": "Campaign name is required"}`, http.StatusBadRequest)
		return
	}
	
	if len(req.Patterns) == 0 {
		http.Error(w, `{"error": "At least one file pattern is required"}`, http.StatusBadRequest)
		return
	}
	
	// Check for duplicate names
	campaigns, err := loadAllCampaigns()
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to check for duplicate names: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	for _, campaign := range campaigns {
		if campaign.Name == req.Name {
			http.Error(w, `{"error": "Campaign name already exists"}`, http.StatusConflict)
			return
		}
	}
	
	// Create new campaign
	campaign := Campaign{
		ID:                 uuid.New(),
		Name:               req.Name,
		FromAgent:          req.FromAgent,
		Description:        req.Description,
		Patterns:           req.Patterns,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
		Status:             "active",
		Metadata:           req.Metadata,
		TrackedFiles:       []TrackedFile{},
		Visits:             []Visit{},
		StructureSnapshots: []StructureSnapshot{},
	}
	
	if campaign.Metadata == nil {
		campaign.Metadata = make(map[string]interface{})
	}
	
	// Auto-sync files using campaign patterns
	logger.Printf("🚀 Starting auto-sync for new campaign: %s", campaign.Name)
	syncResult, err := syncCampaignFiles(&campaign, campaign.Patterns)
	if err != nil {
		// Log the error but don't fail campaign creation
		logger.Printf("⚠️ Failed to auto-sync files for campaign %s: %v", campaign.Name, err)
		// Add metadata to indicate sync failed
		campaign.Metadata["auto_sync_error"] = err.Error()
		campaign.Metadata["auto_sync_attempted"] = true
	} else {
		logger.Printf("🔄 Auto-synced %d files for new campaign: %s", syncResult.Added, campaign.Name)
		// Add metadata to track sync success
		campaign.Metadata["auto_sync_files_added"] = syncResult.Added
		campaign.Metadata["auto_sync_attempted"] = true
		campaign.Metadata["auto_sync_success"] = true
	}
	
	// Save to file (includes any synced files)
	if err := saveCampaign(&campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	logger.Printf("✅ Created campaign: %s (ID: %s)", campaign.Name, campaign.ID)
	
	// Calculate computed fields for response
	updateStalenessScores(&campaign)
	totalFiles := len(campaign.TrackedFiles)
	visitedFiles := 0
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted && file.VisitCount > 0 {
			visitedFiles++
		}
	}
	
	campaign.TotalFiles = totalFiles
	campaign.VisitedFiles = visitedFiles
	if totalFiles > 0 {
		campaign.CoveragePercent = float64(visitedFiles) / float64(totalFiles) * 100.0
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

func getCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Update computed fields
	updateStalenessScores(campaign)
	
	totalFiles := len(campaign.TrackedFiles)
	visitedFiles := 0
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted && file.VisitCount > 0 {
			visitedFiles++
		}
	}
	
	campaign.TotalFiles = totalFiles
	campaign.VisitedFiles = visitedFiles
	if totalFiles > 0 {
		campaign.CoveragePercent = float64(visitedFiles) / float64(totalFiles) * 100.0
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaign)
}

func deleteCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	// Check if campaign exists (for logging purposes)
	campaign, err := loadCampaign(campaignID)
	var campaignName string
	if err == nil {
		campaignName = campaign.Name
	}
	
	// Delete the campaign file (idempotent operation)
	if err := deleteCampaignFile(campaignID); err != nil {
		// Only return error if it's not a "file not found" error
		if !os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to delete campaign: %v"}`, err), http.StatusInternalServerError)
			return
		}
	}
	
	if campaignName != "" {
		logger.Printf("🗑️ Deleted campaign: %s (ID: %s)", campaignName, campaignID)
	} else {
		logger.Printf("🗑️ Attempted to delete non-existent campaign (ID: %s) - idempotent operation", campaignID)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": true,
		"id":      campaignID,
	})
}

func visitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	var req VisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Parse files from request
	var filePaths []string
	switch files := req.Files.(type) {
	case []interface{}:
		for _, f := range files {
			if path, ok := f.(string); ok {
				filePaths = append(filePaths, path)
			}
		}
	case []string:
		filePaths = files
	default:
		http.Error(w, `{"error": "Invalid files format"}`, http.StatusBadRequest)
		return
	}
	
	if len(filePaths) == 0 {
		http.Error(w, `{"error": "No files specified"}`, http.StatusBadRequest)
		return
	}
	
	recordedCount := 0
	
	// Record visits for each file
	for _, filePath := range filePaths {
		// Get absolute path
		var absolutePath string
		if filepath.IsAbs(filePath) {
			absolutePath = filePath
		} else {
			cwd, _ := os.Getwd()
			absolutePath = filepath.Join(cwd, filePath)
		}
		
		// Find or create tracked file
		var trackedFile *TrackedFile
		for i := range campaign.TrackedFiles {
			if campaign.TrackedFiles[i].AbsolutePath == absolutePath {
				trackedFile = &campaign.TrackedFiles[i]
				break
			}
		}
		
		if trackedFile == nil {
			// Create new tracked file
			fileInfo, err := os.Stat(absolutePath)
			var size int64
			var modTime time.Time
			if err == nil {
				size = fileInfo.Size()
				modTime = fileInfo.ModTime()
			} else {
				modTime = time.Now()
			}
			
			newFile := TrackedFile{
				ID:           uuid.New(),
				FilePath:     filePath,
				AbsolutePath: absolutePath,
				VisitCount:   0,
				FirstSeen:    time.Now().UTC(),
				LastModified: modTime.UTC(),
				SizeBytes:    size,
				Deleted:      false,
				Metadata:     make(map[string]interface{}),
			}
			
			campaign.TrackedFiles = append(campaign.TrackedFiles, newFile)
			trackedFile = &campaign.TrackedFiles[len(campaign.TrackedFiles)-1]
		}
		
		// Record the visit
		now := time.Now().UTC()
		visit := Visit{
			ID:             uuid.New(),
			FileID:         trackedFile.ID,
			Timestamp:      now,
			Context:        req.Context,
			Agent:          req.Agent,
			ConversationID: req.ConversationID,
			Findings:       req.Metadata,
		}
		
		campaign.Visits = append(campaign.Visits, visit)
		
		// Update tracked file stats
		trackedFile.VisitCount++
		trackedFile.LastVisited = &now
		
		recordedCount++
	}
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Save campaign
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save visits: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	logger.Printf("📝 Recorded %d visits for campaign: %s", recordedCount, campaign.Name)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recorded": recordedCount,
		"files":    filePaths,
	})
}

func adjustVisitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	var req AdjustVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	fileID, err := uuid.Parse(req.FileID)
	if err != nil {
		http.Error(w, `{"error": "Invalid file ID"}`, http.StatusBadRequest)
		return
	}
	
	if req.Action != "increment" && req.Action != "decrement" {
		http.Error(w, `{"error": "Action must be 'increment' or 'decrement'"}`, http.StatusBadRequest)
		return
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Find the file
	var trackedFile *TrackedFile
	for i := range campaign.TrackedFiles {
		if campaign.TrackedFiles[i].ID == fileID {
			trackedFile = &campaign.TrackedFiles[i]
			break
		}
	}
	
	if trackedFile == nil {
		http.Error(w, `{"error": "File not found in campaign"}`, http.StatusNotFound)
		return
	}
	
	// Perform the action
	now := time.Now().UTC()
	var actionSymbol string
	
	if req.Action == "increment" {
		trackedFile.VisitCount++
		trackedFile.LastVisited = &now
		actionSymbol = "➕"
	} else { // decrement
		if trackedFile.VisitCount > 0 {
			trackedFile.VisitCount--
			// If count becomes 0, reset last visited
			if trackedFile.VisitCount == 0 {
				trackedFile.LastVisited = nil
			}
		}
		actionSymbol = "➖"
	}
	
	// Record a manual visit entry
	visit := Visit{
		ID:             uuid.New(),
		FileID:         trackedFile.ID,
		Timestamp:      now,
		Context:        strPtr(fmt.Sprintf("manual-%s", req.Action)),
		Agent:          strPtr("web-ui"),
		ConversationID: nil,
		Findings:       map[string]interface{}{"type": fmt.Sprintf("manual-%s", req.Action)},
	}
	
	campaign.Visits = append(campaign.Visits, visit)
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Save campaign
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save visit adjustment: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	logger.Printf("%s %sed visit count for file %s in campaign: %s", actionSymbol, req.Action, trackedFile.FilePath, campaign.Name)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"file_id":     fileID,
		"visit_count": trackedFile.VisitCount,
		"action":      req.Action,
	})
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}

func structureSyncHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	var req StructureSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Use campaign patterns if none provided
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = campaign.Patterns
	}
	
	if len(patterns) == 0 {
		http.Error(w, `{"error": "No patterns specified"}`, http.StatusBadRequest)
		return
	}
	
	// Use shared sync function
	syncResult, err := syncCampaignFiles(campaign, patterns)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Sync failed: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Save campaign
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save sync results: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	logger.Printf("🔄 Synced %d files for campaign: %s", syncResult.Added, campaign.Name)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(syncResult)
}

func leastVisitedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Filter and sort files
	var files []TrackedFile
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted {
			files = append(files, file)
		}
	}
	
	// Sort by visit count (ascending), then staleness (descending)
	sort.Slice(files, func(i, j int) bool {
		if files[i].VisitCount == files[j].VisitCount {
			return files[i].StalenessScore > files[j].StalenessScore
		}
		return files[i].VisitCount < files[j].VisitCount
	})
	
	// Limit results
	if len(files) > limit {
		files = files[:limit]
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
	})
}

func mostStaleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	
	thresholdStr := r.URL.Query().Get("threshold")
	threshold := 0.0
	if thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = parsed
		}
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Filter and sort files
	var files []TrackedFile
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted && file.StalenessScore >= threshold {
			files = append(files, file)
		}
	}
	
	// Sort by staleness score (descending)
	sort.Slice(files, func(i, j int) bool {
		return files[i].StalenessScore > files[j].StalenessScore
	})
	
	// Limit results
	if len(files) > limit {
		files = files[:limit]
	}
	
	// Calculate critical count (staleness > 50)
	criticalCount := 0
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted && file.StalenessScore > 50 {
			criticalCount++
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":          files,
		"critical_count": criticalCount,
	})
}

func coverageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Calculate coverage stats
	totalFiles := 0
	visitedFiles := 0
	totalVisits := 0
	totalStaleness := 0.0
	
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted {
			totalFiles++
			totalVisits += file.VisitCount
			totalStaleness += file.StalenessScore
			
			if file.VisitCount > 0 {
				visitedFiles++
			}
		}
	}
	
	var averageVisits float64
	var averageStaleness float64
	var coveragePercentage float64
	
	if totalFiles > 0 {
		averageVisits = float64(totalVisits) / float64(totalFiles)
		averageStaleness = totalStaleness / float64(totalFiles)
		coveragePercentage = float64(visitedFiles) / float64(totalFiles) * 100.0
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_files":        totalFiles,
		"visited_files":      visitedFiles,
		"unvisited_files":    totalFiles - visitedFiles,
		"coverage_percentage": math.Round(coveragePercentage*100) / 100,
		"average_visits":     math.Round(averageVisits*100) / 100,
		"average_staleness":  math.Round(averageStaleness*100) / 100,
	})
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}
	
	// Load campaign
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	// Update staleness scores
	updateStalenessScores(campaign)
	
	// Check for patterns filter
	patternsParam := r.URL.Query().Get("patterns")
	if patternsParam != "" {
		patterns := strings.Split(patternsParam, ",")
		filteredFiles := []TrackedFile{}
		
		for _, file := range campaign.TrackedFiles {
			for _, pattern := range patterns {
				pattern = strings.TrimSpace(pattern)
				if matched, _ := filepath.Match(pattern, filepath.Base(file.FilePath)); matched {
					filteredFiles = append(filteredFiles, file)
					break
				}
			}
		}
		
		// Create a copy of the campaign with filtered files
		exportCampaign := *campaign
		exportCampaign.TrackedFiles = filteredFiles
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&exportCampaign)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaign)
}