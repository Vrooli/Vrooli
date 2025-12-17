package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/performance"
	archiveingestion "github.com/vrooli/browser-automation-studio/services/archive-ingestion"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/websocket"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
)

// RecordedAction is an alias for livecapture.RecordedAction.
//
// Deprecated: Import livecapture.RecordedAction from
// github.com/vrooli/browser-automation-studio/services/live-capture directly.
// This alias will be removed in a future version.
type RecordedAction = livecapture.RecordedAction

// SelectorSet is an alias for livecapture.SelectorSet.
//
// Deprecated: Import livecapture.SelectorSet from
// github.com/vrooli/browser-automation-studio/services/live-capture directly.
// This alias will be removed in a future version.
type SelectorSet = livecapture.SelectorSet

// SelectorCandidate is an alias for livecapture.SelectorCandidate.
//
// Deprecated: Import livecapture.SelectorCandidate from
// github.com/vrooli/browser-automation-studio/services/live-capture directly.
// This alias will be removed in a future version.
type SelectorCandidate = livecapture.SelectorCandidate

// ElementMeta is an alias for livecapture.ElementMeta.
//
// Deprecated: Import livecapture.ElementMeta from
// github.com/vrooli/browser-automation-studio/services/live-capture directly.
// This alias will be removed in a future version.
type ElementMeta = livecapture.ElementMeta

// CreateRecordingSessionRequest is the request body for creating a browser session for recording.
type CreateRecordingSessionRequest struct {
	// Viewport dimensions (optional, defaults to 1280x720)
	ViewportWidth  int `json:"viewport_width,omitempty"`
	ViewportHeight int `json:"viewport_height,omitempty"`
	// Initial URL to navigate to (optional)
	InitialURL string `json:"initial_url,omitempty"`
	// Optional persisted session profile to load cookies/storage from
	SessionProfileID string `json:"session_profile_id,omitempty"`
	// Stream quality settings (optional)
	// Quality: JPEG quality 0-100 (default 55)
	StreamQuality *int `json:"stream_quality,omitempty"`
	// FPS: Frames per second 1-60 (default 6)
	StreamFPS *int `json:"stream_fps,omitempty"`
	// Scale: "css" for 1x scale, "device" for device pixel ratio (default "css")
	StreamScale string `json:"stream_scale,omitempty"`
}

// CreateRecordingSessionResponse is the response after creating a recording session.
type CreateRecordingSessionResponse struct {
	SessionID          string `json:"session_id"`
	CreatedAt          string `json:"created_at"`
	SessionProfileID   string `json:"session_profile_id,omitempty"`
	SessionProfileName string `json:"session_profile_name,omitempty"`
	LastUsedAt         string `json:"last_used_at,omitempty"`
}

// CloseRecordingSessionRequest is the request body for closing a recording session.
type CloseRecordingSessionRequest struct {
	SessionID string `json:"session_id"`
}

// StartRecordingRequest is the request body for starting a live recording.
type StartRecordingRequest struct {
	SessionID   string `json:"session_id"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// StartRecordingResponse is the response after starting recording.
type StartRecordingResponse struct {
	RecordingID string `json:"recording_id"`
	SessionID   string `json:"session_id"`
	StartedAt   string `json:"started_at"`
}

// StopRecordingResponse is the response after stopping recording.
type StopRecordingResponse struct {
	RecordingID string `json:"recording_id"`
	SessionID   string `json:"session_id"`
	ActionCount int    `json:"action_count"`
	StoppedAt   string `json:"stopped_at"`
}

// RecordingStatusResponse is the response for recording status.
type RecordingStatusResponse struct {
	SessionID   string `json:"session_id"`
	IsRecording bool   `json:"is_recording"`
	RecordingID string `json:"recording_id,omitempty"`
	ActionCount int    `json:"action_count"`
	FrameCount  int    `json:"frame_count,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
}

// GetActionsResponse is the response for getting recorded actions.
type GetActionsResponse struct {
	SessionID   string           `json:"session_id"`
	IsRecording bool             `json:"is_recording,omitempty"`
	Actions     []RecordedAction `json:"actions"`
	Count       int              `json:"count"`
}

// GenerateWorkflowRequest is the request body for generating a workflow from recording.
type GenerateWorkflowRequest struct {
	SessionID   string     `json:"session_id"`
	Name        string     `json:"name"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	ProjectName string     `json:"project_name,omitempty"`
	ActionRange *struct {
		Start int `json:"start"`
		End   int `json:"end"`
	} `json:"action_range,omitempty"`
	// Actions can be provided directly (with edits) instead of fetching from driver
	Actions []RecordedAction `json:"actions,omitempty"`
}

// GenerateWorkflowResponse is the response after generating a workflow.
type GenerateWorkflowResponse struct {
	WorkflowID  uuid.UUID `json:"workflow_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	Name        string    `json:"name"`
	NodeCount   int       `json:"node_count"`
	ActionCount int       `json:"action_count"`
}

