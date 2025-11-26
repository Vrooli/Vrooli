package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// strPtr returns a pointer to the given string
func strPtr(s string) *string {
	return &s
}

// loadAndPrepareCampaign loads a campaign and updates staleness scores
func loadAndPrepareCampaign(campaignID uuid.UUID, w http.ResponseWriter) (*Campaign, bool) {
	campaign, err := loadCampaign(campaignID)
	if err != nil {
		if err.Error() == "campaign not found" {
			http.Error(w, `{"error": "Campaign not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to load campaign: %v"}`, err), http.StatusInternalServerError)
		}
		return nil, false
	}

	// Update staleness scores before returning
	updateStalenessScores(campaign)
	return campaign, true
}

// parseLimit parses the limit query parameter with a default value
func parseLimit(r *http.Request, defaultLimit int) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return defaultLimit
	}
	if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
		return parsed
	}
	return defaultLimit
}

// parseThreshold parses the threshold query parameter with a default value
func parseThreshold(r *http.Request, defaultThreshold float64) float64 {
	thresholdStr := r.URL.Query().Get("threshold")
	if thresholdStr == "" {
		return defaultThreshold
	}
	if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
		return parsed
	}
	return defaultThreshold
}

// getNonDeletedFiles filters out deleted files from a campaign
func getNonDeletedFiles(campaign *Campaign) []TrackedFile {
	var files []TrackedFile
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted {
			files = append(files, file)
		}
	}
	return files
}

// getActiveFiles filters out deleted AND excluded files from a campaign
func getActiveFiles(campaign *Campaign) []TrackedFile {
	var files []TrackedFile
	for _, file := range campaign.TrackedFiles {
		if !file.Deleted && !file.Excluded {
			files = append(files, file)
		}
	}
	return files
}
