package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

// RequirementUpdateRequest represents a request to update a requirement
type RequirementUpdateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Criticality string `json:"criticality"`
	PRDRef      string `json:"prd_ref"`
	Category    string `json:"category"`
}

// handleUpdateRequirement updates a specific requirement in the requirements JSON
func handleUpdateRequirement(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]
	requirementID := vars["requirement_id"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	// Parse request body
	var req RequirementUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	// Get Vrooli root
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "Failed to determine Vrooli root", err)
		return
	}

	// Determine entity directory (scenarios or resources)
	var entityDir string
	if entityType == "scenario" {
		entityDir = "scenarios"
	} else {
		entityDir = "resources"
	}

	// Find the requirement file
	requirementsDir := filepath.Join(vrooliRoot, entityDir, entityName, "requirements")
	requirementFile, err := findRequirementFile(requirementsDir, requirementID)
	if err != nil {
		respondNotFound(w, fmt.Sprintf("Requirement %s not found", requirementID))
		return
	}

	// Read the requirement file
	fullPath := filepath.Join(requirementsDir, requirementFile)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		respondInternalError(w, "Failed to read requirement file", err)
		return
	}

	// Parse JSON
	var reqData map[string]any
	if err := json.Unmarshal(data, &reqData); err != nil {
		respondInternalError(w, "Failed to parse requirement file", err)
		return
	}

	// Update the requirement
	updated := false
	if err := updateRequirementInData(reqData, requirementID, req, &updated); err != nil {
		respondInternalError(w, "Failed to update requirement", err)
		return
	}

	if !updated {
		respondNotFound(w, fmt.Sprintf("Requirement %s not found in file", requirementID))
		return
	}

	// Write back to file
	updatedData, err := json.MarshalIndent(reqData, "", "  ")
	if err != nil {
		respondInternalError(w, "Failed to marshal updated data", err)
		return
	}

	if err := os.WriteFile(fullPath, updatedData, 0644); err != nil {
		respondInternalError(w, "Failed to write requirement file", err)
		return
	}

	slog.Info("Requirement updated", "id", requirementID, "file", requirementFile)

	respondJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Requirement updated successfully",
		"id":      requirementID,
	})
}

// findRequirementFile searches for the JSON file containing the requirement
func findRequirementFile(requirementsDir string, requirementID string) (string, error) {
	entries, err := os.ReadDir(requirementsDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read and check if this file contains the requirement
		fullPath := filepath.Join(requirementsDir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var reqData map[string]any
		if err := json.Unmarshal(data, &reqData); err != nil {
			continue
		}

		if containsRequirement(reqData, requirementID) {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("requirement not found")
}

// containsRequirement checks if a requirement ID exists in the data structure
func containsRequirement(data map[string]any, requirementID string) bool {
	if groups, ok := data["groups"].([]any); ok {
		for _, group := range groups {
			if g, ok := group.(map[string]any); ok {
				if hasRequirementInGroup(g, requirementID) {
					return true
				}
			}
		}
	}
	return false
}

// hasRequirementInGroup recursively checks if a group contains the requirement
func hasRequirementInGroup(group map[string]any, requirementID string) bool {
	if requirements, ok := group["requirements"].([]any); ok {
		for _, req := range requirements {
			if r, ok := req.(map[string]any); ok {
				if id, ok := r["id"].(string); ok && id == requirementID {
					return true
				}
			}
		}
	}

	if children, ok := group["children"].([]any); ok {
		for _, child := range children {
			if c, ok := child.(map[string]any); ok {
				if hasRequirementInGroup(c, requirementID) {
					return true
				}
			}
		}
	}

	return false
}

// updateRequirementInData recursively updates a requirement in the data structure
func updateRequirementInData(data map[string]any, requirementID string, req RequirementUpdateRequest, updated *bool) error {
	if groups, ok := data["groups"].([]any); ok {
		for _, group := range groups {
			if g, ok := group.(map[string]any); ok {
				updateRequirementInGroup(g, requirementID, req, updated)
				if *updated {
					return nil
				}
			}
		}
	}
	return nil
}

// updateRequirementInGroup recursively updates a requirement in a group
func updateRequirementInGroup(group map[string]any, requirementID string, req RequirementUpdateRequest, updated *bool) {
	if requirements, ok := group["requirements"].([]any); ok {
		for i, reqItem := range requirements {
			if r, ok := reqItem.(map[string]any); ok {
				if id, ok := r["id"].(string); ok && id == requirementID {
					// Update fields
					r["title"] = req.Title
					r["description"] = req.Description
					r["status"] = req.Status
					r["criticality"] = req.Criticality
					r["prd_ref"] = req.PRDRef
					r["category"] = req.Category

					requirements[i] = r
					group["requirements"] = requirements
					*updated = true
					return
				}
			}
		}
	}

	if children, ok := group["children"].([]any); ok {
		for _, child := range children {
			if c, ok := child.(map[string]any); ok {
				updateRequirementInGroup(c, requirementID, req, updated)
				if *updated {
					return
				}
			}
		}
	}
}
