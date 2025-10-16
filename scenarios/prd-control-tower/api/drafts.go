package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Draft represents a PRD draft with metadata
type Draft struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityName string    `json:"entity_name"`
	Content    string    `json:"content"`
	Owner      string    `json:"owner,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Status     string    `json:"status"`
}

// CreateDraftRequest represents the request to create a new draft
type CreateDraftRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Content    string `json:"content"`
	Owner      string `json:"owner,omitempty"`
}

// UpdateDraftRequest represents the request to update a draft
type UpdateDraftRequest struct {
	Content string `json:"content"`
}

// DraftListResponse represents a list of drafts
type DraftListResponse struct {
	Drafts []Draft `json:"drafts"`
	Total  int     `json:"total"`
}

func handleListDrafts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	rows, err := db.Query(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		ORDER BY updated_at DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query drafts: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	drafts := []Draft{}
	for rows.Next() {
		var draft Draft
		var owner sql.NullString

		err := rows.Scan(
			&draft.ID,
			&draft.EntityType,
			&draft.EntityName,
			&draft.Content,
			&owner,
			&draft.CreatedAt,
			&draft.UpdatedAt,
			&draft.Status,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan draft: %v", err), http.StatusInternalServerError)
			return
		}

		if owner.Valid {
			draft.Owner = owner.String
		}

		drafts = append(drafts, draft)
	}

	response := DraftListResponse{
		Drafts: drafts,
		Total:  len(drafts),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	var draft Draft
	var owner sql.NullString

	err := db.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		WHERE id = $1
	`, draftID).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&owner,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get draft: %v", err), http.StatusInternalServerError)
		return
	}

	if owner.Valid {
		draft.Owner = owner.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(draft)
}

func handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate entity type
	if req.EntityType != "scenario" && req.EntityType != "resource" {
		http.Error(w, "Invalid entity_type. Must be 'scenario' or 'resource'", http.StatusBadRequest)
		return
	}

	// Validate entity name
	if req.EntityName == "" {
		http.Error(w, "entity_name is required", http.StatusBadRequest)
		return
	}

	// Generate UUID
	draftID := uuid.New().String()
	now := time.Now()

	// Insert into database
	_, err := db.Exec(`
		INSERT INTO drafts (id, entity_type, entity_name, content, owner, created_at, updated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (entity_type, entity_name)
		DO UPDATE SET content = $4, owner = $5, updated_at = $7
		RETURNING id
	`, draftID, req.EntityType, req.EntityName, req.Content, nullString(req.Owner), now, now, "draft")

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Write draft to filesystem
	if err := saveDraftToFile(req.EntityType, req.EntityName, req.Content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save draft file: %v", err), http.StatusInternalServerError)
		return
	}

	draft := Draft{
		ID:         draftID,
		EntityType: req.EntityType,
		EntityName: req.EntityName,
		Content:    req.Content,
		Owner:      req.Owner,
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     "draft",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(draft)
}

func handleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Get draft metadata
	var draft Draft
	var owner sql.NullString
	err := db.QueryRow(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		WHERE id = $1
	`, draftID).Scan(
		&draft.ID,
		&draft.EntityType,
		&draft.EntityName,
		&draft.Content,
		&owner,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.Status,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Update database
	now := time.Now()
	_, err = db.Exec(`
		UPDATE drafts
		SET content = $1, updated_at = $2
		WHERE id = $3
	`, req.Content, now, draftID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Update filesystem
	if err := saveDraftToFile(draft.EntityType, draft.EntityName, req.Content); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save draft file: %v", err), http.StatusInternalServerError)
		return
	}

	draft.Content = req.Content
	draft.UpdatedAt = now

	if owner.Valid {
		draft.Owner = owner.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(draft)
}

func handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Get draft metadata before deletion
	var entityType, entityName string
	err := db.QueryRow(`
		SELECT entity_type, entity_name
		FROM drafts
		WHERE id = $1
	`, draftID).Scan(&entityType, &entityName)

	if err == sql.ErrNoRows {
		http.Error(w, "Draft not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete from database
	_, err = db.Exec(`DELETE FROM drafts WHERE id = $1`, draftID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete draft: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete from filesystem
	draftPath := getDraftPath(entityType, entityName)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		http.Error(w, fmt.Sprintf("Failed to delete draft file: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func saveDraftToFile(entityType string, entityName string, content string) error {
	draftPath := getDraftPath(entityType, entityName)

	// Create directory if it doesn't exist
	draftDir := filepath.Dir(draftPath)
	if err := os.MkdirAll(draftDir, 0755); err != nil {
		return fmt.Errorf("failed to create draft directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(draftPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write draft file: %w", err)
	}

	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
