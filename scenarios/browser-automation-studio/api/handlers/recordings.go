package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services"
)

// ImportRecording handles POST /api/v1/recordings/import
func (h *Handler) ImportRecording(w http.ResponseWriter, r *http.Request) {
	if h.recordingService == nil {
		http.Error(w, "Recording ingestion is not available", http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), recordingImportTimeout)
	defer cancel()

	archivePath, cleanup, err := h.persistRecordingArchive(r)
	if err != nil {
		h.log.WithError(err).Warn("Failed to persist recording archive")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer cleanup()

	opts, err := parseRecordingOptions(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.recordingService.ImportArchive(ctx, archivePath, opts)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			status = http.StatusRequestTimeout
		case errors.Is(err, services.ErrRecordingArchiveTooLarge):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(err, services.ErrRecordingManifestMissingFrames), errors.Is(err, services.ErrRecordingTooManyFrames):
			status = http.StatusBadRequest
		}
		h.log.WithError(err).Warn("Recording import failed")
		http.Error(w, err.Error(), status)
		return
	}

	response := map[string]any{
		"execution_id":  result.Execution.ID,
		"workflow_id":   result.Workflow.ID,
		"workflow_name": result.Workflow.Name,
		"project_id":    result.Project.ID,
		"project_name":  result.Project.Name,
		"frame_count":   result.FrameCount,
		"asset_count":   result.AssetCount,
		"duration_ms":   result.DurationMs,
		"message":       "Recording imported",
		"trigger_type":  result.Execution.TriggerType,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) persistRecordingArchive(r *http.Request) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "bas-recording-*.zip")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp archive: %w", err)
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	limit := int64(recordingUploadLimitBytes)
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))

	var reader io.Reader
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(recordingUploadLimitBytes); err != nil {
			tmpFile.Close()
			cleanup()
			return "", nil, fmt.Errorf("failed to parse multipart form: %w", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			tmpFile.Close()
			cleanup()
			return "", nil, errors.New("form field 'file' is required")
		}
		defer file.Close()
		reader = file
	} else {
		reader = r.Body
	}

	written, err := io.Copy(tmpFile, io.LimitReader(reader, limit+1))
	if err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to read recording archive: %w", err)
	}

	if written > limit {
		tmpFile.Close()
		cleanup()
		return "", nil, errors.New("recording archive exceeds maximum allowed size (200MB)")
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to flush archive: %w", err)
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to reset archive: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), cleanup, nil
}

func parseRecordingOptions(r *http.Request) (services.RecordingImportOptions, error) {
	opts := services.RecordingImportOptions{}

	projectIDRaw := firstNonEmpty(formValue(r, "project_id"), r.URL.Query().Get("project_id"))
	if projectIDRaw != "" {
		id, err := uuid.Parse(projectIDRaw)
		if err != nil {
			return opts, fmt.Errorf("invalid project_id: %s", projectIDRaw)
		}
		opts.ProjectID = &id
	}

	workflowIDRaw := firstNonEmpty(formValue(r, "workflow_id"), r.URL.Query().Get("workflow_id"))
	if workflowIDRaw != "" {
		id, err := uuid.Parse(workflowIDRaw)
		if err != nil {
			return opts, fmt.Errorf("invalid workflow_id: %s", workflowIDRaw)
		}
		opts.WorkflowID = &id
	}

	opts.ProjectName = strings.TrimSpace(firstNonEmpty(formValue(r, "project_name"), r.URL.Query().Get("project_name")))
	opts.WorkflowName = strings.TrimSpace(firstNonEmpty(formValue(r, "workflow_name"), r.URL.Query().Get("workflow_name")))

	return opts, nil
}

func formValue(r *http.Request, key string) string {
	if r.MultipartForm != nil {
		return strings.TrimSpace(r.FormValue(key))
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ServeRecordingAsset handles GET /api/v1/recordings/assets/{executionID}/*
// Streams stored recording assets from disk.
func (h *Handler) ServeRecordingAsset(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.recordingsRoot) == "" {
		http.Error(w, "Recording storage unavailable", http.StatusServiceUnavailable)
		return
	}

	execIDStr := chi.URLParam(r, "executionID")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	remainder := chi.URLParam(r, "*")
	if strings.TrimSpace(remainder) == "" {
		http.Error(w, "Asset path required", http.StatusBadRequest)
		return
	}

	cleaned := filepath.Clean(remainder)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		http.Error(w, "Invalid asset path", http.StatusBadRequest)
		return
	}

	baseDir := filepath.Join(h.recordingsRoot, execID.String())
	assetPath := filepath.Join(baseDir, cleaned)
	if !strings.HasPrefix(assetPath, baseDir) {
		http.Error(w, "Asset path escapes execution directory", http.StatusBadRequest)
		return
	}

	file, err := os.Open(assetPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Asset not found", http.StatusNotFound)
			return
		}
		h.log.WithError(err).WithField("asset", assetPath).Error("Failed to open recording asset")
		http.Error(w, "Unable to read asset", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to read asset", http.StatusInternalServerError)
		return
	}

	header := make([]byte, 512)
	n, _ := file.Read(header)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Unable to read asset", http.StatusInternalServerError)
		return
	}

	contentType := http.DetectContentType(header[:n])
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")

	http.ServeContent(w, r, filepath.Base(assetPath), info.ModTime(), file)
}
