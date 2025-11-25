package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

// UpdateTargetsRequest defines the request body for updating operational targets
type UpdateTargetsRequest struct {
	Targets []OperationalTargetUpdate `json:"targets"`
}

// OperationalTargetUpdate represents a target with editable fields
type OperationalTargetUpdate struct {
	ID                 string   `json:"id"`                     // Target ID for matching
	Title              string   `json:"title"`                  // Editable title
	Notes              string   `json:"notes"`                  // Editable notes
	Status             string   `json:"status"`                 // complete | pending
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
	currentTargets := parseOperationalTargets(draft.Content, draft.EntityType, draft.EntityName)

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
			criticality := "P0"
			title := update.Title
			if title == "" {
				title = "Target"
			}
			pathID := update.ID
			if pathID == "" {
				pathID = title
			}
			updatedTargets = append(updatedTargets, OperationalTarget{
				ID:                 update.ID,
				EntityType:         draft.EntityType,
				EntityName:         draft.EntityName,
				Category:           criticality,
				Criticality:        criticality,
				Title:              title,
				Notes:              update.Notes,
				Status:             update.Status,
				Path:               fmt.Sprintf("Operational Targets > %s > %s", criticality, pathID),
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

	// Parse operational targets (supports explicit linkages)
	targets := parseOperationalTargets(draft.Content, draft.EntityType, draft.EntityName)

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

// CreateOperationalTargetRequest defines the request body for creating a new operational target
type CreateOperationalTargetRequest struct {
	Title              string   `json:"title"`
	Notes              string   `json:"notes"`
	Category           string   `json:"category"`            // Must Have | Should Have | Nice to Have
	Criticality        string   `json:"criticality"`         // P0 | P1 | P2
	Status             string   `json:"status"`              // complete | pending
	LinkedRequirements []string `json:"linked_requirement_ids"`
}

// UpdateOperationalTargetRequest defines the request body for updating an operational target
type UpdateOperationalTargetRequest struct {
	Title              string   `json:"title"`
	Notes              string   `json:"notes"`
	Status             string   `json:"status"`
	LinkedRequirements []string `json:"linked_requirement_ids"`
}

// handleCreateOperationalTarget creates a new operational target in the published PRD
//
// POST /api/v1/catalog/{type}/{name}/targets
//
// This endpoint creates a draft from the published PRD (if not exists),
// adds the new target, and returns the draft ID for further editing.
func handleCreateOperationalTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	// Parse request body
	var req CreateOperationalTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	// Validate required fields
	if req.Title == "" {
		respondBadRequest(w, "Title is required")
		return
	}
	if req.Category == "" {
		req.Category = "Must Have"
	}
	if req.Criticality == "" {
		req.Criticality = "P0"
	}
	if req.Status == "" {
		req.Status = "pending"
	}

	// Get or create draft
	draft, err := getOrCreateDraft(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to get or create draft", err)
		return
	}

	// Parse current operational targets
	targets := parseOperationalTargets(draft.Content, entityType, entityName)

	// Generate new target ID (use title as base, ensure uniqueness)
	newID := generateTargetID(req.Title, targets)

	// Create new target
	newTarget := OperationalTarget{
		ID:                 newID,
		EntityType:         entityType,
		EntityName:         entityName,
		Category:           req.Category,
		Criticality:        req.Criticality,
		Title:              req.Title,
		Notes:              req.Notes,
		Status:             req.Status,
		Path:               fmt.Sprintf("Operational Targets > %s > %s", req.Criticality, newID),
		LinkedRequirements: req.LinkedRequirements,
	}

	// Append new target
	targets = append(targets, newTarget)

	// Update draft content
	updatedContent := updateDraftOperationalTargets(draft.Content, targets)

	// Save updated draft
	if _, err := db.Exec(`
		UPDATE drafts
		SET content = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, updatedContent, draft.ID); err != nil {
		respondInternalError(w, "Failed to update draft", err)
		return
	}

	if err := saveDraftToFile(entityType, entityName, updatedContent); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"success":   true,
		"message":   "Operational target created successfully",
		"target":    newTarget,
		"draft_id":  draft.ID,
		"draft_url": fmt.Sprintf("/draft/%s/%s", entityType, entityName),
	})
}

// handleUpdateOperationalTarget updates an existing operational target in the published PRD
//
// PUT /api/v1/catalog/{type}/{name}/targets/{target_id}
//
// This endpoint creates a draft from the published PRD (if not exists),
// updates the target, and returns the draft ID for further editing.
func handleUpdateOperationalTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]
	targetID := vars["target_id"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	// Parse request body
	var req UpdateOperationalTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	// Get or create draft
	draft, err := getOrCreateDraft(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to get or create draft", err)
		return
	}

	// Parse current operational targets
	targets := parseOperationalTargets(draft.Content, entityType, entityName)

	// Find and update the target
	found := false
	for i, target := range targets {
		if target.ID == targetID {
			if req.Title != "" {
				targets[i].Title = req.Title
			}
			if req.Notes != "" {
				targets[i].Notes = req.Notes
			}
			if req.Status != "" {
				targets[i].Status = req.Status
			}
			if req.LinkedRequirements != nil {
				targets[i].LinkedRequirements = req.LinkedRequirements
			}
			found = true
			break
		}
	}

	if !found {
		respondNotFound(w, fmt.Sprintf("Operational target %s not found", targetID))
		return
	}

	// Update draft content
	updatedContent := updateDraftOperationalTargets(draft.Content, targets)

	// Save updated draft
	if _, err := db.Exec(`
		UPDATE drafts
		SET content = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, updatedContent, draft.ID); err != nil {
		respondInternalError(w, "Failed to update draft", err)
		return
	}

	if err := saveDraftToFile(entityType, entityName, updatedContent); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"message":   "Operational target updated successfully",
		"target_id": targetID,
		"draft_id":  draft.ID,
		"draft_url": fmt.Sprintf("/draft/%s/%s", entityType, entityName),
	})
}

