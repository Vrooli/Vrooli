package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
)

// Export errors
var (
	ErrInvalidExportID = &APIError{
		Status:  http.StatusBadRequest,
		Code:    "INVALID_EXPORT_ID",
		Message: "Invalid export ID format",
	}

	ErrExportNotFound = &APIError{
		Status:  http.StatusNotFound,
		Code:    "EXPORT_NOT_FOUND",
		Message: "Export not found",
	}
)

// CreateExportRequest represents the request to create an export record
type CreateExportRequest struct {
	ExecutionID   string         `json:"execution_id"`
	WorkflowID    string         `json:"workflow_id,omitempty"`
	Name          string         `json:"name"`
	Format        string         `json:"format"`
	Settings      map[string]any `json:"settings,omitempty"`
	StorageURL    string         `json:"storage_url,omitempty"`
	ThumbnailURL  string         `json:"thumbnail_url,omitempty"`
	FileSizeBytes *int64         `json:"file_size_bytes,omitempty"`
	DurationMs    *int           `json:"duration_ms,omitempty"`
	FrameCount    *int           `json:"frame_count,omitempty"`
	Status        string         `json:"status,omitempty"`
}

// UpdateExportRequest represents the request to update an export
type UpdateExportRequest struct {
	Name          string         `json:"name,omitempty"`
	Settings      map[string]any `json:"settings,omitempty"`
	StorageURL    string         `json:"storage_url,omitempty"`
	ThumbnailURL  string         `json:"thumbnail_url,omitempty"`
	FileSizeBytes *int64         `json:"file_size_bytes,omitempty"`
	DurationMs    *int           `json:"duration_ms,omitempty"`
	FrameCount    *int           `json:"frame_count,omitempty"`
	AICaption     string         `json:"ai_caption,omitempty"`
	Status        string         `json:"status,omitempty"`
	Error         string         `json:"error,omitempty"`
}

// ListExports handles GET /api/v1/exports
func (h *Handler) ListExports(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	limit, offset := parsePaginationParams(r, 0, 0)

	// Check for execution_id filter
	executionIDStr := strings.TrimSpace(r.URL.Query().Get("execution_id"))
	if executionIDStr != "" {
		executionID, err := uuid.Parse(executionIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidExecutionID)
			return
		}

		exports, err := h.repo.ListExportsByExecution(ctx, executionID)
		if err != nil {
			h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to list exports by execution")
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_exports_by_execution"}))
			return
		}

		h.respondSuccess(w, http.StatusOK, map[string]any{
			"exports": exports,
			"total":   len(exports),
		})
		return
	}

	// Check for workflow_id filter
	workflowIDStr := strings.TrimSpace(r.URL.Query().Get("workflow_id"))
	if workflowIDStr != "" {
		workflowID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID)
			return
		}

		exports, err := h.repo.ListExportsByWorkflow(ctx, workflowID, limit, offset)
		if err != nil {
			h.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to list exports by workflow")
			h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_exports_by_workflow"}))
			return
		}

		h.respondSuccess(w, http.StatusOK, map[string]any{
			"exports": exports,
			"total":   len(exports),
		})
		return
	}

	// List all exports
	exports, err := h.repo.ListExports(ctx, limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Failed to list exports")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "list_exports"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"exports": exports,
		"total":   len(exports),
	})
}

// GetExport handles GET /api/v1/exports/{id}
func (h *Handler) GetExport(w http.ResponseWriter, r *http.Request) {
	exportID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExportID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	export, err := h.repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExportNotFound)
			return
		}
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to get export")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_export"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"export": export,
	})
}

