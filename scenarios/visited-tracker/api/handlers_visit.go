package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

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
		// Normalize path using campaign location
		relativePath, absolutePath := normalizeFilePath(campaign, filePath)

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
				FilePath:     relativePath,
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

	// Process file notes if provided
	if req.FileNotes != nil && len(req.FileNotes) > 0 {
		for filePath, note := range req.FileNotes {
			// Normalize the path using campaign location
			_, absPath := normalizeFilePath(campaign, filePath)

			// Find and update the file's notes
			for i := range campaign.TrackedFiles {
				file := &campaign.TrackedFiles[i]
				if file.AbsolutePath == absPath {
					file.Notes = &note
					break
				}
			}
		}
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