// ReplayPreviewRequest is the request body for testing recorded actions.
type ReplayPreviewRequest struct {
	SessionID     string           `json:"session_id"`
	Actions       []RecordedAction `json:"actions"`
	Limit         *int             `json:"limit,omitempty"`
	StopOnFailure *bool            `json:"stop_on_failure,omitempty"`
	ActionTimeout *int             `json:"action_timeout,omitempty"`
}

// ActionReplayError contains error details for a failed action.
type ActionReplayError struct {
	Message    string `json:"message"`
	Code       string `json:"code"`
	MatchCount *int   `json:"match_count,omitempty"`
	Selector   string `json:"selector,omitempty"`
}

// ActionReplayResult is the result of replaying a single action.
type ActionReplayResult struct {
	ActionID          string             `json:"action_id"`
	SequenceNum       int                `json:"sequence_num"`
	ActionType        string             `json:"action_type"`
	Success           bool               `json:"success"`
	DurationMs        int                `json:"duration_ms"`
	Error             *ActionReplayError `json:"error,omitempty"`
	ScreenshotOnError string             `json:"screenshot_on_error,omitempty"`
}

// ReplayPreviewResponse is the response from replay preview.
type ReplayPreviewResponse struct {
	Success         bool                 `json:"success"`
	TotalActions    int                  `json:"total_actions"`
	PassedActions   int                  `json:"passed_actions"`
	FailedActions   int                  `json:"failed_actions"`
	Results         []ActionReplayResult `json:"results"`
	TotalDurationMs int                  `json:"total_duration_ms"`
	StoppedEarly    bool                 `json:"stopped_early"`
}

// NavigateRecordingRequest is the request body for navigating the recording session.
type NavigateRecordingRequest struct {
	URL       string `json:"url"`
	WaitUntil string `json:"wait_until,omitempty"`
	TimeoutMs int    `json:"timeout_ms,omitempty"`
	Capture   bool   `json:"capture,omitempty"`
}

// NavigateRecordingResponse is the response from navigating the recording session.
type NavigateRecordingResponse struct {
	SessionID  string `json:"session_id"`
	URL        string `json:"url"`
	Title      string `json:"title,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
}

// RecordingScreenshotResponse is the response for capturing a screenshot from the recording session.
type RecordingScreenshotResponse struct {
	SessionID  string `json:"session_id"`
	Screenshot string `json:"screenshot"`
}

// RecordingFrameResponse is the response for lightweight frame previews.
// Uses WebP format for ~25-30% better compression than JPEG at same quality.
type RecordingFrameResponse struct {
	SessionID   string `json:"session_id"`
	Mime        string `json:"mime"` // "image/webp" or "image/jpeg"
	Image       string `json:"image"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	CapturedAt  string `json:"captured_at"`
	ContentHash string `json:"content_hash"`         // MD5 hash of raw frame buffer for reliable ETag
	PageTitle   string `json:"page_title,omitempty"` // Current page title (document.title)
	PageURL     string `json:"page_url,omitempty"`   // Current page URL
}

// RecordingViewportRequest updates viewport dimensions.
type RecordingViewportRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// UpdateStreamSettingsRequest is the request body for updating stream settings mid-session.
type UpdateStreamSettingsRequest struct {
	// Quality: JPEG quality 1-100
	Quality *int `json:"quality,omitempty"`
	// FPS: Target frames per second 1-60
	FPS *int `json:"fps,omitempty"`
	// Scale: "css" for 1x, "device" for devicePixelRatio (cannot change mid-session)
	Scale string `json:"scale,omitempty"`
	// PerfMode: Enable/disable debug performance mode for this session
	PerfMode *bool `json:"perfMode,omitempty"`
}

// UpdateStreamSettingsResponse is the response after updating stream settings.
type UpdateStreamSettingsResponse struct {
	SessionID    string `json:"session_id"`
	Quality      int    `json:"quality"`
	FPS          int    `json:"fps"`
	CurrentFPS   int    `json:"current_fps"`
	Scale        string `json:"scale"`
	IsStreaming  bool   `json:"is_streaming"`
	Updated      bool   `json:"updated"`
	ScaleWarning string `json:"scale_warning,omitempty"`
	PerfMode     bool   `json:"perf_mode"`
}

const (
	recordModeTimeout    = 30 * time.Second
	playwrightDriverEnv  = "PLAYWRIGHT_DRIVER_URL"
	defaultPlaywrightURL = "http://127.0.0.1:39400"
)

func getPlaywrightDriverURL() string {
	url := os.Getenv(playwrightDriverEnv)
	if url == "" {
		return defaultPlaywrightURL
	}
	return url
}