// CreateExport handles POST /api/v1/exports
func (h *Handler) CreateExport(w http.ResponseWriter, r *http.Request) {
	var req CreateExportRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode create export request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	// Validate required fields
	if req.ExecutionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "execution_id"}))
		return
	}

	executionID, err := uuid.Parse(req.ExecutionID)
	if err != nil {
		h.respondError(w, ErrInvalidExecutionID)
		return
	}

	if req.Name == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "name"}))
		return
	}

	if req.Format == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "format"}))
		return
	}

	// Validate format
	validFormats := map[string]bool{"mp4": true, "gif": true, "json": true, "html": true}
	if !validFormats[strings.ToLower(req.Format)] {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "format",
			"error": "format must be one of: mp4, gif, json, html",
		}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Verify execution exists
	_, err = h.executionService.GetExecution(ctx, executionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound)
			return
		}
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to verify execution")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_execution"}))
		return
	}

	export := &database.ExportIndex{
		ExecutionID:   executionID,
		Name:          req.Name,
		Format:        strings.ToLower(req.Format),
		Settings:      database.JSONMap(req.Settings),
		StorageURL:    req.StorageURL,
		ThumbnailURL:  req.ThumbnailURL,
		FileSizeBytes: req.FileSizeBytes,
		DurationMs:    req.DurationMs,
		FrameCount:    req.FrameCount,
		Status:        "completed",
	}

	// Parse optional workflow ID
	if req.WorkflowID != "" {
		workflowID, err := uuid.Parse(req.WorkflowID)
		if err != nil {
			h.respondError(w, ErrInvalidWorkflowID)
			return
		}
		export.WorkflowID = &workflowID
	}

	// Override status if provided
	if req.Status != "" {
		validStatuses := map[string]bool{"pending": true, "processing": true, "completed": true, "failed": true}
		if !validStatuses[req.Status] {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"field": "status",
				"error": "status must be one of: pending, processing, completed, failed",
			}))
			return
		}
		export.Status = req.Status
	}

	if err := h.repo.CreateExport(ctx, export); err != nil {
		h.log.WithError(err).Error("Failed to create export")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "create_export"}))
		return
	}

	h.respondSuccess(w, http.StatusCreated, map[string]any{
		"export_id": export.ID,
		"status":    "created",
		"export":    export,
	})
}

// UpdateExport handles PATCH /api/v1/exports/{id}
func (h *Handler) UpdateExport(w http.ResponseWriter, r *http.Request) {
	exportID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExportID)
	if !ok {
		return
	}

	var req UpdateExportRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode update export request")
		h.respondError(w, ErrInvalidRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get existing export
	export, err := h.repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExportNotFound)
			return
		}
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to get export for update")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_export"}))
		return
	}

	// Apply updates
	if req.Name != "" {
		export.Name = req.Name
	}
	if req.Settings != nil {
		export.Settings = database.JSONMap(req.Settings)
	}
	if req.StorageURL != "" {
		export.StorageURL = req.StorageURL
	}
	if req.ThumbnailURL != "" {
		export.ThumbnailURL = req.ThumbnailURL
	}
	if req.FileSizeBytes != nil {
		export.FileSizeBytes = req.FileSizeBytes
	}
	if req.DurationMs != nil {
		export.DurationMs = req.DurationMs
	}
	if req.FrameCount != nil {
		export.FrameCount = req.FrameCount
	}
	if req.AICaption != "" {
		export.AICaption = req.AICaption
	}
	if req.Status != "" {
		validStatuses := map[string]bool{"pending": true, "processing": true, "completed": true, "failed": true}
		if !validStatuses[req.Status] {
			h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
				"field": "status",
				"error": "status must be one of: pending, processing, completed, failed",
			}))
			return
		}
		export.Status = req.Status
	}
	if req.Error != "" {
		export.Error = req.Error
	}

	if err := h.repo.UpdateExport(ctx, export); err != nil {
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to update export")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "update_export"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"export_id": export.ID,
		"status":    "updated",
		"export":    export,
	})
}

