package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

const (
	recordModeTimeout     = 30 * time.Second
	playwrightDriverEnv   = "PLAYWRIGHT_DRIVER_URL"
	defaultPlaywrightURL  = "http://127.0.0.1:39400"
)

func getPlaywrightDriverURL() string {
	url := os.Getenv(playwrightDriverEnv)
	if url == "" {
		return defaultPlaywrightURL
	}
	return url
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

	reqBody := map[string]string{}
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

	// Convert actions to workflow nodes
	flowDefinition := convertActionsToWorkflow(actions)

	// Create the workflow using existing service
	workflow, err := h.workflowService.CreateWorkflowWithProject(
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

	h.respondSuccess(w, http.StatusCreated, GenerateWorkflowResponse{
		WorkflowID:  workflow.ID,
		ProjectID:   projectID,
		Name:        workflow.Name,
		NodeCount:   len(flowDefinition["nodes"].([]map[string]interface{})),
		ActionCount: len(actions),
	})
}

// convertActionsToWorkflow converts recorded actions to a workflow flow definition.
func convertActionsToWorkflow(actions []RecordedAction) map[string]interface{} {
	nodes := make([]map[string]interface{}, 0, len(actions))
	edges := make([]map[string]interface{}, 0, len(actions)-1)

	var prevNodeID string

	for i, action := range actions {
		nodeID := fmt.Sprintf("node_%d", i+1)

		node := mapActionToNode(action, nodeID, i)
		nodes = append(nodes, node)

		// Chain nodes sequentially
		if prevNodeID != "" {
			edges = append(edges, map[string]interface{}{
				"id":     fmt.Sprintf("edge_%d", i),
				"source": prevNodeID,
				"target": nodeID,
			})
		}

		prevNodeID = nodeID
	}

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

	switch action.ActionType {
	case "click":
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		if action.Payload != nil {
			if btn, ok := action.Payload["button"]; ok {
				data["button"] = btn
			}
			if mods, ok := action.Payload["modifiers"]; ok {
				data["modifiers"] = mods
			}
		}
		data["label"] = generateClickLabel(action)

	case "type":
		nodeType = "type"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		if action.Payload != nil {
			if text, ok := action.Payload["text"].(string); ok {
				data["text"] = text
				data["label"] = fmt.Sprintf("Type: %q", truncateString(text, 20))
			}
		}

	case "navigate":
		nodeType = "navigate"
		data["url"] = action.URL
		data["label"] = fmt.Sprintf("Navigate to %s", extractHostname(action.URL))

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

	case "focus":
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		data["label"] = "Focus element"

	case "keypress":
		nodeType = "keyboard"
		if action.Payload != nil {
			if key, ok := action.Payload["key"].(string); ok {
				data["key"] = key
				data["label"] = fmt.Sprintf("Press %s", key)
			}
		}

	default:
		// Default to click for unknown types
		nodeType = "click"
		if action.Selector != nil {
			data["selector"] = action.Selector.Primary
		}
		data["label"] = action.ActionType
	}

	node["type"] = nodeType
	node["data"] = data

	return node
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

	h.respondSuccess(w, http.StatusOK, driverResp)
}
