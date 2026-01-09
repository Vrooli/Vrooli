package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

type executionVideosResponse struct {
	ExecutionID string                            `json:"execution_id"`
	Videos      []workflow.ExecutionVideoArtifact `json:"videos"`
}

// GetExecutionRecordedVideos handles GET /api/v1/executions/{id}/recorded-videos
func (h *Handler) GetExecutionRecordedVideos(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	videos, err := h.executionService.GetExecutionVideoArtifacts(ctx, executionID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.respondError(w, ErrExecutionNotFound.WithDetails(map[string]string{"execution_id": executionID.String()}))
			return
		}
		h.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get recorded videos")
		h.respondError(w, ErrDatabaseError.WithDetails(map[string]string{"operation": "get_recorded_videos"}))
		return
	}

	h.respondSuccess(w, http.StatusOK, executionVideosResponse{
		ExecutionID: executionID.String(),
		Videos:      videos,
	})
}
