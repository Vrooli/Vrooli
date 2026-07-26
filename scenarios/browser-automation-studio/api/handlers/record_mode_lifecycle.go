package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
)

// StartLiveRecording, StopLiveRecording, and GetRecordingStatus form the
// recording-runtime lifecycle. Browser session lifecycle is intentionally kept
// separate in record_mode.go.
func (h *Handler) StartLiveRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	var req StartRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": "Invalid JSON body: " + err.Error()}))
		return
	}
	if req.SessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "session_id"}))
		return
	}
	cfg := &livecapture.RecordingConfig{APIHost: os.Getenv("API_HOST"), APIPort: os.Getenv("API_PORT")}
	if req.FrameQuality != nil {
		cfg.FrameQuality = *req.FrameQuality
	}
	if req.FrameFPS != nil {
		cfg.FrameFPS = *req.FrameFPS
	}
	resp, err := h.recordModeService.StartRecording(ctx, req.SessionID, cfg)
	if err != nil {
		h.log.WithError(err).Error("Failed to start recording")
		if driverErr, ok := err.(*driver.Error); ok && strings.Contains(driverErr.Message, "RECORDING_IN_PROGRESS") {
			h.respondError(w, ErrRecordingInProgress)
			return
		}
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	payload := StartRecordingResponse{RecordingID: uuid.NewString(), SessionID: resp.SessionID, StartedAt: resp.StartedAt}
	if pb, err := protoconv.StartRecordingToProto(payload); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, payload)
}

func (h *Handler) StopLiveRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	resp, err := h.recordModeService.DriverClient().StopRecording(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to stop recording")
		if driverErr, ok := err.(*driver.Error); ok && driverErr.Status == http.StatusNotFound {
			h.respondError(w, ErrExecutionNotFound.WithMessage("No recording in progress for this session"))
			return
		}
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile after stop")
	}
	payload := StopRecordingResponse{SessionID: resp.SessionID, ActionCount: resp.ActionCount, StoppedAt: resp.StoppedAt}
	if pb, err := protoconv.StopRecordingToProto(payload); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, payload)
}

func (h *Handler) GetRecordingStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	status, err := h.recordModeService.DriverClient().GetRecordingStatus(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get recording status")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	payload := RecordingStatusResponse{SessionID: status.SessionID, IsRecording: status.IsRecording, ActionCount: status.ActionCount, StartedAt: status.StartedAt, FrameCount: status.FrameCount}
	if pb, err := protoconv.RecordingStatusToProto(payload); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, payload)
}
