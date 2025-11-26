package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	apiVersion = "3.0.0"
	serviceName = "visited-tracker"
	defaultMaxFiles = 200
)

var defaultExcludePatterns = []string{
	"**/data/**",
	"**/tmp/**",
	"**/temp/**",
	"**/coverage/**",
	"**/dist/**",
	"**/out/**",
	"**/build/**",
	"**/.git/**",
	"**/node_modules/**",
	"**/__pycache__/**",
}

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

	// Apply defaults
	maxFiles := req.MaxFiles
	if maxFiles == 0 {
		maxFiles = defaultMaxFiles
	}

	excludePatterns := req.ExcludePatterns
	if len(excludePatterns) == 0 {
		excludePatterns = defaultExcludePatterns
	}

	// Create new campaign
	campaign := Campaign{
		ID:                 uuid.New(),
		Name:               req.Name,
		FromAgent:          req.FromAgent,
		Description:        req.Description,
		Patterns:           req.Patterns,
		Location:           req.Location,
		Tag:                req.Tag,
		Notes:              req.Notes,
		MaxFiles:           maxFiles,
		ExcludePatterns:    excludePatterns,
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
func findOrCreateCampaignHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	// If location and tag are provided, try to find existing campaign
	if req.Location != nil && req.Tag != nil {
		campaigns, err := loadAllCampaigns()
		if err == nil {
			for _, campaign := range campaigns {
				if campaign.Location != nil && campaign.Tag != nil &&
					*campaign.Location == *req.Location && *campaign.Tag == *req.Tag {
					// Found existing campaign with matching location+tag
					logger.Printf("🔍 Found existing campaign for location=%s, tag=%s: %s", *req.Location, *req.Tag, campaign.Name)
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"created":  false,
						"campaign": campaign,
					})
					return
				}
			}
		}
	}

	// No matching campaign found, create new one
	logger.Printf("🆕 Creating new campaign with location=%v, tag=%v", req.Location, req.Tag)

	// Generate name from location+tag if not provided
	if req.Name == "" {
		if req.Location != nil && req.Tag != nil {
			req.Name = fmt.Sprintf("%s-%s", *req.Location, *req.Tag)
		} else {
			http.Error(w, `{"error": "Campaign name is required when location/tag not provided"}`, http.StatusBadRequest)
			return
		}
	}

	if len(req.Patterns) == 0 {
		http.Error(w, `{"error": "At least one file pattern is required"}`, http.StatusBadRequest)
		return
	}

	// Apply defaults
	maxFiles := req.MaxFiles
	if maxFiles == 0 {
		maxFiles = defaultMaxFiles
	}

	excludePatterns := req.ExcludePatterns
	if len(excludePatterns) == 0 {
		excludePatterns = defaultExcludePatterns
	}

	campaign := Campaign{
		ID:                 uuid.New(),
		Name:               req.Name,
		FromAgent:          req.FromAgent,
		Description:        req.Description,
		Patterns:           req.Patterns,
		Location:           req.Location,
		Tag:                req.Tag,
		Notes:              req.Notes,
		MaxFiles:           maxFiles,
		ExcludePatterns:    excludePatterns,
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

	// Auto-sync files
	syncResult, err := syncCampaignFiles(&campaign, campaign.Patterns)
	if err != nil {
		logger.Printf("⚠️ Failed to auto-sync files: %v", err)
		campaign.Metadata["auto_sync_error"] = err.Error()
	} else {
		logger.Printf("🔄 Auto-synced %d files", syncResult.Added)
		campaign.Metadata["auto_synced"] = true
		campaign.Metadata["files_added"] = syncResult.Added
	}

	if err := saveCampaign(&campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"created":  true,
		"campaign": campaign,
	})
}

func updateCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	var updates struct {
		Notes *string `json:"notes,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	campaign, err := loadCampaign(campaignID)
	if err != nil {
		http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
		return
	}

	if updates.Notes != nil {
		campaign.Notes = updates.Notes
		campaign.UpdatedAt = time.Now().UTC()

		if err := saveCampaign(campaign); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaign)
}

func resetCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	campaign, err := loadCampaign(campaignID)
	if err != nil {
		http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
		return
	}

	// Clear all visits
	campaign.Visits = []Visit{}

	// Reset visit counts and last visited timestamps on all files
	for i := range campaign.TrackedFiles {
		campaign.TrackedFiles[i].VisitCount = 0
		campaign.TrackedFiles[i].LastVisited = nil
	}

	// Update staleness scores
	updateStalenessScores(campaign)

	campaign.UpdatedAt = time.Now().UTC()
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	logger.Printf("🔄 Reset campaign: %s (ID: %s)", campaign.Name, campaign.ID)

	// Calculate computed fields for response
	totalFiles := len(campaign.TrackedFiles)
	campaign.TotalFiles = totalFiles
	campaign.VisitedFiles = 0
	campaign.CoveragePercent = 0.0

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Campaign reset successfully",
		"campaign": campaign,
	})
}