// CreateRecordingSession handles POST /api/v1/recordings/live/session
// Creates a new browser session for recording user actions.
func (h *Handler) CreateRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	var req CreateRecordingSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Resolve session profile for authentication persistence
	var profileID, profileName, profileLastUsed string
	var storageState json.RawMessage
	if h.sessionProfiles != nil {
		profile, err := h.resolveSessionProfile(req.SessionProfileID)
		if err != nil {
			h.respondError(w, err)
			return
		}
		if profile != nil {
			profileID = profile.ID
			profileName = profile.Name
			profileLastUsed = profile.LastUsedAt.Format(time.RFC3339)
			storageState = profile.StorageState
		}
	}

	// Apply stream settings with defaults
	streamQuality := 55
	if req.StreamQuality != nil && *req.StreamQuality >= 1 && *req.StreamQuality <= 100 {
		streamQuality = *req.StreamQuality
	}
	streamFPS := 6
	if req.StreamFPS != nil && *req.StreamFPS >= 1 && *req.StreamFPS <= 60 {
		streamFPS = *req.StreamFPS
	}
	streamScale := "css"
	if req.StreamScale == "device" {
		streamScale = "device"
	}

	// Delegate to recordmode service
	cfg := &livecapture.SessionConfig{
		ViewportWidth:  req.ViewportWidth,
		ViewportHeight: req.ViewportHeight,
		InitialURL:     req.InitialURL,
		StreamQuality:  streamQuality,
		StreamFPS:      streamFPS,
		StreamScale:    streamScale,
		StorageState:   storageState,
		APIHost:        os.Getenv("API_HOST"),
		APIPort:        os.Getenv("API_PORT"),
	}

	result, err := h.recordModeService.CreateSession(ctx, cfg)
	if err != nil {
		h.log.WithError(err).Error("Failed to create recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Update session profile usage tracking
	if profileID != "" && h.sessionProfiles != nil {
		if updated, err := h.sessionProfiles.Touch(profileID); err != nil {
			h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to update session profile usage")
		} else if updated != nil {
			profileName = updated.Name
			profileLastUsed = updated.LastUsedAt.Format(time.RFC3339)
		}
		h.setActiveSessionProfile(result.SessionID, profileID)
	}

	response := CreateRecordingSessionResponse{
		SessionID:          result.SessionID,
		CreatedAt:          result.CreatedAt.Format(time.RFC3339),
		SessionProfileID:   profileID,
		SessionProfileName: profileName,
		LastUsedAt:         profileLastUsed,
	}

	if pb, err := protoconv.RecordingSessionToProto(response); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, response)
}

// CloseRecordingSession handles POST /api/v1/recordings/live/session/{sessionId}/close
// Closes a recording session and cleans up resources.
func (h *Handler) CloseRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Capture storage state before closing (for session profile persistence)
	var storageState json.RawMessage
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID != "" && h.sessionProfiles != nil {
		if state, err := h.recordModeService.GetStorageState(ctx, sessionID); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"session_id": sessionID,
				"profile_id": profileID,
			}).Warn("Failed to capture storage state before closing session")
		} else {
			storageState = state
		}
	}

	// Delegate to recordmode service
	if err := h.recordModeService.CloseSession(ctx, sessionID); err != nil {
		h.log.WithError(err).Error("Failed to close recording session")
		// Check for not found error
		if driverErr, ok := err.(*livecapture.DriverError); ok && driverErr.StatusCode == 404 {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
			return
		}
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Persist storage state to profile after successful close
	if profileID != "" && h.sessionProfiles != nil && len(storageState) > 0 {
		if _, err := h.sessionProfiles.SaveStorageState(profileID, storageState); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"profile_id": profileID,
				"session_id": sessionID,
			}).Warn("Failed to persist session profile storage state")
		}
	}

	h.clearActiveSessionProfile(sessionID)

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"session_id": sessionID,
		"status":     "closed",
	})
}

// StartLiveRecording handles POST /api/v1/recordings/live/start
// Starts recording user actions in a browser session.
func (h *Handler) StartLiveRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	var req StartRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if req.SessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "session_id",
		}))
		return
	}

	// Delegate to recordmode service
	resp, err := h.recordModeService.StartRecording(ctx, req.SessionID, &livecapture.RecordingConfig{
		APIHost: os.Getenv("API_HOST"),
		APIPort: os.Getenv("API_PORT"),
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to start recording")
		// Check for specific error types
		if driverErr, ok := err.(*livecapture.DriverError); ok {
			if strings.Contains(driverErr.Body, "RECORDING_IN_PROGRESS") {
				h.respondError(w, ErrConflict.WithMessage("Recording is already in progress for this session"))
				return
			}
		}
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := StartRecordingResponse{
		SessionID: resp.SessionID,
		StartedAt: resp.StartedAt,
	}

	if pb, err := protoconv.StartRecordingToProto(driverResp); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, driverResp)
}

