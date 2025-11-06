package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var ErrDatabaseNotAvailable = errors.New("database not available")

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
	// Defensive check for unit tests (middleware protects in production)
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	// Sync filesystem drafts with database
	if err := syncDraftFilesystemWithDatabase(db); err != nil {
		slog.Warn("Failed to sync filesystem drafts", "error", err)
	}

	rows, err := db.Query(`
		SELECT id, entity_type, entity_name, content, owner, created_at, updated_at, status
		FROM drafts
		ORDER BY updated_at DESC
	`)
	if err != nil {
		respondInternalError(w, "Failed to query drafts", err)
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
			respondInternalError(w, "Failed to scan draft", err)
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

	respondJSON(w, http.StatusOK, response)
}

func handleGetDraft(w http.ResponseWriter, r *http.Request) {
	// Defensive check for unit tests (middleware protects in production)
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	respondJSON(w, http.StatusOK, draft)
}

func handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	// Defensive check for unit tests (middleware protects in production)
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	var req CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if !isValidEntityType(req.EntityType) {
		respondInvalidEntityType(w)
		return
	}

	if req.EntityName == "" {
		respondBadRequest(w, "entity_name is required")
		return
	}

	// Generate UUID for new drafts
	draftID := uuid.New().String()
	now := time.Now()

	// Insert into database and capture the actual ID (which may differ on conflict)
	var actualID string
	err := db.QueryRow(`
		INSERT INTO drafts (id, entity_type, entity_name, content, owner, created_at, updated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (entity_type, entity_name)
		DO UPDATE SET content = $4, owner = $5, updated_at = $7
		RETURNING id
	`, draftID, req.EntityType, req.EntityName, req.Content, nullString(req.Owner), now, now, DraftStatusDraft).Scan(&actualID)

	if err != nil {
		respondInternalError(w, "Failed to create draft", err)
		return
	}

	// Use the actual ID from the database
	draftID = actualID

	// Write draft to filesystem
	if err := saveDraftToFile(req.EntityType, req.EntityName, req.Content); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
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
		Status:     DraftStatusDraft,
	}

	respondJSON(w, http.StatusCreated, draft)
}

func handleUpdateDraft(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	draftID := vars["id"]

	// Get draft metadata (getDraftByID handles db nil check)
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	var req UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
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
		respondInternalError(w, "Failed to update draft", err)
		return
	}

	// Update filesystem
	if err := saveDraftToFile(draft.EntityType, draft.EntityName, req.Content); err != nil {
		respondInternalError(w, "Failed to save draft file", err)
		return
	}

	draft.Content = req.Content
	draft.UpdatedAt = now

	respondJSON(w, http.StatusOK, draft)
}

func handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	// Defensive check for unit tests (middleware protects in production)
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
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
		respondNotFound(w, "Draft")
		return
	}
	if err != nil {
		respondInternalError(w, "Failed to get draft", err)
		return
	}

	// Delete from database
	_, err = db.Exec(`DELETE FROM drafts WHERE id = $1`, draftID)
	if err != nil {
		respondInternalError(w, "Failed to delete draft", err)
		return
	}

	// Delete from filesystem
	draftPath := getDraftPath(entityType, entityName)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		respondInternalError(w, "Failed to delete draft file", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func syncDraftFilesystemWithDatabase(exec dbExecutor) error {
	if exec == nil {
		return fmt.Errorf("draft executor is nil")
	}

	draftRoot := filepath.Join("..", "data", "prd-drafts")
	info, err := os.Stat(draftRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat draft directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("draft path is not a directory: %s", draftRoot)
	}

	return filepath.WalkDir(draftRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		rel, err := filepath.Rel(draftRoot, path)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) != 2 {
			return nil
		}

		entityType := parts[0]
		if !isValidEntityType(entityType) {
			return nil
		}

		entityName := strings.TrimSuffix(parts[1], filepath.Ext(parts[1]))
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read draft file %s: %w", path, err)
		}

		modTime := time.Now()
		if fileInfo, err := entry.Info(); err == nil {
			if !fileInfo.ModTime().IsZero() {
				modTime = fileInfo.ModTime()
			}
		}

		if _, err := exec.Exec(`
			INSERT INTO drafts (entity_type, entity_name, content, created_at, updated_at, status)
			VALUES ($1, $2, $3, $4, $4, 'draft')
			ON CONFLICT (entity_type, entity_name) DO NOTHING
		`, entityType, entityName, string(content), modTime); err != nil {
			return fmt.Errorf("failed to sync draft %s/%s: %w", entityType, entityName, err)
		}

		return nil
	})
}

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

// getDraftByID retrieves a draft by ID from the database
func getDraftByID(draftID string) (Draft, error) {
	if db == nil {
		return Draft{}, ErrDatabaseNotAvailable
	}

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

	if err != nil {
		return Draft{}, err
	}

	if owner.Valid {
		draft.Owner = owner.String
	}

	return draft, nil
}
