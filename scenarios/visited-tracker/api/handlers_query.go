package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func leastVisitedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid campaign ID"}`, http.StatusBadRequest)
		return
	}

	// Parse limit with default of 10
	limit := parseLimit(r, 10)

	// Load and prepare campaign
	campaign, ok := loadAndPrepareCampaign(campaignID, w)
	if !ok {
		return
	}

	// Filter deleted and excluded files
	files := getActiveFiles(campaign)

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

	// Parse query parameters with defaults
	limit := parseLimit(r, 10)
	threshold := parseThreshold(r, 0.0)

	// Load and prepare campaign
	campaign, ok := loadAndPrepareCampaign(campaignID, w)
	if !ok {
		return
	}

	// Filter files by staleness threshold and deletion status
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
		"total_files":         totalFiles,
		"visited_files":       visitedFiles,
		"unvisited_files":     totalFiles - visitedFiles,
		"coverage_percentage": math.Round(coveragePercentage*100) / 100,
		"average_visits":      math.Round(averageVisits*100) / 100,
		"average_staleness":   math.Round(averageStaleness*100) / 100,
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

func importHandler(w http.ResponseWriter, r *http.Request) {
	var importedCampaign Campaign

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&importedCampaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Invalid campaign data: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if importedCampaign.Name == "" {
		http.Error(w, `{"error": "Campaign name is required"}`, http.StatusBadRequest)
		return
	}
	if len(importedCampaign.Patterns) == 0 {
		http.Error(w, `{"error": "At least one pattern is required"}`, http.StatusBadRequest)
		return
	}

	// Check if merge mode is enabled
	mergeParam := r.URL.Query().Get("merge")
	merge := mergeParam == "true"

	// If merging, try to find existing campaign by name
	var existingCampaign *Campaign
	if merge {
		campaigns, err := loadAllCampaigns()
		if err == nil {
			for _, c := range campaigns {
				if c.Name == importedCampaign.Name {
					existingCampaign = &c
					break
				}
			}
		}
	}

	if existingCampaign != nil {
		// Merge mode: Update existing campaign
		existingCampaign.Patterns = importedCampaign.Patterns
		if importedCampaign.Description != nil {
			existingCampaign.Description = importedCampaign.Description
		}
		if importedCampaign.FromAgent != "" {
			existingCampaign.FromAgent = importedCampaign.FromAgent
		}

		// Merge tracked files (add new ones, update existing ones)
		fileMap := make(map[string]*TrackedFile)
		for i := range existingCampaign.TrackedFiles {
			fileMap[existingCampaign.TrackedFiles[i].FilePath] = &existingCampaign.TrackedFiles[i]
		}

		for _, importedFile := range importedCampaign.TrackedFiles {
			if existing, exists := fileMap[importedFile.FilePath]; exists {
				// Update visit count (take maximum)
				if importedFile.VisitCount > existing.VisitCount {
					existing.VisitCount = importedFile.VisitCount
				}
				// Update last visit time (take most recent)
				if importedFile.LastVisited != nil && (existing.LastVisited == nil || importedFile.LastVisited.After(*existing.LastVisited)) {
					existing.LastVisited = importedFile.LastVisited
				}
			} else {
				// Add new file
				importedFile.ID = uuid.New()
				existingCampaign.TrackedFiles = append(existingCampaign.TrackedFiles, importedFile)
			}
		}

		// Merge visits
		existingCampaign.Visits = append(existingCampaign.Visits, importedCampaign.Visits...)

		// Update metadata
		existingCampaign.UpdatedAt = time.Now()
		existingCampaign.Metadata["last_import"] = time.Now().Format(time.RFC3339)

		// Save updated campaign
		if err := saveCampaign(existingCampaign); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to save merged campaign: %v"}`, err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":  "Campaign merged successfully",
			"campaign": existingCampaign,
		})
		return
	}

	// Create new campaign from imported data
	now := time.Now()
	importedCampaign.ID = uuid.New()
	importedCampaign.CreatedAt = now
	importedCampaign.UpdatedAt = now
	importedCampaign.Status = "active"

	// Generate new IDs for tracked files
	for i := range importedCampaign.TrackedFiles {
		importedCampaign.TrackedFiles[i].ID = uuid.New()
	}

	// Initialize metadata if nil
	if importedCampaign.Metadata == nil {
		importedCampaign.Metadata = make(map[string]interface{})
	}
	importedCampaign.Metadata["imported_at"] = now.Format(time.RFC3339)

	// Save campaign
	if err := saveCampaign(&importedCampaign); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to save campaign: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Campaign imported successfully",
		"campaign": importedCampaign,
	})
}

