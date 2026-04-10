package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// configExportRequest is the expected body for POST /api/v1/config/export.
type configExportRequest struct {
	Resources map[string]resourceConfig `json:"resources"`
	OutputDir string                    `json:"output_dir,omitempty"`
}

// configExportResponse contains the result of a config export operation.
type configExportResponse struct {
	Path      string `json:"path"`
	Success   bool   `json:"success"`
	BackupDir string `json:"backup_dir,omitempty"`
}

// backupFile copies an existing file into a timestamped backup under outputDir/backups/.
// Returns the backup directory path (empty if no backup was needed) or an error.
func backupFile(outputPath, outputDir string) (string, error) {
	if _, err := os.Stat(outputPath); err != nil {
		return "", nil // No existing file to back up
	}

	backupDir := filepath.Join(outputDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to read existing config for backup: %w", err)
	}

	backupName := fmt.Sprintf("generated-service.%s.json", time.Now().UTC().Format("20060102-150405"))
	if err := os.WriteFile(filepath.Join(backupDir, backupName), data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupDir, nil
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	var req configExportRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if len(req.Resources) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "resources map must not be empty",
		})
		return
	}

	// Determine output directory
	outputDir := req.OutputDir
	if outputDir == "" {
		root := os.Getenv("VROOLI_ROOT")
		if root == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "VROOLI_ROOT not set and no output_dir provided",
			})
			return
		}
		outputDir = filepath.Join(root, ".vrooli")
	}

	outputPath := filepath.Join(outputDir, "generated-service.json")

	backupDir, err := backupFile(outputPath, outputDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Build and write the service.json content
	snippet := serviceJSONSnippet{
		Resources: req.Resources,
	}
	data, err := json.MarshalIndent(snippet, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to marshal config: " + err.Error(),
		})
		return
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to create output directory: " + err.Error(),
		})
		return
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to write config file: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, configExportResponse{
		Path:      outputPath,
		Success:   true,
		BackupDir: backupDir,
	})
}
