package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
)

// ImportRecording handles POST /api/v1/recordings/import
func (h *Handler) ImportRecording(w http.ResponseWriter, r *http.Request) {
	if h.recordingService == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Recording ingestion is not available"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), recordingImportTimeout())
	defer cancel()

	archivePath, cleanup, err := h.persistRecordingArchive(r)
	if err != nil {
		h.log.WithError(err).Warn("Failed to persist recording archive")
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	defer cleanup()

	opts, err := parseRecordingOptions(r)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	result, err := h.recordingService.ImportArchive(ctx, archivePath, opts)
	if err != nil {
		var apiErr *APIError
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			apiErr = ErrRequestTimeout.WithDetails(map[string]string{"error": err.Error()})
		case errors.Is(err, archiveingestion.ErrArchiveTooLarge):
			apiErr = ErrRequestTooLarge.WithMessage("Recording archive exceeds maximum allowed size")
		case errors.Is(err, archiveingestion.ErrManifestMissingFrames), errors.Is(err, archiveingestion.ErrTooManyFrames):
			apiErr = ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()})
		default:
			apiErr = ErrInternalServer.WithDetails(map[string]string{"operation": "import_recording", "error": err.Error()})
		}
		h.log.WithError(err).Warn("Recording import failed")
		h.respondError(w, apiErr)
		return
	}

	h.respondSuccess(w, http.StatusCreated, map[string]any{
		"execution_id":  result.Execution.ID,
		"workflow_id":   result.Workflow.ID,
		"workflow_name": result.Workflow.Name,
		"project_id":    result.Project.ID,
		"project_name":  result.Project.Name,
		"frame_count":   result.FrameCount,
		"asset_count":   result.AssetCount,
		"duration_ms":   result.DurationMs,
		"message":       "Recording imported",
		"trigger_type":  "recording_import",
	})
}

func (h *Handler) persistRecordingArchive(r *http.Request) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "bas-recording-*.zip")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp archive: %w", err)
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	limit := recordingUploadLimitBytes()
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))

	var reader io.Reader
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(recordingUploadLimitBytes()); err != nil {
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

func parseRecordingOptions(r *http.Request) (archiveingestion.IngestionOptions, error) {
	opts := archiveingestion.IngestionOptions{}

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
		h.respondError(w, ErrServiceUnavailable.WithMessage("Recording storage unavailable"))
		return
	}

	execIDStr := chi.URLParam(r, "executionID")
	execID, err := uuid.Parse(execIDStr)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	remainder := chi.URLParam(r, "*")
	if strings.TrimSpace(remainder) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "asset_path"}))
		return
	}

	cleaned := filepath.Clean(remainder)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "asset_path", "error": "invalid path"}))
		return
	}

	baseDir := filepath.Join(h.recordingsRoot, execID.String())
	assetPath := filepath.Join(baseDir, cleaned)
	if !strings.HasPrefix(assetPath, baseDir) {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"field": "asset_path", "error": "path traversal attempt"}))
		return
	}

	file, err := os.Open(assetPath)
	if err != nil {
		if os.IsNotExist(err) {
			h.respondError(w, &APIError{
				Status:  http.StatusNotFound,
				Code:    "ASSET_NOT_FOUND",
				Message: "Asset not found",
				Details: map[string]string{"asset": filepath.Base(assetPath)},
			})
			return
		}
		h.log.WithError(err).WithField("asset", assetPath).Error("Failed to open recording asset")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "read_asset"}))
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "stat_asset"}))
		return
	}

	header := make([]byte, 512)
	n, _ := file.Read(header)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "seek_asset"}))
		return
	}

	contentType := http.DetectContentType(header[:n])
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")

	http.ServeContent(w, r, filepath.Base(assetPath), info.ModTime(), file)
}
