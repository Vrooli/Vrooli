package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
)

// PublishRequest represents a publish request
type PublishRequest struct {
	CreateBackup bool `json:"create_backup"`
}

// PublishResponse represents the result of a publish operation
type PublishResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PublishedTo string `json:"published_to"`
	BackupPath  string `json:"backup_path,omitempty"`
	PublishedAt string `json:"published_at"`
}

func handlePublishDraft(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body (optional)
	var req PublishRequest
	req.CreateBackup = true // Default to creating backup
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	// Get draft from database
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

	// Construct target PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get Vrooli root: %v", err), http.StatusInternalServerError)
		return
	}

	baseDir := filepath.Join(vrooliRoot, draft.EntityType+"s")
	targetPath := filepath.Join(baseDir, draft.EntityName, "PRD.md")

	// Create backup if requested and file exists
	var backupPath string
	if req.CreateBackup {
		if _, err := os.Stat(targetPath); err == nil {
			backupPath = targetPath + ".backup." + time.Now().Format("20060102-150405")
			if err := copyFile(targetPath, backupPath); err != nil {
				http.Error(w, fmt.Sprintf("Failed to create backup: %v", err), http.StatusInternalServerError)
				return
			}
		}
	}

	// Write PRD.md file
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create target directory: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(targetPath, []byte(draft.Content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write PRD file: %v", err), http.StatusInternalServerError)
		return
	}

	// Update draft status to 'published'
	_, err = db.Exec(`
		UPDATE drafts
		SET status = 'published', updated_at = $1
		WHERE id = $2
	`, time.Now(), draftID)

	if err != nil {
		// Non-fatal, just log
		slog.Warn("Failed to update draft status", "error", err, "draft_id", draftID)
	}

	// Delete draft file
	draftPath := getDraftPath(draft.EntityType, draft.EntityName)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		// Non-fatal, just log
		slog.Warn("Failed to delete draft file", "error", err, "path", draftPath)
	}

	response := PublishResponse{
		Success:     true,
		Message:     "Draft published successfully",
		PublishedTo: targetPath,
		BackupPath:  backupPath,
		PublishedAt: time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}
