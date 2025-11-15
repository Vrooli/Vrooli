package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// UpdateTargetsRequest defines the request body for updating operational targets
type UpdateTargetsRequest struct {
	Targets []OperationalTargetUpdate `json:"targets"`
}

// OperationalTargetUpdate represents a target with editable fields
type OperationalTargetUpdate struct {
	ID                 string   `json:"id"`                   // Target ID for matching
	Title              string   `json:"title"`                // Editable title
	Notes              string   `json:"notes"`                // Editable notes
	Status             string   `json:"status"`               // complete | pending
	LinkedRequirements []string `json:"linked_requirement_ids"` // Explicit linkages
}

// UpdateTargetsResponse returns the updated draft content
type UpdateTargetsResponse struct {
	DraftID        string              `json:"draft_id"`
	UpdatedTargets []OperationalTarget `json:"updated_targets"`
	UpdatedContent string              `json:"updated_content"`
	Message        string              `json:"message"`
}

// handleUpdateDraftTargets updates operational targets in a draft's markdown content
//
// PUT /api/v1/drafts/{id}/targets
//
// This endpoint:
// 1. Retrieves the draft by ID
// 2. Parses current operational targets from draft content
// 3. Applies updates from request (title, notes, status, linked_requirement_ids)
// 4. Serializes updated targets back to markdown
// 5. Replaces Functional Requirements section in draft
// 6. Saves updated draft to database and filesystem
func handleUpdateDraftTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body
	var req UpdateTargetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Parse current operational targets from draft content
	currentTargets := parseOperationalTargetsV2(draft.Content, draft.EntityType, draft.EntityName)

	// Build update map for efficient lookup
	updateMap := make(map[string]OperationalTargetUpdate)
	for _, update := range req.Targets {
		updateMap[update.ID] = update
	}

	// Apply updates to matching targets
	updatedTargets := make([]OperationalTarget, 0, len(currentTargets))
	for _, target := range currentTargets {
		if update, found := updateMap[target.ID]; found {
			// Apply updates
			target.Title = update.Title
			target.Notes = update.Notes
			target.Status = update.Status
			target.LinkedRequirements = update.LinkedRequirements
		}
		updatedTargets = append(updatedTargets, target)
	}

	// Handle new targets (IDs not found in current targets)
	for _, update := range req.Targets {
		found := false
		for _, current := range currentTargets {
			if current.ID == update.ID {
				found = true
				break
			}
		}
		if !found {
			// New target - create with defaults
			updatedTargets = append(updatedTargets, OperationalTarget{
				ID:                 update.ID,
				EntityType:         draft.EntityType,
				EntityName:         draft.EntityName,
				Category:           "Must Have", // Default category
				Criticality:        "P0",        // Default criticality
				Title:              update.Title,
				Notes:              update.Notes,
				Status:             update.Status,
				Path:               "Functional Requirements > Must Have > " + update.Title,
				LinkedRequirements: update.LinkedRequirements,
			})
		}
	}

	// Serialize targets back to markdown and update draft content
	updatedContent := updateDraftOperationalTargets(draft.Content, updatedTargets)

	// Save updated draft to database
	if _, err := db.Exec(`
		UPDATE drafts
		SET content = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, updatedContent, draftID); err != nil {
		respondInternalError(w, "Failed to update draft in database", err)
		return
	}

	// Save updated draft to filesystem
	if err := saveDraftToFile(draft.EntityType, draft.EntityName, updatedContent); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	// Return success response
	response := UpdateTargetsResponse{
		DraftID:        draftID,
		UpdatedTargets: updatedTargets,
		UpdatedContent: updatedContent,
		Message:        "Operational targets updated successfully",
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetDraftTargets retrieves operational targets from a draft with v2 parser
//
// GET /api/v1/drafts/{id}/targets
//
// Returns targets with explicit requirement linkages parsed from markdown
func handleGetDraftTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Parse operational targets with v2 parser (supports explicit linkages)
	targets := parseOperationalTargetsV2(draft.Content, draft.EntityType, draft.EntityName)

	response := struct {
		DraftID    string              `json:"draft_id"`
		EntityType string              `json:"entity_type"`
		EntityName string              `json:"entity_name"`
		Targets    []OperationalTarget `json:"targets"`
	}{
		DraftID:    draftID,
		EntityType: draft.EntityType,
		EntityName: draft.EntityName,
		Targets:    targets,
	}

	respondJSON(w, http.StatusOK, response)
}