// StopLiveRecording handles POST /api/v1/recordings/live/{sessionId}/stop
// Stops recording user actions.
func (h *Handler) StopLiveRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Delegate to recordmode service
	resp, err := h.recordModeService.StopRecording(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to stop recording")
		// Check for not found error
		if driverErr, ok := err.(*livecapture.DriverError); ok && driverErr.StatusCode == 404 {
			h.respondError(w, ErrExecutionNotFound.WithMessage("No recording in progress for this session"))
			return
		}
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Persist session profile after stopping
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile after stop")
	}

	// Map service response to handler response type
	driverResp := StopRecordingResponse{
		SessionID:   resp.SessionID,
		ActionCount: resp.ActionCount,
		StoppedAt:   resp.StoppedAt,
	}

	if pb, err := protoconv.StopRecordingToProto(driverResp); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GetRecordingStatus handles GET /api/v1/recordings/live/{sessionId}/status
// Gets the current recording status.
func (h *Handler) GetRecordingStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Delegate to recordmode service
	status, err := h.recordModeService.GetRecordingStatus(ctx, sessionID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get recording status")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := RecordingStatusResponse{
		SessionID:   status.SessionID,
		IsRecording: status.IsRecording,
		ActionCount: status.ActionCount,
		StartedAt:   status.StartedAt,
		FrameCount:  status.FrameCount,
	}

	if pb, err := protoconv.RecordingStatusToProto(driverResp); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GetRecordedActions handles GET /api/v1/recordings/live/{sessionId}/actions
// Gets all recorded actions for a session.
func (h *Handler) GetRecordedActions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Check for clear query param
	clearActions := r.URL.Query().Get("clear") == "true"

	// Delegate to recordmode service
	resp, err := h.recordModeService.GetRecordedActions(ctx, sessionID, clearActions)
	if err != nil {
		h.log.WithError(err).Error("Failed to get recorded actions")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := &GetActionsResponse{
		SessionID:   resp.SessionID,
		IsRecording: resp.IsRecording,
		Actions:     resp.Actions,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// GenerateWorkflowFromRecording handles POST /api/v1/recordings/live/{sessionId}/generate-workflow
// Converts recorded actions into a workflow.
func (h *Handler) GenerateWorkflowFromRecording(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req GenerateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("Recorded Workflow %s", time.Now().Format("2006-01-02 15:04"))
	}

	if req.ProjectID == nil || *req.ProjectID == uuid.Nil {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "project_id"}))
		return
	}
	projectID := *req.ProjectID

	// Persist session profile before workflow generation
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile before workflow generation")
	}

	// Delegate workflow generation to the recordmode service
	// The service handles action fetching, merging, and smart wait insertion
	var actionRange *livecapture.ActionRange
	if req.ActionRange != nil {
		actionRange = &livecapture.ActionRange{
			Start: req.ActionRange.Start,
			End:   req.ActionRange.End,
		}
	}

	genResult, err := h.recordModeService.GenerateWorkflow(ctx, sessionID, &livecapture.GenerateWorkflowConfig{
		Name:        req.Name,
		Actions:     req.Actions, // Pass through user-edited actions if provided
		ActionRange: actionRange,
	})
	if err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Error("Failed to generate workflow from recording")
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Build V2 flow definition for storage
	v2, err := workflowservice.BuildFlowDefinitionV2ForWrite(genResult.FlowDefinition, nil, nil)
	if err != nil {
		h.respondError(w, ErrInvalidWorkflowPayload.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	// Create the workflow via catalog service
	createResp, err := h.workflowCatalog.CreateWorkflow(ctx, &basapi.CreateWorkflowRequest{
		ProjectId:      projectID.String(),
		Name:           req.Name,
		FolderPath:     "/",
		FlowDefinition: v2,
	})
	if err != nil || createResp == nil || createResp.Workflow == nil {
		h.log.WithError(err).Error("Failed to create workflow from recording")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create workflow: " + err.Error(),
		}))
		return
	}

	respPayload := GenerateWorkflowResponse{
		WorkflowID:  uuid.MustParse(createResp.Workflow.Id),
		ProjectID:   projectID,
		Name:        createResp.Workflow.Name,
		NodeCount:   genResult.NodeCount,
		ActionCount: genResult.ActionCount,
	}
	if pb, err := protoconv.GenerateWorkflowToProto(respPayload); err == nil && pb != nil {
		h.respondProto(w, http.StatusCreated, pb)
		return
	}
	h.respondSuccess(w, http.StatusCreated, respPayload)
}

