package handlers

import (
	"net/http"

	"github.com/vrooli/browser-automation-studio/services/workflow"
)

type executionHarResponse struct {
	ExecutionID string                           `json:"execution_id"`
	HARFiles    []workflow.ExecutionFileArtifact `json:"har_files"`
}

// GetExecutionRecordedHar handles GET /api/v1/executions/{id}/recorded-har
func (h *Handler) GetExecutionRecordedHar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	harFiles, err := h.executionService.GetExecutionHarArtifacts(ctx, executionID)
	if err != nil {
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get recorded HAR files")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_recorded_har"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, executionHarResponse{
		ExecutionID: executionID.String(),
		HARFiles:    harFiles,
	})
}
