package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func updateFileNotesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]
	fileID := vars["file_id"]

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

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		http.Error(w, `{"error": "Invalid file ID"}`, http.StatusBadRequest)
		return
	}

	found := false
	for i := range campaign.TrackedFiles {
		if campaign.TrackedFiles[i].ID == fileUUID {
			campaign.TrackedFiles[i].Notes = updates.Notes
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error": "File not found in campaign"}`, http.StatusNotFound)
		return
	}

	campaign.UpdatedAt = time.Now().UTC()
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "File notes updated successfully",
	})
}
func updateFilePriorityHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]
	fileID := vars["file_id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	var updates struct {
		PriorityWeight float64 `json:"priority_weight"`
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

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		http.Error(w, `{"error": "Invalid file ID"}`, http.StatusBadRequest)
		return
	}

	found := false
	for i := range campaign.TrackedFiles {
		if campaign.TrackedFiles[i].ID == fileUUID {
			campaign.TrackedFiles[i].PriorityWeight = updates.PriorityWeight
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error": "File not found in campaign"}`, http.StatusNotFound)
		return
	}

	campaign.UpdatedAt = time.Now().UTC()
	updateStalenessScores(campaign)

	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "File priority updated successfully",
	})
}
func toggleFileExclusionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]
	fileID := vars["file_id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	var updates struct {
		Excluded bool `json:"excluded"`
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

	fileUUID, err := uuid.Parse(fileID)
	if err != nil {
		http.Error(w, `{"error": "Invalid file ID"}`, http.StatusBadRequest)
		return
	}

	found := false
	for i := range campaign.TrackedFiles {
		if campaign.TrackedFiles[i].ID == fileUUID {
			campaign.TrackedFiles[i].Excluded = updates.Excluded
			found = true
			break
		}
	}

	if !found {
		http.Error(w, `{"error": "File not found in campaign"}`, http.StatusNotFound)
		return
	}

	campaign.UpdatedAt = time.Now().UTC()
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "File exclusion updated successfully",
	})
}

// getFileByPathHandler - GET /api/v1/campaigns/{id}/files/by-path?path=...
func getFileByPathHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	filePath := r.URL.Query().Get("path")
	if filePath == "" {
		http.Error(w, `{"error": "path query parameter is required"}`, http.StatusBadRequest)
		return
	}

	campaign, err := loadCampaign(campaignID)
	if err != nil {
		http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
		return
	}

	// Normalize the path for comparison
	var absPath string
	if filepath.IsAbs(filePath) {
		absPath = filepath.Clean(filePath)
	} else {
		cwd, _ := os.Getwd()
		absPath = filepath.Clean(filepath.Join(cwd, filePath))
	}

	// Search for file by path or absolute path
	for _, file := range campaign.TrackedFiles {
		if file.FilePath == filePath || file.AbsolutePath == absPath || file.FilePath == absPath {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(file)
			return
		}
	}

	http.Error(w, `{"error": "File not found in campaign"}`, http.StatusNotFound)
}

// bulkExcludeFilesHandler - POST /api/v1/campaigns/{id}/files/exclude
func bulkExcludeFilesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignIDStr := vars["id"]

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	var req BulkExcludeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if len(req.Files) == 0 {
		http.Error(w, `{"error": "files array cannot be empty"}`, http.StatusBadRequest)
		return
	}

	campaign, err := loadCampaign(campaignID)
	if err != nil {
		http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
		return
	}

	// Normalize paths using campaign location
	normalizedPaths := make(map[string]string) // maps original path -> absolute path
	for _, filePath := range req.Files {
		_, absPath := normalizeFilePath(campaign, filePath)
		normalizedPaths[filePath] = absPath
	}

	excludedCount := 0
	updatedFiles := make([]string, 0)

	// Update files
	for _, filePath := range req.Files {
		absPath := normalizedPaths[filePath]
		found := false

		for i := range campaign.TrackedFiles {
			file := &campaign.TrackedFiles[i]
			if file.AbsolutePath == absPath {
				file.Excluded = req.Excluded
				if req.Reason != nil {
					file.Notes = req.Reason
				}
				excludedCount++
				updatedFiles = append(updatedFiles, file.FilePath)
				found = true
				break
			}
		}

		if !found {
			logger.Printf("⚠️  File not found in campaign: %s (normalized to: %s)", filePath, absPath)
		}
	}

	campaign.UpdatedAt = time.Now().UTC()
	if err := saveCampaign(campaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	logger.Printf("✅ Bulk excluded %d files in campaign: %s", excludedCount, campaign.Name)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"excluded_count": excludedCount,
		"files":          updatedFiles,
	})
}

// resetCampaignHandler - POST /api/v1/campaigns/{id}/reset