// ReceiveRecordingAction handles POST /api/v1/recordings/live/{sessionId}/action
// Receives a streamed action from the playwright-driver and broadcasts it via WebSocket.
// The action is converted to a unified TimelineEntry for V2 format compatibility.
func (h *Handler) ReceiveRecordingAction(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var action RecordedAction
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Convert to unified TimelineEntry format for V2 migration
	timelineEntry := h.convertRecordedActionToTimelineEntry(&action)

	// Broadcast with both legacy action and timeline_entry
	if timelineEntry != nil {
		h.wsHub.BroadcastRecordingActionWithTimeline(sessionID, action, timelineEntry)
	} else {
		// Fallback to legacy format if conversion fails
		h.wsHub.BroadcastRecordingAction(sessionID, action)
	}

	h.log.WithFields(map[string]interface{}{
		"session_id":   sessionID,
		"action_type":  action.ActionType,
		"sequence_num": action.SequenceNum,
	}).Debug("Received and broadcast recording action")

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// convertRecordedActionToTimelineEntry converts a RecordedAction to a TimelineEntry proto,
// then to a map for JSON serialization over WebSocket.
func (h *Handler) convertRecordedActionToTimelineEntry(action *RecordedAction) map[string]any {
	// RecordedAction is now an alias for events.RecordedAction, so we can pass it directly
	timelineEntry := events.RecordedActionToTimelineEntry(action)
	if timelineEntry == nil {
		return nil
	}

	// Marshal to JSON using protojson (snake_case for consistency)
	jsonBytes, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(timelineEntry)
	if err != nil {
		h.log.WithError(err).Debug("Failed to marshal TimelineEntry to JSON")
		return nil
	}

	// Unmarshal to map for inclusion in WebSocket message
	var result map[string]any
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		h.log.WithError(err).Debug("Failed to unmarshal TimelineEntry JSON to map")
		return nil
	}

	return result
}

// ReceiveRecordingFrame handles POST /api/v1/recordings/live/{sessionId}/frame
// Receives a streamed frame from the playwright-driver and broadcasts it via WebSocket.
// This eliminates the need for clients to poll for frames - they receive them in real-time.
func (h *Handler) ReceiveRecordingFrame(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var frame websocket.RecordingFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Only broadcast if there are subscribers (avoid unnecessary work)
	if h.wsHub.HasRecordingSubscribers(sessionID) {
		h.wsHub.BroadcastRecordingFrame(sessionID, &frame)
	}

	// Return 200 immediately - don't wait for broadcast
	w.WriteHeader(http.StatusOK)
}

// HandleDriverFrameStream handles WebSocket connection for binary frame streaming from playwright-driver.
// GET /ws/recording/{sessionId}/frames
// This is more efficient than HTTP POST as it:
// 1. Uses a persistent connection (no per-frame TCP overhead)
// 2. Sends raw binary JPEG data (no base64 encoding = 33% smaller)
// 3. Pass-through to browser clients (no JSON parsing/re-encoding)
//
// When performance mode is enabled, frames may include a performance header:
// [4 bytes: header length (uint32 big-endian)][N bytes: JSON perf header][remaining: JPEG data]
// Detection: If first 2 bytes are 0xFF 0xD8 (JPEG magic), no header present.
func (h *Handler) HandleDriverFrameStream(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.log.Error("Missing sessionId in driver frame stream request")
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to upgrade driver frame stream connection")
		return
	}
	defer conn.Close()

	h.log.WithField("session_id", sessionID).Info("Driver frame stream connected")

	// Get performance config (used for logging/broadcast intervals)
	cfg := config.Load()

	// Get or create performance collector for this session (always, for potential runtime enabling)
	collector := h.perfRegistry.GetOrCreate(sessionID)

	// Read binary frames from driver and broadcast to browser clients
	for {
		receiveStart := time.Now()

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// Check for normal closure
			if websocket.IsCloseError(err) {
				h.log.WithField("session_id", sessionID).Debug("Driver frame stream closed normally")
			} else {
				h.log.WithError(err).WithField("session_id", sessionID).Warn("Driver frame stream read error")
			}
			break
		}

		// Only process binary messages (JPEG data)
		if messageType != websocket.BinaryMessage {
			continue
		}

		receiveMs := float64(time.Since(receiveStart).Microseconds()) / 1000.0

		// Parse performance header if present
		// Detection: JPEG files start with 0xFF 0xD8 magic bytes
		var driverHeader *performance.FrameHeader
		frameData := data
		if len(data) > 4 && !(data[0] == 0xFF && data[1] == 0xD8) {
			// Has performance header - parse it
			headerLen := binary.BigEndian.Uint32(data[:4])
			if int(headerLen) <= len(data)-4 {
				headerJSON := data[4 : 4+headerLen]
				var header performance.FrameHeader
				if err := json.Unmarshal(headerJSON, &header); err == nil {
					driverHeader = &header
					frameData = data[4+headerLen:]
				} else {
					h.log.WithError(err).Debug("Failed to parse frame perf header")
				}
			}
		}

		// Broadcast binary frame to subscribed browser clients
		broadcastStart := time.Now()
		if h.wsHub.HasRecordingSubscribers(sessionID) {
			h.wsHub.BroadcastBinaryFrame(sessionID, frameData)
		}
		broadcastMs := float64(time.Since(broadcastStart).Microseconds()) / 1000.0

		// Record performance data if driver sent a perf header
		// (presence of header indicates per-session perf mode is enabled)
		if driverHeader != nil {
			timing := &performance.FrameTimings{
				FrameID:         driverHeader.FrameID,
				SessionID:       sessionID,
				Timestamp:       time.Now(),
				DriverCaptureMs: driverHeader.CaptureMs,
				DriverCompareMs: driverHeader.CompareMs,
				DriverWsSendMs:  driverHeader.WsSendMs,
				DriverTotalMs:   driverHeader.CaptureMs + driverHeader.CompareMs + driverHeader.WsSendMs,
				APIReceiveMs:    receiveMs,
				APIBroadcastMs:  broadcastMs,
				APITotalMs:      receiveMs + broadcastMs,
				FrameBytes:      len(frameData),
				Skipped:         false,
			}
			collector.Record(timing)

			// Broadcast perf stats periodically (every 60 frames by default)
			if cfg.Performance.StreamToWebSocket && collector.ShouldBroadcast() {
				stats := collector.GetAggregated()
				h.wsHub.BroadcastPerfStats(sessionID, stats)

				// Log summary if enabled
				if cfg.Performance.LogSummaryInterval > 0 {
					h.log.WithFields(map[string]interface{}{
						"session_id":      sessionID,
						"frame_count":     stats.FrameCount,
						"capture_p50_ms":  stats.CaptureP50Ms,
						"capture_p90_ms":  stats.CaptureP90Ms,
						"e2e_p50_ms":      stats.E2EP50Ms,
						"e2e_p90_ms":      stats.E2EP90Ms,
						"actual_fps":      stats.ActualFps,
						"target_fps":      stats.TargetFps,
						"bottleneck":      stats.PrimaryBottleneck,
						"bandwidth_bps":   stats.BandwidthBytesPerSec,
						"avg_frame_bytes": stats.AvgFrameBytes,
					}).Info("recording: frame perf summary")
				}
			}
		}
	}

	// Cleanup collector when stream disconnects
	h.perfRegistry.Remove(sessionID)

	h.log.WithField("session_id", sessionID).Info("Driver frame stream disconnected")
}

