package main

import (
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
	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body (optional)
	var req PublishRequest
	req.CreateBackup = true // Default to creating backup
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Non-fatal: just use defaults if decode fails
			slog.Warn("Failed to decode publish request, using defaults", "error", err)
		}
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	// Validate operational targets linkage before publishing
	if err := validateOperationalTargetsLinkage(draft.EntityType, draft.EntityName, draft.Content); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
			"error":   "ORPHANED_CRITICAL_TARGETS",
		})
		return
	}

	// Construct target PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "Failed to get Vrooli root", err)
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
				respondInternalError(w, "Failed to create backup", err)
				return
			}
		}
	}

	// Write PRD.md file
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		respondInternalError(w, "Failed to create target directory", err)
		return
	}

	if err := os.WriteFile(targetPath, []byte(draft.Content), 0644); err != nil {
		respondInternalError(w, "Failed to write PRD file", err)
		return
	}

	// Update draft status to 'published'
	_, err = db.Exec(`
		UPDATE drafts
		SET status = $1, updated_at = $2
		WHERE id = $3
	`, DraftStatusPublished, time.Now(), draftID)

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

	respondJSON(w, http.StatusOK, response)
}

// validateOperationalTargetsLinkage checks if P0/P1 operational targets have linkages to requirements
func validateOperationalTargetsLinkage(entityType, entityName, content string) error {
	// Get operational targets from draft content
	targets := parseOperationalTargetsV2(content, entityType, entityName)
	if len(targets) == 0 {
		// No targets to validate
		return nil
	}

	// Load requirements to check linkages
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		slog.Warn("Failed to get Vrooli root for validation", "error", err)
		return nil
	}

	reqsPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "requirements", "index.json")
	if _, err := os.Stat(reqsPath); os.IsNotExist(err) {
		// No requirements file - skip validation
		return nil
	}

	// Find orphaned critical targets (P0/P1 without linked requirements)
	orphanedCritical := []OperationalTarget{}
	for _, target := range targets {
		if (target.Criticality == "P0" || target.Criticality == "P1") && len(target.LinkedRequirements) == 0 {
			orphanedCritical = append(orphanedCritical, target)
		}
	}

	if len(orphanedCritical) > 0 {
		msg := fmt.Sprintf("Cannot publish: %d critical operational target(s) (P0/P1) are not linked to requirements. "+
			"P0 and P1 targets MUST have linked requirements before publishing. "+
			"Orphaned targets: ", len(orphanedCritical))
		for i, target := range orphanedCritical {
			if i > 0 {
				msg += ", "
			}
			msg += fmt.Sprintf("%s (%s)", target.Title, target.Criticality)
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
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