// DeleteExport handles DELETE /api/v1/exports/{id}
func (h *Handler) DeleteExport(w http.ResponseWriter, r *http.Request) {
	exportID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExportID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	// Get export to check if it exists and get storage URL for cleanup
	export, err := h.repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExportNotFound)
			return
		}
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to get export for deletion")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_export"}))
		return
	}

	// Delete from database
	if err := h.repo.DeleteExport(ctx, exportID); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExportNotFound)
			return
		}
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to delete export")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "delete_export"}))
		return
	}

	// Note: Storage file cleanup is intentionally deferred to external processes.
	// The StorageURL typically points to cloud storage (S3-compatible) or local filesystem
	// paths for rendered video exports. These are cleaned up by:
	// 1. TTL-based expiration rules on the storage bucket
	// 2. Periodic cleanup jobs that scan for orphaned files
	// This avoids tight coupling between export deletion and storage implementation details.
	if export.StorageURL != "" {
		h.log.WithFields(logrus.Fields{
			"export_id":   exportID,
			"storage_url": export.StorageURL,
		}).Debug("Export deleted; storage file cleanup delegated to external process")
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"export_id": exportID,
		"status":    "deleted",
	})
}

// GenerateExportCaption handles POST /api/v1/exports/{id}/generate-caption
func (h *Handler) GenerateExportCaption(w http.ResponseWriter, r *http.Request) {
	exportID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExportID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.ExtendedRequestTimeout)
	defer cancel()

	// Get export details
	export, err := h.repo.GetExport(ctx, exportID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExportNotFound)
			return
		}
		h.log.WithError(err).WithField("export_id", exportID).Error("Failed to get export for caption generation")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_export"}))
		return
	}

	// Get workflow details if available
	var workflowName string
	var workflowDescription string
	if export.WorkflowID != nil {
		workflow, wfErr := h.catalogService.GetWorkflow(ctx, *export.WorkflowID)
		if wfErr == nil && workflow != nil {
			workflowName = workflow.GetName()
			workflowDescription = workflow.GetDescription()
		}
	}

	// Build the prompt for caption generation
	formatLabel := map[string]string{
		"mp4":  "video",
		"gif":  "animated GIF",
		"json": "JSON export package",
		"html": "interactive HTML replay",
	}[export.Format]
	if formatLabel == "" {
		formatLabel = export.Format
	}

	durationStr := ""
	if export.DurationMs != nil && *export.DurationMs > 0 {
		seconds := *export.DurationMs / 1000
		if seconds < 60 {
			durationStr = fmt.Sprintf("%d seconds", seconds)
		} else {
			minutes := seconds / 60
			remainingSec := seconds % 60
			durationStr = fmt.Sprintf("%d:%02d", minutes, remainingSec)
		}
	}

	prompt := fmt.Sprintf(`Generate a short, engaging caption for sharing this browser automation export on social media or in a presentation.

Export Details:
- Name: %s
- Format: %s
- Workflow: %s
%s%s

Requirements:
- Keep the caption concise (1-2 sentences, max 280 characters for Twitter compatibility)
- Make it engaging and professional
- Highlight what the automation does if clear from the name
- Don't use hashtags unless specifically requested
- Don't include emoji unless it adds value

Return ONLY the caption text, nothing else.`,
		export.Name,
		formatLabel,
		workflowName,
		func() string {
			if workflowDescription != "" {
				return fmt.Sprintf("- Description: %s\n", workflowDescription)
			}
			return ""
		}(),
		func() string {
			if durationStr != "" {
				return fmt.Sprintf("- Duration: %s\n", durationStr)
			}
			return ""
		}(),
	)

	// Call the AI service
	aiClient := ai.NewOpenRouterClient(h.log)
	caption, aiErr := aiClient.ExecutePrompt(ctx, prompt)
	if aiErr != nil {
		h.log.WithError(aiErr).WithField("export_id", exportID).Error("Failed to generate AI caption")
		// Return a fallback caption instead of erroring
		caption = fmt.Sprintf("Check out this %s replay: %s", formatLabel, export.Name)
	}

	// Clean up the caption
	caption = strings.TrimSpace(caption)
	caption = strings.Trim(caption, "\"'")

	// Update the export with the generated caption
	now := time.Now()
	export.AICaption = caption
	export.AICaptionGeneratedAt = &now

	if updateErr := h.repo.UpdateExport(ctx, export); updateErr != nil {
		h.log.WithError(updateErr).WithField("export_id", exportID).Error("Failed to save generated caption")
		// Still return the caption even if save fails
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"export_id": exportID,
		"caption":   caption,
		"export":    export,
	})
}
