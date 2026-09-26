package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
)

// ValidateSelector owns the selector-validation boundary used by recording
// preview and authoring. It deliberately has no recording session mutation.
func (h *Handler) ValidateSelector(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	var req struct {
		Selector string `json:"selector"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	if req.Selector == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "selector"}))
		return
	}
	resp, err := h.recordModeService.DriverClient().ValidateSelector(ctx, sessionID, &driver.ValidateSelectorRequest{Selector: req.Selector})
	if err != nil {
		h.log.WithError(err).Error("Failed to validate selector")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	payload := struct {
		Valid      bool   `json:"valid"`
		MatchCount int    `json:"match_count"`
		Selector   string `json:"selector"`
		Error      string `json:"error,omitempty"`
	}{resp.Valid, resp.MatchCount, resp.Selector, resp.Error}
	if pb, err := protoconv.SelectorValidationToProto(payload); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, payload)
}

// ReplayRecordingPreview verifies a proposed action sequence without mutating
// workflow state. It belongs with selector validation because both are
// driver-backed authoring checks.
func (h *Handler) ReplayRecordingPreview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	var req ReplayPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	if len(req.Actions) == 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "No actions to replay"}))
		return
	}
	resp, err := h.recordModeService.DriverClient().ReplayPreview(ctx, sessionID, &driver.ReplayPreviewRequest{
		Actions: req.Actions, Limit: req.Limit, StopOnFailure: req.StopOnFailure, ActionTimeout: req.ActionTimeout,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to replay preview")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	results := make([]ActionReplayResult, len(resp.Results))
	for i, result := range resp.Results {
		var replayErr *ActionReplayError
		if result.Error != "" {
			replayErr = &ActionReplayError{Message: result.Error}
		}
		results[i] = ActionReplayResult{SequenceNum: result.Index, ActionType: result.ActionType, Success: result.Success, DurationMs: result.DurationMs, Error: replayErr}
	}
	payload := ReplayPreviewResponse{
		Success: resp.Success, TotalActions: len(req.Actions), PassedActions: resp.PassedActions, FailedActions: resp.FailedActions,
		Results: results, TotalDurationMs: resp.TotalDurationMs,
		StoppedEarly: resp.FailedActions > 0 && req.StopOnFailure != nil && *req.StopOnFailure,
	}
	if pb, err := protoconv.ReplayPreviewToProto(payload); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, payload)
}