// handleDeleteOperationalTarget deletes an operational target from the published PRD
//
// DELETE /api/v1/catalog/{type}/{name}/targets/{target_id}
//
// This endpoint creates a draft from the published PRD (if not exists),
// removes the target, and returns the draft ID for further editing.
func handleDeleteOperationalTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]
	targetID := vars["target_id"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	// Get or create draft
	draft, err := getOrCreateDraft(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to get or create draft", err)
		return
	}

	// Parse current operational targets
	targets := parseOperationalTargets(draft.Content, entityType, entityName)

	// Find and remove the target
	found := false
	updatedTargets := make([]OperationalTarget, 0, len(targets))
	for _, target := range targets {
		if target.ID == targetID {
			found = true
			continue // Skip this target (delete it)
		}
		updatedTargets = append(updatedTargets, target)
	}

	if !found {
		respondNotFound(w, fmt.Sprintf("Operational target %s not found", targetID))
		return
	}

	// Update draft content
	updatedContent := updateDraftOperationalTargets(draft.Content, updatedTargets)

	// Save updated draft
	if _, err := db.Exec(`
		UPDATE drafts
		SET content = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, updatedContent, draft.ID); err != nil {
		respondInternalError(w, "Failed to update draft", err)
		return
	}

	if err := saveDraftToFile(entityType, entityName, updatedContent); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"success":   true,
		"message":   "Operational target deleted successfully",
		"target_id": targetID,
		"draft_id":  draft.ID,
		"draft_url": fmt.Sprintf("/draft/%s/%s", entityType, entityName),
	})
}

// generateTargetID creates a unique ID from a title
func generateTargetID(title string, existingTargets []OperationalTarget) string {
	// Convert title to ID format (lowercase, hyphenated)
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	baseID := strings.ToLower(reg.ReplaceAllString(title, "-"))
	baseID = strings.Trim(baseID, "-")

	// Check if ID already exists
	idMap := make(map[string]bool)
	for _, target := range existingTargets {
		idMap[target.ID] = true
	}

	// If base ID is unique, use it
	if !idMap[baseID] {
		return baseID
	}

	// Otherwise, append a number
	counter := 2
	for {
		candidateID := fmt.Sprintf("%s-%d", baseID, counter)
		if !idMap[candidateID] {
			return candidateID
		}
		counter++
	}
}

// getOrCreateDraft gets an existing draft or creates one from the published PRD
func getOrCreateDraft(entityType, entityName string) (*Draft, error) {
	// Try to get existing draft
	var draft Draft
	err := db.QueryRow(`
		SELECT id, entity_type, entity_name, content, status, created_at, updated_at
		FROM drafts
		WHERE entity_type = $1 AND entity_name = $2 AND status = $3
		ORDER BY updated_at DESC
		LIMIT 1
	`, entityType, entityName, DraftStatusDraft).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&draft.Status,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	)

	if err == nil {
		// Draft exists
		return &draft, nil
	}

	// Draft doesn't exist, create one from published PRD
	prdContent, err := loadPublishedPRD(entityType, entityName)
	if err != nil {
		return nil, fmt.Errorf("failed to load published PRD: %w", err)
	}

	// Generate draft ID
	draftID := fmt.Sprintf("%s-%s", entityType, entityName)

	// Insert draft into database
	err = db.QueryRow(`
		INSERT INTO drafts (id, entity_type, entity_name, content, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, entity_type, entity_name, content, status, created_at, updated_at
	`, draftID, entityType, entityName, prdContent, DraftStatusDraft).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&draft.Status,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	// Save draft to filesystem
	if err := saveDraftToFile(entityType, entityName, prdContent); err != nil {
		return nil, fmt.Errorf("failed to save draft file: %w", err)
	}

	return &draft, nil
}

// loadPublishedPRD loads the content of a published PRD
func loadPublishedPRD(entityType, entityName string) (string, error) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get Vrooli root: %w", err)
	}

	prdPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "PRD.md")
	content, err := os.ReadFile(prdPath)
	if err != nil {
		return "", fmt.Errorf("failed to read PRD file: %w", err)
	}

	return string(content), nil
}
