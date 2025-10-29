package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// GetExecutionScreenshots handles GET /api/v1/executions/{id}/screenshots
func (h *Handler) GetExecutionScreenshots(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	screenshots, err := h.workflowService.GetExecutionScreenshots(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get screenshots")
		http.Error(w, "Failed to get screenshots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"screenshots": screenshots,
	})
}

// GetExecutionTimeline handles GET /api/v1/executions/{id}/timeline
func (h *Handler) GetExecutionTimeline(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	timeline, err := h.workflowService.GetExecutionTimeline(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get execution timeline")
		http.Error(w, "Failed to get execution timeline", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(timeline); err != nil {
		h.log.WithError(err).Error("Failed to encode execution timeline response")
	}
}

// PostExecutionExport handles POST /api/v1/executions/{id}/export
func (h *Handler) PostExecutionExport(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	executionID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	preview, svcErr := h.workflowService.DescribeExecutionExport(ctx, executionID)
	if svcErr != nil {
		if errors.Is(svcErr, database.ErrNotFound) {
			http.Error(w, "Execution not found", http.StatusNotFound)
			return
		}
		h.log.WithError(svcErr).WithField("execution_id", executionID).Error("Failed to describe execution export")
		http.Error(w, "Failed to describe execution export", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(preview); err != nil {
		h.log.WithError(err).Error("Failed to encode execution export response")
	}
}

// GetExecution handles GET /api/v1/executions/{id}
func (h *Handler) GetExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	execution, err := h.workflowService.GetExecution(ctx, id)
	if err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// ListExecutions handles GET /api/v1/executions
func (h *Handler) ListExecutions(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	workflowIDStr := r.URL.Query().Get("workflow_id")
	var workflowID *uuid.UUID
	if workflowIDStr != "" {
		if id, err := uuid.Parse(workflowIDStr); err == nil {
			workflowID = &id
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	executions, err := h.workflowService.ListExecutions(ctx, workflowID, 100, 0)
	if err != nil {
		h.log.WithError(err).Error("Failed to list executions")
		http.Error(w, "Failed to list executions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"executions": executions,
	})
}

// StopExecution handles POST /api/v1/executions/{id}/stop
func (h *Handler) StopExecution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid execution ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.workflowService.StopExecution(ctx, id); err != nil {
		h.log.WithError(err).WithField("id", id).Error("Failed to stop execution")
		http.Error(w, "Failed to stop execution", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "stopped",
	})
}
