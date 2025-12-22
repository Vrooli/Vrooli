package handlers

import (
	"net/http"

	"github.com/vrooli/browser-automation-studio/services/workflow"
)

type executionTracesResponse struct {
	ExecutionID string                           `json:"execution_id"`
	Traces      []workflow.ExecutionFileArtifact `json:"traces"`
}

// GetExecutionRecordedTraces handles GET /api/v1/executions/{id}/recorded-traces
func (h *Handler) GetExecutionRecordedTraces(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	traces, err := h.executionService.GetExecutionTraceArtifacts(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get recorded traces")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_recorded_traces"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, executionTracesResponse{
		ExecutionID: executionID.String(),
		Traces:      traces,
	})
}