// ValidateSelector handles POST /api/v1/recordings/live/{sessionId}/validate-selector
// Validates a selector on the current page.
func (h *Handler) ValidateSelector(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req struct {
		Selector string `json:"selector"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if req.Selector == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "selector",
		}))
		return
	}

	// Delegate to recordmode service
	resp, err := h.recordModeService.ValidateSelector(ctx, sessionID, req.Selector)
	if err != nil {
		h.log.WithError(err).Error("Failed to validate selector")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := struct {
		Valid      bool   `json:"valid"`
		MatchCount int    `json:"match_count"`
		Selector   string `json:"selector"`
		Error      string `json:"error,omitempty"`
	}{
		Valid:      resp.Valid,
		MatchCount: resp.MatchCount,
		Selector:   resp.Selector,
		Error:      resp.Error,
	}

	if pb, err := protoconv.SelectorValidationToProto(driverResp); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, driverResp)
}

// ReplayRecordingPreview handles POST /api/v1/recordings/live/{sessionId}/replay-preview
// Tests recorded actions by replaying them in the browser.
func (h *Handler) ReplayRecordingPreview(w http.ResponseWriter, r *http.Request) {
	// Longer timeout for replay operations - may need to execute multiple actions
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req ReplayPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if len(req.Actions) == 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "No actions to replay",
		}))
		return
	}

	// Delegate to recordmode service
	svcReq := &livecapture.ReplayPreviewRequest{
		Actions:       req.Actions,
		Limit:         req.Limit,
		StopOnFailure: req.StopOnFailure,
		ActionTimeout: req.ActionTimeout,
	}

	resp, err := h.recordModeService.ReplayPreview(ctx, sessionID, svcReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to replay preview")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	results := make([]ActionReplayResult, len(resp.Results))
	for i, r := range resp.Results {
		var replayErr *ActionReplayError
		if r.Error != "" {
			replayErr = &ActionReplayError{Message: r.Error}
		}
		results[i] = ActionReplayResult{
			SequenceNum: r.Index,
			ActionType:  r.ActionType,
			Success:     r.Success,
			DurationMs:  r.DurationMs,
			Error:       replayErr,
		}
	}

	replayResp := ReplayPreviewResponse{
		Success:         resp.Success,
		TotalActions:    len(req.Actions),
		PassedActions:   resp.PassedActions,
		FailedActions:   resp.FailedActions,
		Results:         results,
		TotalDurationMs: resp.TotalDurationMs,
		StoppedEarly:    resp.FailedActions > 0 && req.StopOnFailure != nil && *req.StopOnFailure,
	}

	h.log.WithFields(map[string]interface{}{
		"session_id":     sessionID,
		"success":        replayResp.Success,
		"passed_actions": replayResp.PassedActions,
		"failed_actions": replayResp.FailedActions,
		"total_duration": replayResp.TotalDurationMs,
	}).Info("Replay preview complete")

	if pb, err := protoconv.ReplayPreviewToProto(replayResp); err == nil && pb != nil {
		h.respondProto(w, http.StatusOK, pb)
		return
	}
	h.respondSuccess(w, http.StatusOK, replayResp)
}

// NavigateRecordingSession handles POST /api/v1/recordings/live/{sessionId}/navigate
// Navigates the Playwright recording session to a URL and optionally returns a screenshot.
func (h *Handler) NavigateRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var req NavigateRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if strings.TrimSpace(req.URL) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "url",
		}))
		return
	}

	// Delegate to recordmode service
	resp, err := h.recordModeService.Navigate(ctx, sessionID, &livecapture.NavigateRequest{
		URL:       req.URL,
		WaitUntil: req.WaitUntil,
		TimeoutMs: req.TimeoutMs,
		Capture:   req.Capture,
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to navigate recording session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := NavigateRecordingResponse{
		URL:        resp.URL,
		Title:      resp.Title,
		StatusCode: resp.StatusCode,
		Screenshot: resp.Screenshot,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// CaptureRecordingScreenshot handles POST /api/v1/recordings/live/{sessionId}/screenshot
// Captures a screenshot from the current recording page.
func (h *Handler) CaptureRecordingScreenshot(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Parse optional request body for format/quality
	var reqBody struct {
		Format  string `json:"format,omitempty"`
		Quality int    `json:"quality,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate to recordmode service
	svcReq := &livecapture.CaptureScreenshotRequest{
		Format:  reqBody.Format,
		Quality: reqBody.Quality,
	}

	resp, err := h.recordModeService.CaptureScreenshot(ctx, sessionID, svcReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to capture screenshot")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := RecordingScreenshotResponse{
		SessionID:  sessionID,
		Screenshot: resp.Data,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// UpdateRecordingViewport handles POST /api/v1/recordings/live/{sessionId}/viewport
// Updates the viewport dimensions for the active recording session.
func (h *Handler) UpdateRecordingViewport(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var reqBody RecordingViewportRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if reqBody.Width <= 0 || reqBody.Height <= 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "width and height must be positive integers",
		}))
		return
	}

	// Delegate to recordmode service
	resp, err := h.recordModeService.UpdateViewport(ctx, sessionID, reqBody.Width, reqBody.Height)
	if err != nil {
		h.log.WithError(err).Error("Failed to update recording viewport")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response type
	driverResp := struct {
		SessionID string `json:"session_id"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}{
		SessionID: resp.SessionID,
		Width:     resp.Width,
		Height:    resp.Height,
	}

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// UpdateStreamSettings handles POST /api/v1/recordings/live/{sessionId}/stream-settings
// Updates stream settings (quality, fps) for an active recording session.
// Quality and FPS can be updated immediately. Scale changes require a new session.
func (h *Handler) UpdateStreamSettings(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	var reqBody UpdateStreamSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	// Delegate to recordmode service
	svcReq := &livecapture.UpdateStreamSettingsRequest{
		Quality:  reqBody.Quality,
		FPS:      reqBody.FPS,
		Scale:    reqBody.Scale,
		PerfMode: reqBody.PerfMode,
	}

	resp, err := h.recordModeService.UpdateStreamSettings(ctx, sessionID, svcReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to update stream settings")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := UpdateStreamSettingsResponse{
		SessionID:    resp.SessionID,
		Quality:      resp.Quality,
		FPS:          resp.FPS,
		CurrentFPS:   resp.CurrentFPS,
		Scale:        resp.Scale,
		IsStreaming:  resp.IsStreaming,
		Updated:      resp.Updated,
		ScaleWarning: resp.ScaleWarning,
		PerfMode:     resp.PerfMode,
	}

	h.log.WithFields(map[string]interface{}{
		"session_id":  sessionID,
		"quality":     driverResp.Quality,
		"fps":         driverResp.FPS,
		"current_fps": driverResp.CurrentFPS,
		"updated":     driverResp.Updated,
	}).Debug("Stream settings updated")

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// ForwardRecordingInput handles POST /api/v1/recordings/live/{sessionId}/input
// Forwards pointer/keyboard/wheel events to the Playwright driver.
func (h *Handler) ForwardRecordingInput(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Failed to read request body: " + err.Error(),
		}))
		return
	}
	if len(bodyBytes) == 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Empty request body",
		}))
		return
	}

	// Delegate to recordmode service
	err = h.recordModeService.ForwardInput(ctx, sessionID, bodyBytes)
	if err != nil {
		h.log.WithError(err).Error("Failed to forward recording input")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// GetRecordingFrame handles GET /api/v1/recordings/live/{sessionId}/frame
// Retrieves a lightweight frame preview from the driver.
// Supports ETag-based caching to skip identical frames (If-None-Match header).
func (h *Handler) GetRecordingFrame(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	// Delegate to recordmode service, passing query params for filtering
	resp, err := h.recordModeService.GetFrame(ctx, sessionID, r.URL.RawQuery)
	if err != nil {
		h.log.WithError(err).Error("Failed to get frame")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	// Map service response to handler response
	driverResp := RecordingFrameResponse{
		SessionID:   sessionID,
		Mime:        resp.MediaType,
		Image:       resp.Data,
		Width:       resp.Width,
		Height:      resp.Height,
		CapturedAt:  resp.CapturedAt,
		ContentHash: resp.ContentHash,
		PageTitle:   resp.PageTitle,
		PageURL:     resp.PageURL,
	}

	// Generate ETag from content hash provided by playwright-driver.
	// The driver computes MD5 hash of raw JPEG buffer, which is a reliable
	// content fingerprint that changes if and only if the frame content changes.
	var etag string
	if driverResp.ContentHash != "" {
		etag = fmt.Sprintf(`"%s"`, driverResp.ContentHash)
	} else {
		// Fallback for older driver versions without content_hash field
		etag = fmt.Sprintf(`"%s"`, driverResp.CapturedAt)
	}

	// Check If-None-Match header for conditional request
	clientETag := r.Header.Get("If-None-Match")
	if clientETag != "" && clientETag == etag {
		// Frame hasn't changed, return 304 Not Modified
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Set ETag header for client caching
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")

	h.respondSuccess(w, http.StatusOK, driverResp)
}

// PersistRecordingSession handles POST /api/v1/recordings/live/{sessionId}/persist
// Captures current storage state and saves it to the active session profile without closing the session.
func (h *Handler) PersistRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "sessionId",
		}))
		return
	}

	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status":     "persisted",
		"session_id": sessionID,
	})
}

func (h *Handler) resolveSessionProfile(requestedID string) (*archiveingestion.SessionProfile, *APIError) {
	if h == nil || h.sessionProfiles == nil {
		return nil, nil
	}

	id := strings.TrimSpace(requestedID)
	if id != "" {
		profile, err := h.sessionProfiles.Get(id)
		if err != nil {
			return nil, ErrExecutionNotFound.WithMessage("Session profile not found")
		}
		return profile, nil
	}

	profile, err := h.sessionProfiles.MostRecent()
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Error("Failed to list session profiles")
		}
		return nil, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to load session profiles",
		})
	}

	if profile != nil {
		return profile, nil
	}

	profile, err = h.sessionProfiles.Create("")
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Error("Failed to create default session profile")
		}
		return nil, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create session profile",
		})
	}
	return profile, nil
}

func (h *Handler) setActiveSessionProfile(sessionID, profileID string) {
	h.activeSessionsMu.Lock()
	defer h.activeSessionsMu.Unlock()
	if h.activeSessions == nil {
		h.activeSessions = make(map[string]string)
	}
	if sessionID != "" && profileID != "" {
		h.activeSessions[sessionID] = profileID
	}
}

func (h *Handler) clearActiveSessionProfile(sessionID string) string {
	h.activeSessionsMu.Lock()
	defer h.activeSessionsMu.Unlock()
	if h.activeSessions == nil {
		return ""
	}
	profileID := h.activeSessions[sessionID]
	delete(h.activeSessions, sessionID)
	return profileID
}

func (h *Handler) getActiveSessionProfile(sessionID string) string {
	h.activeSessionsMu.Lock()
	defer h.activeSessionsMu.Unlock()
	if h.activeSessions == nil {
		return ""
	}
	return h.activeSessions[sessionID]
}

func (h *Handler) persistSessionProfile(ctx context.Context, sessionID string) error {
	if h.sessionProfiles == nil {
		return nil
	}
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID == "" {
		return nil
	}

	// Delegate to recordmode service for storage state
	state, err := h.recordModeService.GetStorageState(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(state) == 0 {
		return nil
	}

	_, err = h.sessionProfiles.SaveStorageState(profileID, state)
	return err
}

// CreateInputForwarder returns a function that forwards input events to the playwright-driver.
// This is used by the WebSocket hub to forward input messages without going through HTTP.
//
// Performance optimizations:
// - Uses a shared HTTP client with connection pooling (reuses TCP connections)
// - Keep-alive connections reduce latency by avoiding TCP handshake per request
// - Connection pool sized for concurrent input events across sessions
func (h *Handler) CreateInputForwarder() func(sessionID string, input map[string]any) error {
	// Shared HTTP client with connection pooling for all input forwarding.
	// This dramatically reduces latency vs creating a new client per request.
	// Keep-alive connections mean subsequent requests reuse existing TCP connections.
	transport := &http.Transport{
		MaxIdleConns:        100,              // Total pool size
		MaxIdleConnsPerHost: 10,               // Per-driver connections (usually just one driver)
		IdleConnTimeout:     90 * time.Second, // Keep connections warm
		DisableKeepAlives:   false,            // Explicitly enable keep-alive
	}
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}

	return func(sessionID string, input map[string]any) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second) // Tighter timeout for input
		defer cancel()

		driverURL := fmt.Sprintf("%s/session/%s/record/input", getPlaywrightDriverURL(), sessionID)

		jsonBody, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("marshal input: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(httpReq)
		if err != nil {
			return fmt.Errorf("forward input: %w", err)
		}
		defer resp.Body.Close()

		// Drain the response body to enable connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("driver returned %d", resp.StatusCode)
		}

		return nil
	}
}
