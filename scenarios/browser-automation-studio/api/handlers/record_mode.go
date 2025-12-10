package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/internal/protoconv"
	"github.com/vrooli/browser-automation-studio/services/recording"
	"github.com/vrooli/browser-automation-studio/websocket"
)

// RecordedAction represents a single user action captured during recording.
// This mirrors the TypeScript RecordedAction type from playwright-driver.
type RecordedAction struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int                    `json:"durationMs,omitempty"`
	ActionType  string                 `json:"actionType"`
	Confidence  float64                `json:"confidence"`
	Selector    *SelectorSet           `json:"selector,omitempty"`
	ElementMeta *ElementMeta           `json:"elementMeta,omitempty"`
	BoundingBox *BoundingBox           `json:"boundingBox,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	URL         string                 `json:"url"`
	FrameID     string                 `json:"frameId,omitempty"`
	CursorPos   *Point                 `json:"cursorPos,omitempty"`
}

// SelectorSet contains multiple selector strategies for resilience.
type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates"`
}

// SelectorCandidate is a single selector with metadata.
type SelectorCandidate struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Specificity int     `json:"specificity"`
}

// ElementMeta captures information about the target element.
type ElementMeta struct {
	TagName    string            `json:"tagName"`
	ID         string            `json:"id,omitempty"`
	ClassName  string            `json:"className,omitempty"`
	InnerText  string            `json:"innerText,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	IsVisible  bool              `json:"isVisible"`
	IsEnabled  bool              `json:"isEnabled"`
	Role       string            `json:"role,omitempty"`
	AriaLabel  string            `json:"ariaLabel,omitempty"`
}

// BoundingBox for element position on screen.
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Point for cursor/click positions.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// CreateRecordingSessionRequest is the request body for creating a browser session for recording.
type CreateRecordingSessionRequest struct {
	// Viewport dimensions (optional, defaults to 1280x720)
	ViewportWidth  int `json:"viewport_width,omitempty"`
	ViewportHeight int `json:"viewport_height,omitempty"`
	// Initial URL to navigate to (optional)
	InitialURL string `json:"initial_url,omitempty"`
	// Optional persisted session profile to load cookies/storage from
	SessionProfileID string `json:"session_profile_id,omitempty"`
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
	StartedAt   string `json:"started_at,omitempty"`
}

// GetActionsResponse is the response for getting recorded actions.
type GetActionsResponse struct {
	SessionID string           `json:"session_id"`
	Actions   []RecordedAction `json:"actions"`
	Count     int              `json:"count"`
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
	ContentHash string `json:"content_hash"` // MD5 hash of raw frame buffer for reliable ETag
}

// RecordingViewportRequest updates viewport dimensions.
type RecordingViewportRequest struct {
	Width  int `json:"width"`
	Height int `json:"height"`
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

	// Set default viewport if not provided
	viewportWidth := req.ViewportWidth
	viewportHeight := req.ViewportHeight
	if viewportWidth <= 0 {
		viewportWidth = 1280
	}
	if viewportHeight <= 0 {
		viewportHeight = 720
	}

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

	// Build request to playwright-driver
	driverURL := fmt.Sprintf("%s/session/start", getPlaywrightDriverURL())

	// Generate IDs for the recording session
	// For record mode, we create ephemeral IDs since this isn't tied to a stored workflow/execution
	recordingExecutionID := uuid.New().String()
	recordingWorkflowID := uuid.New().String()

	// Construct frame callback URL for live preview streaming
	// Frame streaming starts immediately when session is created (not when recording starts)
	// The driver extracts the host from this URL and builds the WebSocket URL with the actual session ID
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	frameCallbackURL := fmt.Sprintf("http://%s:%s/api/v1/recordings/live/placeholder/frame", apiHost, apiPort)

	driverReq := map[string]interface{}{
		"execution_id": recordingExecutionID,
		"workflow_id":  recordingWorkflowID,
		"viewport": map[string]int{
			"width":  viewportWidth,
			"height": viewportHeight,
		},
		"reuse_mode": "fresh",
		"labels": map[string]string{
			"purpose": "record-mode",
		},
		// Enable frame streaming immediately for live preview
		"frame_streaming": map[string]interface{}{
			"callback_url": frameCallbackURL,
			"quality":      65,
			"fps":          6,
		},
	}
	if len(storageState) > 0 {
		// Pass through persisted storage state to reuse authentication
		driverReq["storage_state"] = json.RawMessage(storageState)
	}

	jsonBody, err := json.Marshal(driverReq)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to marshal request",
		}))
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for create session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.log.WithFields(map[string]interface{}{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Driver returned error for create session")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
		}))
		return
	}

	var driverResp struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
	}

	if profileID != "" && h.sessionProfiles != nil {
		if updated, err := h.sessionProfiles.Touch(profileID); err != nil {
			h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to update session profile usage")
		} else if updated != nil {
			profileName = updated.Name
			profileLastUsed = updated.LastUsedAt.Format(time.RFC3339)
		}
		h.setActiveSessionProfile(driverResp.SessionID, profileID)
	}

	// If initial URL provided, navigate to it
	if req.InitialURL != "" {
		navURL := fmt.Sprintf("%s/session/%s/run", getPlaywrightDriverURL(), driverResp.SessionID)
		navReq := map[string]interface{}{
			"instructions": []map[string]interface{}{
				{
					"op":  "navigate",
					"url": req.InitialURL,
				},
			},
		}
		navBody, _ := json.Marshal(navReq)
		navHTTPReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, navURL, bytes.NewReader(navBody))
		navHTTPReq.Header.Set("Content-Type", "application/json")
		navResp, err := client.Do(navHTTPReq)
		if err != nil {
			h.log.WithError(err).Warn("Failed to navigate to initial URL")
		} else {
			navResp.Body.Close()
		}
	}

	response := CreateRecordingSessionResponse{
		SessionID:          driverResp.SessionID,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
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

	var storageState json.RawMessage
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID != "" && h.sessionProfiles != nil {
		if state, err := h.fetchSessionStorageState(ctx, sessionID); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"session_id": sessionID,
				"profile_id": profileID,
			}).Warn("Failed to capture storage state before closing session")
		} else {
			storageState = state
		}
	}

	// Call playwright-driver to close the session
	driverURL := fmt.Sprintf("%s/session/%s/close", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for close session")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		h.respondError(w, ErrExecutionNotFound.WithMessage("Session not found"))
		return
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
		}))
		return
	}

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

	// Call playwright-driver to start recording
	driverURL := fmt.Sprintf("%s/session/%s/record/start", getPlaywrightDriverURL(), req.SessionID)

	// Construct callback URLs for real-time streaming
	// The driver will POST each action/frame to these URLs for WebSocket broadcasting
	apiHost := os.Getenv("API_HOST")
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		apiPort = "8080"
	}
	actionCallbackURL := fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/action", apiHost, apiPort, req.SessionID)
	frameCallbackURL := fmt.Sprintf("http://%s:%s/api/v1/recordings/live/%s/frame", apiHost, apiPort, req.SessionID)

	reqBody := map[string]interface{}{
		"callback_url":       actionCallbackURL,
		"frame_callback_url": frameCallbackURL,
		"frame_quality":      65, // WebP quality for live preview
		"frame_fps":          6,  // 6 FPS for low-latency streaming
	}
	// Allow override from request if provided
	if req.CallbackURL != "" {
		reqBody["callback_url"] = req.CallbackURL
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to marshal request",
		}))
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for start recording")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errMsg, ok := errResp["error"].(string); ok && errMsg == "RECORDING_IN_PROGRESS" {
				h.respondError(w, ErrConflict.WithMessage("Recording is already in progress for this session"))
				return
			}
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
		}))
		return
	}

	var driverResp StartRecordingResponse
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	// Call playwright-driver to stop recording
	driverURL := fmt.Sprintf("%s/session/%s/record/stop", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, nil)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for stop recording")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		h.respondError(w, ErrExecutionNotFound.WithMessage("No recording in progress for this session"))
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
		}))
		return
	}

	var driverResp StopRecordingResponse
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
	}

	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile after stop")
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

	// Call playwright-driver to get status
	driverURL := fmt.Sprintf("%s/session/%s/record/status", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, driverURL, nil)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for recording status")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
		}))
		return
	}

	var driverResp RecordingStatusResponse
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	// Call playwright-driver to get actions
	driverURL := fmt.Sprintf("%s/session/%s/record/actions", getPlaywrightDriverURL(), sessionID)
	if clearActions {
		driverURL += "?clear=true"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, driverURL, nil)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for recorded actions")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
		}))
		return
	}

	var driverResp GetActionsResponse
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
	}
	if driverResp.Actions == nil {
		driverResp.Actions = []RecordedAction{}
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

	var actions []RecordedAction

	// Use actions from request if provided (these contain user edits)
	if len(req.Actions) > 0 {
		actions = req.Actions
	} else {
		// Fall back to fetching from driver
		actionsURL := fmt.Sprintf("%s/session/%s/record/actions", getPlaywrightDriverURL(), sessionID)

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, actionsURL, nil)
		if err != nil {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": "Failed to create request to driver",
			}))
			return
		}

		client := &http.Client{Timeout: recordModeTimeout}
		resp, err := client.Do(httpReq)
		if err != nil {
			h.log.WithError(err).Error("Failed to get recorded actions from driver")
			h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
				"error": "Playwright driver unavailable",
			}))
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != http.StatusOK {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": "Failed to get recorded actions",
			}))
			return
		}

		var actionsResp GetActionsResponse
		if err := json.Unmarshal(body, &actionsResp); err != nil {
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": "Failed to parse actions response",
			}))
			return
		}

		actions = actionsResp.Actions
	}

	// Apply action range filter if specified
	if req.ActionRange != nil {
		start := req.ActionRange.Start
		end := req.ActionRange.End
		if start < 0 {
			start = 0
		}
		if end >= len(actions) {
			end = len(actions) - 1
		}
		if start <= end && start < len(actions) {
			actions = actions[start : end+1]
		}
	}

	if len(actions) == 0 {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "No actions to convert to workflow",
		}))
		return
	}

	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile before workflow generation")
	}

	// Convert actions to workflow nodes
	flowDefinition := convertActionsToWorkflow(actions)

	// Create the workflow using existing service
	workflow, err := h.workflowCatalog.CreateWorkflowWithProject(
		ctx,
		req.ProjectID,
		req.Name,
		"/",
		flowDefinition,
		"", // no AI prompt
	)
	if err != nil {
		h.log.WithError(err).Error("Failed to create workflow from recording")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create workflow: " + err.Error(),
		}))
		return
	}

	var projectID uuid.UUID
	if workflow.ProjectID != nil {
		projectID = *workflow.ProjectID
	}

	respPayload := GenerateWorkflowResponse{
		WorkflowID:  workflow.ID,
		ProjectID:   projectID,
		Name:        workflow.Name,
		NodeCount:   len(flowDefinition["nodes"].([]map[string]interface{})),
		ActionCount: len(actions),
	}
	if pb, err := protoconv.GenerateWorkflowToProto(respPayload); err == nil && pb != nil {
		h.respondProto(w, http.StatusCreated, pb)
		return
	}
	h.respondSuccess(w, http.StatusCreated, respPayload)
}

// mergeConsecutiveActions optimizes recorded actions by merging:
// - Consecutive type actions on the same selector (text is concatenated)
// - Consecutive scroll actions (uses final scroll position)
// - Removes focus events that precede type events on the same element
func mergeConsecutiveActions(actions []RecordedAction) []RecordedAction {
	if len(actions) <= 1 {
		return actions
	}

	merged := make([]RecordedAction, 0, len(actions))

	for i := 0; i < len(actions); i++ {
		action := actions[i]

		// Skip focus events that are immediately followed by type on the same element
		if action.ActionType == "focus" && i+1 < len(actions) {
			next := actions[i+1]
			if next.ActionType == "type" && selectorsMatch(action.Selector, next.Selector) {
				continue // Skip this focus event
			}
		}

		// Merge consecutive type actions on same selector
		if action.ActionType == "type" && action.Selector != nil {
			mergedText := ""
			if action.Payload != nil {
				if text, ok := action.Payload["text"].(string); ok {
					mergedText = text
				}
			}

			// Look ahead for more type actions on same element
			for i+1 < len(actions) {
				next := actions[i+1]
				if next.ActionType != "type" || !selectorsMatch(action.Selector, next.Selector) {
					break
				}
				// Merge the text
				if next.Payload != nil {
					if text, ok := next.Payload["text"].(string); ok {
						mergedText += text
					}
				}
				i++ // Skip this action, we've merged it
			}

			// Update the action with merged text
			if mergedText != "" {
				if action.Payload == nil {
					action.Payload = make(map[string]interface{})
				}
				action.Payload["text"] = mergedText
			}
		}

		// Merge consecutive scroll actions
		if action.ActionType == "scroll" {
			var finalScrollY float64
			if action.Payload != nil {
				if y, ok := action.Payload["scrollY"].(float64); ok {
					finalScrollY = y
				}
			}

			// Look ahead for more scroll actions
			for i+1 < len(actions) {
				next := actions[i+1]
				if next.ActionType != "scroll" {
					break
				}
				// Use the final scroll position
				if next.Payload != nil {
					if y, ok := next.Payload["scrollY"].(float64); ok {
						finalScrollY = y
					}
				}
				i++ // Skip this action, we've merged it
			}

			// Update the action with final scroll position
			if action.Payload == nil {
				action.Payload = make(map[string]interface{})
			}
			action.Payload["scrollY"] = finalScrollY
		}

		merged = append(merged, action)
	}

	return merged
}

// selectorsMatch checks if two SelectorSets refer to the same element
func selectorsMatch(a, b *SelectorSet) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Primary == b.Primary
}

// convertActionsToWorkflow converts recorded actions to a workflow flow definition.
// It applies action merging and inserts smart wait nodes to improve reliability.
func convertActionsToWorkflow(actions []RecordedAction) map[string]interface{} {
	// First, merge consecutive actions for cleaner workflows
	mergedActions := mergeConsecutiveActions(actions)

	// Insert smart wait nodes between actions that need them
	nodes, edges := insertSmartWaits(mergedActions)

	return map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}
}

// mapActionToNode converts a single recorded action to a workflow node.
func mapActionToNode(action RecordedAction, nodeID string, index int) map[string]interface{} {
	// Calculate position (vertical layout)
	posX := 250.0
	posY := float64(100 + index*120)

	node := map[string]interface{}{
		"id": nodeID,
		"position": map[string]interface{}{
			"x": posX,
			"y": posY,
		},
	}

	var nodeType string
	data := map[string]interface{}{}
	config := map[string]interface{}{}

	switch action.ActionType {
	case "click":
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
			config["click"] = map[string]any{
				"selector": action.Selector.Primary,
			}
		}
		if action.Payload != nil {
			if btn, ok := action.Payload["button"]; ok {
				data["button"] = btn
				ensureConfig(config, "click")["button"] = btn
			}
			if mods, ok := action.Payload["modifiers"]; ok {
				data["modifiers"] = mods
			}
			if count, ok := action.Payload["clickCount"]; ok {
				ensureConfig(config, "click")["click_count"] = count
			}
			if delay, ok := action.Payload["delay"]; ok {
				ensureConfig(config, "click")["delay_ms"] = delay
			}
		}
		data["label"] = generateClickLabel(action)

	case "type":
		nodeType = "type"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
			config["input"] = map[string]any{
				"selector": action.Selector.Primary,
			}
		}
		if action.Payload != nil {
			if text, ok := action.Payload["text"].(string); ok {
				data["text"] = text
				data["label"] = fmt.Sprintf("Type: %q", truncateString(text, 20))
				ensureConfig(config, "input")["value"] = text
			}
			if submit, ok := action.Payload["submit"]; ok {
				ensureConfig(config, "input")["submit"] = submit
			}
		}

	case "navigate":
		nodeType = "navigate"
		data["url"] = action.URL
		data["label"] = fmt.Sprintf("Navigate to %s", extractHostname(action.URL))
		config["navigate"] = map[string]any{
			"url": action.URL,
		}
		if action.Payload != nil {
			if waitFor, ok := action.Payload["waitForSelector"]; ok {
				ensureConfig(config, "navigate")["wait_for_selector"] = waitFor
			}
			if timeout, ok := action.Payload["timeoutMs"]; ok {
				ensureConfig(config, "navigate")["timeout_ms"] = timeout
			}
		}

	case "scroll":
		nodeType = "scroll"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		if action.Payload != nil {
			if y, ok := action.Payload["scrollY"].(float64); ok {
				data["y"] = y
			}
		}
		data["label"] = "Scroll"
		config["custom"] = map[string]any{
			"kind": "scroll",
			"payload": map[string]any{
				"y": data["y"],
			},
		}

	case "select":
		nodeType = "select"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		if action.Payload != nil {
			if val, ok := action.Payload["value"]; ok {
				data["value"] = val
			}
		}
		data["label"] = "Select option"
		config["custom"] = map[string]any{
			"kind":    "select",
			"payload": data,
		}

	case "focus":
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		data["label"] = "Focus element"
		config["click"] = map[string]any{
			"selector": action.Selector.Primary,
		}

	case "keypress":
		nodeType = "keyboard"
		if action.Payload != nil {
			if key, ok := action.Payload["key"].(string); ok {
				data["key"] = key
				data["label"] = fmt.Sprintf("Press %s", key)
			}
		}
		config["custom"] = map[string]any{
			"kind":    "keypress",
			"payload": data,
		}

	default:
		// Default to click for unknown types
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		data["label"] = action.ActionType
		config["custom"] = map[string]any{
			"kind":    action.ActionType,
			"payload": data,
		}
	}

	node["type"] = nodeType
	node["data"] = data
	if len(config) > 0 {
		node["config"] = config
	}

	return node
}

func ensureConfig(config map[string]any, key string) map[string]any {
	if existing, ok := config[key]; ok {
		if typed, ok := existing.(map[string]any); ok {
			return typed
		}
	}
	typed := map[string]any{}
	config[key] = typed
	return typed
}

// WaitTemplate describes a wait node to be inserted between actions.
type WaitTemplate struct {
	WaitType  string // "selector" or "timeout"
	Selector  string // For selector waits
	TimeoutMs int    // Timeout for selector waits, or duration for timeout waits
	Label     string // Human-readable label
}

// analyzeTransitionForWait examines two consecutive actions and determines
// if a wait node should be inserted between them.
// Returns nil if no wait is needed.
func analyzeTransitionForWait(current, next RecordedAction) *WaitTemplate {
	// Actions that need the element to exist before interaction
	needsSelectorWait := map[string]bool{
		"click":    true,
		"type":     true,
		"select":   true,
		"focus":    true,
		"hover":    true,
		"scroll":   true,
		"dragDrop": true,
	}

	// Check if the next action needs its selector to exist
	if needsSelectorWait[next.ActionType] && next.Selector != nil && next.Selector.Primary != "" {
		// If current action might trigger DOM changes, add a wait
		triggersChanges := current.ActionType == "click" ||
			current.ActionType == "type" ||
			current.ActionType == "select" ||
			current.ActionType == "navigate" ||
			current.ActionType == "keypress"

		// Check for URL change (indicates navigation happened)
		urlChanged := current.URL != next.URL

		// Check for significant time gap (>500ms suggests async activity)
		var timeDiff int64
		if current.Timestamp != "" && next.Timestamp != "" {
			currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
			nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
			if err1 == nil && err2 == nil {
				timeDiff = nextTime.Sub(currentTime).Milliseconds()
			}
		}
		significantGap := timeDiff > 500

		// Insert wait if any condition is met
		if triggersChanges || urlChanged || significantGap {
			label := fmt.Sprintf("Wait for %s", describeElement(next))
			return &WaitTemplate{
				WaitType:  "selector",
				Selector:  next.Selector.Primary,
				TimeoutMs: 10000, // 10 second default timeout
				Label:     label,
			}
		}
	}

	// Check for large time gaps that suggest async operations even without selector needs
	if current.Timestamp != "" && next.Timestamp != "" {
		currentTime, err1 := time.Parse(time.RFC3339Nano, current.Timestamp)
		nextTime, err2 := time.Parse(time.RFC3339Nano, next.Timestamp)
		if err1 == nil && err2 == nil {
			timeDiff := nextTime.Sub(currentTime).Milliseconds()
			// If gap > 2 seconds, insert a proportional wait (capped at 5 seconds)
			if timeDiff > 2000 {
				waitDuration := timeDiff / 2 // Wait for half the observed gap
				if waitDuration > 5000 {
					waitDuration = 5000
				}
				return &WaitTemplate{
					WaitType:  "timeout",
					TimeoutMs: int(waitDuration),
					Label:     "Wait for page to stabilize",
				}
			}
		}
	}

	return nil
}

// describeElement creates a human-readable description of an element for labels.
func describeElement(action RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 15)
			return fmt.Sprintf("\"%s\"", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return action.ElementMeta.AriaLabel
		}
		if action.ElementMeta.TagName != "" {
			return action.ElementMeta.TagName
		}
	}
	return "element"
}

// createWaitNode generates a workflow wait node from a WaitTemplate.
func createWaitNode(template *WaitTemplate, nodeID string, posY float64) map[string]interface{} {
	data := map[string]interface{}{
		"label":     template.Label,
		"timeoutMs": template.TimeoutMs,
	}

	if template.WaitType == "selector" && template.Selector != "" {
		data["selector"] = template.Selector
	}

	return map[string]interface{}{
		"id":   nodeID,
		"type": "wait",
		"position": map[string]interface{}{
			"x": 250.0,
			"y": posY,
		},
		"data": data,
		"config": map[string]any{
			"custom": map[string]any{
				"kind":    "wait",
				"payload": data,
			},
		},
	}
}

// insertSmartWaits analyzes action transitions and inserts wait nodes where needed.
// This improves reliability of recorded workflows by ensuring elements exist before interaction.
func insertSmartWaits(actions []RecordedAction) ([]map[string]interface{}, []map[string]interface{}) {
	if len(actions) == 0 {
		return nil, nil
	}

	nodes := make([]map[string]interface{}, 0, len(actions)*2)
	edges := make([]map[string]interface{}, 0, len(actions)*2)

	var prevNodeID string
	nodeIndex := 0
	edgeIndex := 0
	posY := 100.0
	posYIncrement := 120.0

	for i, action := range actions {
		// Create the action node
		nodeID := fmt.Sprintf("node_%d", nodeIndex+1)
		node := mapActionToNode(action, nodeID, nodeIndex)
		// Override position to account for inserted wait nodes
		node["position"] = map[string]interface{}{
			"x": 250.0,
			"y": posY,
		}
		nodes = append(nodes, node)
		posY += posYIncrement
		nodeIndex++

		// Create edge from previous node
		if prevNodeID != "" {
			edges = append(edges, map[string]interface{}{
				"id":     fmt.Sprintf("edge_%d", edgeIndex+1),
				"source": prevNodeID,
				"target": nodeID,
			})
			edgeIndex++
		}
		prevNodeID = nodeID

		// Check if we need a wait before the next action
		if i < len(actions)-1 {
			nextAction := actions[i+1]
			waitTemplate := analyzeTransitionForWait(action, nextAction)

			if waitTemplate != nil {
				// Create wait node
				waitNodeID := fmt.Sprintf("wait_%d", nodeIndex+1)
				waitNode := createWaitNode(waitTemplate, waitNodeID, posY)
				nodes = append(nodes, waitNode)
				posY += posYIncrement
				nodeIndex++

				// Create edge from action to wait
				edges = append(edges, map[string]interface{}{
					"id":     fmt.Sprintf("edge_%d", edgeIndex+1),
					"source": prevNodeID,
					"target": waitNodeID,
				})
				edgeIndex++
				prevNodeID = waitNodeID
			}
		}
	}

	return nodes, edges
}

// generateClickLabel creates a readable label for a click action.
func generateClickLabel(action RecordedAction) string {
	if action.ElementMeta != nil {
		if action.ElementMeta.InnerText != "" {
			text := truncateString(action.ElementMeta.InnerText, 20)
			return fmt.Sprintf("Click: %s", text)
		}
		if action.ElementMeta.AriaLabel != "" {
			return fmt.Sprintf("Click: %s", action.ElementMeta.AriaLabel)
		}
		return fmt.Sprintf("Click %s", action.ElementMeta.TagName)
	}
	return "Click element"
}

// truncateString truncates a string to maxLen and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractHostname extracts the hostname from a URL.
func extractHostname(urlStr string) string {
	if len(urlStr) > 50 {
		return urlStr[:50] + "..."
	}
	return urlStr
}

// ReceiveRecordingAction handles POST /api/v1/recordings/live/{sessionId}/action
// Receives a streamed action from the playwright-driver and broadcasts it via WebSocket.
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

	// Broadcast the action to WebSocket clients subscribed to this session
	h.wsHub.BroadcastRecordingAction(sessionID, action)

	h.log.WithFields(map[string]interface{}{
		"session_id":   sessionID,
		"action_type":  action.ActionType,
		"sequence_num": action.SequenceNum,
	}).Debug("Received and broadcast recording action")

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
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

	// Read binary frames from driver and broadcast to browser clients
	for {
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

		// Broadcast binary frame to subscribed browser clients
		// This is a pass-through - no parsing, no re-encoding
		if h.wsHub.HasRecordingSubscribers(sessionID) {
			h.wsHub.BroadcastBinaryFrame(sessionID, data)
		}
	}

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

	// Call playwright-driver to validate selector
	driverURL := fmt.Sprintf("%s/session/%s/record/validate-selector", getPlaywrightDriverURL(), sessionID)

	jsonBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for selector validation")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
		}))
		return
	}

	var driverResp struct {
		Valid      bool   `json:"valid"`
		MatchCount int    `json:"match_count"`
		Selector   string `json:"selector"`
		Error      string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	// Build request for playwright-driver
	driverReq := map[string]interface{}{
		"actions": req.Actions,
	}
	if req.Limit != nil {
		driverReq["limit"] = *req.Limit
	}
	if req.StopOnFailure != nil {
		driverReq["stop_on_failure"] = *req.StopOnFailure
	}
	if req.ActionTimeout != nil {
		driverReq["action_timeout"] = *req.ActionTimeout
	}

	jsonBody, err := json.Marshal(driverReq)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to marshal request",
		}))
		return
	}

	// Call playwright-driver to replay actions
	driverURL := fmt.Sprintf("%s/session/%s/record/replay-preview", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for replay preview")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
		}))
		return
	}

	var replayResp ReplayPreviewResponse
	if err := json.Unmarshal(body, &replayResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	driverURL := fmt.Sprintf("%s/session/%s/record/navigate", getPlaywrightDriverURL(), sessionID)
	body, _ := json.Marshal(map[string]interface{}{
		"url":        req.URL,
		"wait_until": req.WaitUntil,
		"timeout_ms": req.TimeoutMs,
		"capture":    req.Capture,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(body))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for navigate")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(bodyBytes),
		}))
		return
	}

	var driverResp NavigateRecordingResponse
	if err := json.Unmarshal(bodyBytes, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	driverURL := fmt.Sprintf("%s/session/%s/record/screenshot", getPlaywrightDriverURL(), sessionID)

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Failed to read request body: " + err.Error(),
		}))
		return
	}
	if len(bodyBytes) == 0 {
		bodyBytes = []byte("{}")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(bodyBytes))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for screenshot")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(respBody),
		}))
		return
	}

	var driverResp RecordingScreenshotResponse
	if err := json.Unmarshal(respBody, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
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

	driverURL := fmt.Sprintf("%s/session/%s/record/viewport", getPlaywrightDriverURL(), sessionID)
	jsonBody, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(jsonBody))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for viewport update")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
		}))
		return
	}

	var driverResp struct {
		SessionID string `json:"session_id"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if err := json.Unmarshal(body, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
	}

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

	driverURL := fmt.Sprintf("%s/session/%s/record/input", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, driverURL, bytes.NewReader(bodyBytes))
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for record input")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(body),
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

	driverURL := fmt.Sprintf("%s/session/%s/record/frame", getPlaywrightDriverURL(), sessionID)
	if rawQuery := r.URL.RawQuery; rawQuery != "" {
		driverURL += "?" + rawQuery
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, driverURL, nil)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to create request to driver",
		}))
		return
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		h.log.WithError(err).Error("Failed to call playwright-driver for frame")
		h.respondError(w, ErrServiceUnavailable.WithDetails(map[string]string{
			"error": "Playwright driver unavailable: " + err.Error(),
		}))
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":  "Driver returned error",
			"status": fmt.Sprintf("%d", resp.StatusCode),
			"body":   string(bodyBytes),
		}))
		return
	}

	var driverResp RecordingFrameResponse
	if err := json.Unmarshal(bodyBytes, &driverResp); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "Failed to parse driver response",
		}))
		return
	}

	// Generate ETag from content hash provided by playwright-driver.
	// The driver computes MD5 hash of raw JPEG buffer, which is a reliable
	// content fingerprint that changes if and only if the frame content changes.
	// This eliminates false positives (stale frames) and false negatives (unnecessary transfers).
	var etag string
	if driverResp.ContentHash != "" {
		// Use the MD5 hash directly - it's already a reliable content fingerprint
		etag = fmt.Sprintf(`"%s"`, driverResp.ContentHash)
	} else {
		// Fallback for older driver versions without content_hash field
		// Use timestamp only (less reliable but safe)
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

func (h *Handler) resolveSessionProfile(requestedID string) (*recording.SessionProfile, *APIError) {
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

func (h *Handler) fetchSessionStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	driverURL := fmt.Sprintf("%s/session/%s/storage-state", getPlaywrightDriverURL(), sessionID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, driverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build storage state request: %w", err)
	}

	client := &http.Client{Timeout: recordModeTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request storage state: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("driver returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		StorageState json.RawMessage `json:"storage_state"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse storage state: %w", err)
	}
	return payload.StorageState, nil
}

func (h *Handler) persistSessionProfile(ctx context.Context, sessionID string) error {
	if h.sessionProfiles == nil {
		return nil
	}
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID == "" {
		return nil
	}

	state, err := h.fetchSessionStorageState(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(state) == 0 {
		return nil
	}

	_, err = h.sessionProfiles.SaveStorageState(profileID, state)
	return err
}
