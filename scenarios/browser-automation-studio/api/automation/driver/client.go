package driver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"google.golang.org/protobuf/encoding/protojson"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

const (
	// DefaultDriverURL is the default playwright-driver URL for local development.
	DefaultDriverURL = "http://127.0.0.1:39400"

	// DefaultRecordingTimeout is the timeout for recording operations (shorter, user-interactive).
	DefaultRecordingTimeout = 30 * time.Second

	// DefaultExecutionTimeout is the timeout for execution operations (longer, for slow playwright ops).
	DefaultExecutionTimeout = 5 * time.Minute

	// PlaywrightDriverEnv is the environment variable for the driver URL.
	PlaywrightDriverEnv = "PLAYWRIGHT_DRIVER_URL"
)

// HTTPDoer is an interface for making HTTP requests.
// This abstraction enables testing the client without a real HTTP server.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Compile-time interface enforcement
var _ HTTPDoer = (*http.Client)(nil)

// Client provides unified HTTP communication with the playwright-driver.
// It supports both recording mode and execution mode operations.
type Client struct {
	baseURL    string
	httpClient HTTPDoer
	log        *logrus.Logger
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithTimeout sets a custom HTTP timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient = &http.Client{Timeout: timeout}
	}
}

// WithHTTPClient sets a custom HTTP client (for testing).
func WithHTTPClient(client HTTPDoer) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithLogger sets a custom logger.
func WithLogger(log *logrus.Logger) ClientOption {
	return func(c *Client) {
		c.log = log
	}
}

// NewClient creates a unified playwright driver client.
// By default, it uses the execution timeout (5 minutes) for operations.
// Use WithTimeout(DefaultRecordingTimeout) for recording-oriented usage.
func NewClient(opts ...ClientOption) (*Client, error) {
	driverURL := resolveDriverURL(true)
	if strings.TrimSpace(driverURL) == "" {
		return nil, fmt.Errorf("PLAYWRIGHT_DRIVER_URL is required")
	}

	c := &Client{
		baseURL:    strings.TrimRight(driverURL, "/"),
		httpClient: &http.Client{Timeout: DefaultExecutionTimeout},
		log:        logrus.StandardLogger(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// NewClientWithURL creates a client with a specific driver URL (for testing).
func NewClientWithURL(driverURL string, opts ...ClientOption) (*Client, error) {
	if strings.TrimSpace(driverURL) == "" {
		return nil, fmt.Errorf("driver URL is required")
	}

	c := &Client{
		baseURL:    strings.TrimRight(driverURL, "/"),
		httpClient: &http.Client{Timeout: DefaultExecutionTimeout},
		log:        logrus.StandardLogger(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// NewRecordingClient creates a client optimized for recording operations (shorter timeout).
func NewRecordingClient(opts ...ClientOption) (*Client, error) {
	allOpts := append([]ClientOption{WithTimeout(DefaultRecordingTimeout)}, opts...)
	return NewClient(allOpts...)
}

// GetDriverURL returns the configured driver URL (for logging/debugging).
func (c *Client) GetDriverURL() string {
	return c.baseURL
}

// resolveDriverURL resolves the driver URL from environment or default.
func resolveDriverURL(allowDefault bool) string {
	raw := strings.TrimSpace(os.Getenv(PlaywrightDriverEnv))
	if raw != "" {
		return raw
	}
	if allowDefault {
		return DefaultDriverURL
	}
	return ""
}

// Health checks if the driver is reachable.
func (c *Client) Health(ctx context.Context) error {
	if c == nil || c.httpClient == nil {
		return &Error{
			Op:      "health",
			Message: "client not configured",
			Hint:    "ensure NewClient() was called successfully",
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", http.NoBody)
	if err != nil {
		return &Error{
			Op:      "health",
			URL:     c.baseURL,
			Message: "failed to create health check request",
			Cause:   err,
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{
			Op:      "health",
			URL:     c.baseURL,
			Message: "playwright driver is not responding",
			Cause:   err,
			Hint:    "ensure playwright-driver is running (check 'make start' or lifecycle status)",
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return &Error{
			Op:      "health",
			URL:     c.baseURL,
			Status:  resp.StatusCode,
			Message: fmt.Sprintf("unhealthy status: %s", strings.TrimSpace(string(body))),
			Hint:    "check playwright-driver logs for errors",
		}
	}

	return nil
}

// CreateSession creates a new browser session.
// Used by both recording mode (with FrameStreaming) and execution mode (with RequiredCapabilities).
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	if c.log != nil {
		c.log.WithFields(logrus.Fields{
			"execution_id":    req.ExecutionID,
			"viewport_width":  req.Viewport.Width,
			"viewport_height": req.Viewport.Height,
		}).Debug("Creating playwright session")
	}

	var resp CreateSessionResponse
	if err := c.post(ctx, "/session/start", req, &resp); err != nil {
		return nil, err
	}

	if strings.TrimSpace(resp.SessionID) == "" {
		return nil, &Error{
			Op:      "create_session",
			URL:     c.baseURL,
			Message: "driver returned empty session_id",
			Hint:    "driver may be overloaded or encountered an internal error",
		}
	}

	return &resp, nil
}

// CloseSession closes a browser session.
func (c *Client) CloseSession(ctx context.Context, sessionID string) (*CloseSessionResponse, error) {
	var resp CloseSessionResponse
	if err := c.postNoBody(ctx, fmt.Sprintf("/session/%s/close", url.PathEscape(sessionID)), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResetSession resets a session to clean state (for execution reuse).
func (c *Client) ResetSession(ctx context.Context, sessionID string) error {
	return c.postNoBody(ctx, fmt.Sprintf("/session/%s/reset", url.PathEscape(sessionID)), nil)
}

// StartRecording starts recording user actions in a session.
func (c *Client) StartRecording(ctx context.Context, sessionID string, req *StartRecordingRequest) (*StartRecordingResponse, error) {
	var resp StartRecordingResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/start", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopRecording stops recording user actions.
func (c *Client) StopRecording(ctx context.Context, sessionID string) (*StopRecordingResponse, error) {
	var resp StopRecordingResponse
	if err := c.postNoBody(ctx, fmt.Sprintf("/session/%s/record/stop", url.PathEscape(sessionID)), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRecordingStatus gets the current recording status.
func (c *Client) GetRecordingStatus(ctx context.Context, sessionID string) (*RecordingStatusResponse, error) {
	var resp RecordingStatusResponse
	if err := c.get(ctx, fmt.Sprintf("/session/%s/record/status", url.PathEscape(sessionID)), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRecordedActions retrieves all recorded actions for a session.
func (c *Client) GetRecordedActions(ctx context.Context, sessionID string, clear bool) (*GetActionsResponse, error) {
	path := fmt.Sprintf("/session/%s/record/actions", url.PathEscape(sessionID))
	if clear {
		path += "?clear=true"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{
			Op:      "GET " + path,
			URL:     c.baseURL,
			Message: "driver unavailable",
			Cause:   err,
			Hint:    "verify playwright-driver is running and PLAYWRIGHT_DRIVER_URL is correct",
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyStr := strings.TrimSpace(string(body))
		hint := "check playwright-driver logs for details"
		if strings.Contains(bodyStr, "Maximum concurrent sessions") {
			hint = "too many concurrent sessions - wait for other executions to complete or increase session limit"
		} else if strings.Contains(bodyStr, "browser") && strings.Contains(bodyStr, "launch") {
			hint = "browser failed to launch - check chromium installation and system resources"
		}
		return nil, &Error{
			Op:      "GET " + path,
			URL:     c.baseURL,
			Status:  resp.StatusCode,
			Message: bodyStr,
			Hint:    hint,
		}
	}

	if len(body) == 0 {
		return &GetActionsResponse{SessionID: sessionID, Actions: []RecordedAction{}}, nil
	}

	var raw struct {
		SessionID   string            `json:"session_id"`
		IsRecording bool              `json:"is_recording"`
		Actions     []RecordedAction  `json:"actions"`
		Entries     []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	actions := raw.Actions
	if actions == nil {
		actions = []RecordedAction{}
	}

	var entries []*bastimeline.TimelineEntry
	for _, entryRaw := range raw.Entries {
		if len(entryRaw) == 0 {
			continue
		}
		var entry bastimeline.TimelineEntry
		if err := protojson.Unmarshal(entryRaw, &entry); err != nil {
			continue
		}
		entries = append(entries, &entry)
	}

	if len(actions) == 0 && len(entries) > 0 {
		actions = make([]RecordedAction, 0, len(entries))
		for _, entry := range entries {
			actions = append(actions, RecordedActionFromTimelineEntry(entry))
		}
	}

	return &GetActionsResponse{
		SessionID:   raw.SessionID,
		IsRecording: raw.IsRecording,
		Actions:     actions,
		Entries:     raw.Entries,
	}, nil
}

// Navigate navigates the session to a URL (recording mode).
func (c *Client) Navigate(ctx context.Context, sessionID string, req *NavigateRequest) (*NavigateResponse, error) {
	var resp NavigateResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/navigate", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateViewport updates the viewport dimensions.
func (c *Client) UpdateViewport(ctx context.Context, sessionID string, req *UpdateViewportRequest) (*UpdateViewportResponse, error) {
	var resp UpdateViewportResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/viewport", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ValidateSelector validates a selector on the current page.
func (c *Client) ValidateSelector(ctx context.Context, sessionID string, req *ValidateSelectorRequest) (*ValidateSelectorResponse, error) {
	var resp ValidateSelectorResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/validate-selector", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ReplayPreview replays recorded actions for testing.
func (c *Client) ReplayPreview(ctx context.Context, sessionID string, req *ReplayPreviewRequest) (*ReplayPreviewResponse, error) {
	var resp ReplayPreviewResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/replay-preview", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateStreamSettings updates the stream settings for a session.
func (c *Client) UpdateStreamSettings(ctx context.Context, sessionID string, req *UpdateStreamSettingsRequest) (*UpdateStreamSettingsResponse, error) {
	var resp UpdateStreamSettingsResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/stream-settings", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CaptureScreenshot captures a screenshot from the current page.
func (c *Client) CaptureScreenshot(ctx context.Context, sessionID string, req *CaptureScreenshotRequest) (*CaptureScreenshotResponse, error) {
	var resp CaptureScreenshotResponse
	if err := c.post(ctx, fmt.Sprintf("/session/%s/record/screenshot", url.PathEscape(sessionID)), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetFrame retrieves the current frame from the session.
func (c *Client) GetFrame(ctx context.Context, sessionID, queryParams string) (*GetFrameResponse, error) {
	path := fmt.Sprintf("/session/%s/record/frame", url.PathEscape(sessionID))
	if queryParams != "" {
		path += "?" + queryParams
	}
	var resp GetFrameResponse
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ForwardInput forwards pointer/keyboard/wheel events to the driver.
func (c *Client) ForwardInput(ctx context.Context, sessionID string, body []byte) error {
	return c.postRaw(ctx, fmt.Sprintf("/session/%s/record/input", url.PathEscape(sessionID)), body, nil)
}

// GetStorageState retrieves the browser storage state for session persistence.
func (c *Client) GetStorageState(ctx context.Context, sessionID string) (json.RawMessage, error) {
	var resp struct {
		StorageState json.RawMessage `json:"storage_state"`
	}
	if err := c.get(ctx, fmt.Sprintf("/session/%s/storage-state", url.PathEscape(sessionID)), &resp); err != nil {
		return nil, err
	}
	return resp.StorageState, nil
}

// RunInstructions runs simple instructions in a session (e.g., initial navigation).
// This is a convenience method for recording mode that doesn't need full step outcomes.
func (c *Client) RunInstructions(ctx context.Context, sessionID string, instructions []map[string]interface{}) error {
	req := RunInstructionsRequest{Instructions: instructions}
	return c.post(ctx, fmt.Sprintf("/session/%s/run", url.PathEscape(sessionID)), req, nil)
}

// RunInstruction executes a compiled instruction and returns the step outcome.
// This is the primary execution method used by the workflow executor.
func (c *Client) RunInstruction(ctx context.Context, sessionID string, instruction contracts.CompiledInstruction) (contracts.StepOutcome, error) {
	requestBody, err := buildInstructionPayload(instruction)
	if err != nil {
		return contracts.StepOutcome{}, err
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(requestBody); err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("encode run request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/session/%s/run", c.baseURL, url.PathEscape(sessionID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return contracts.StepOutcome{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("run instruction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return contracts.StepOutcome{}, fmt.Errorf("run instruction failed: %s", strings.TrimSpace(string(body)))
	}

	return decodeStepOutcome(resp.Body)
}

func buildInstructionPayload(instruction contracts.CompiledInstruction) (map[string]any, error) {
	payload := make(map[string]any)
	wire := map[string]any{
		"index":   instruction.Index,
		"node_id": instruction.NodeID,
	}

	if instruction.PreloadHTML != "" {
		wire["preload_html"] = instruction.PreloadHTML
	}
	if len(instruction.Context) > 0 {
		context := make(map[string]any, len(instruction.Context))
		for key, value := range instruction.Context {
			context[key] = typeconv.WrapJsonValue(value)
		}
		wire["context"] = context
	}
	if len(instruction.Metadata) > 0 {
		wire["metadata"] = instruction.Metadata
	}

	if instruction.Action != nil {
		actionJSON, err := protojson.MarshalOptions{EmitUnpopulated: false, UseEnumNumbers: true}.Marshal(instruction.Action)
		if err != nil {
			return nil, fmt.Errorf("encode action: %w", err)
		}
		var action map[string]any
		if err := json.Unmarshal(actionJSON, &action); err != nil {
			return nil, fmt.Errorf("decode action: %w", err)
		}
		wire["action"] = action
	}

	stepType := actionTypeToString(instruction.Action)
	if stepType != "" {
		wire["type"] = stepType
	}

	params := actionToParams(instruction.Action)
	if params == nil {
		params = map[string]any{}
	}
	wire["params"] = params

	payload["instruction"] = wire
	return payload, nil
}

func actionTypeToString(action *basactions.ActionDefinition) string {
	if action == nil {
		return ""
	}
	switch action.Type {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "type"
	case basactions.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basactions.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basactions.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basactions.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	case basactions.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case basactions.ActionType_ACTION_TYPE_UPLOAD_FILE:
		return "uploadFile"
	case basactions.ActionType_ACTION_TYPE_DOWNLOAD:
		return "download"
	case basactions.ActionType_ACTION_TYPE_FRAME_SWITCH:
		return "frameSwitch"
	case basactions.ActionType_ACTION_TYPE_TAB_SWITCH:
		return "tabSwitch"
	case basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE:
		return "setCookie"
	case basactions.ActionType_ACTION_TYPE_SHORTCUT:
		return "shortcut"
	case basactions.ActionType_ACTION_TYPE_DRAG_DROP:
		return "dragDrop"
	case basactions.ActionType_ACTION_TYPE_GESTURE:
		return "gesture"
	case basactions.ActionType_ACTION_TYPE_NETWORK_MOCK:
		return "networkMock"
	case basactions.ActionType_ACTION_TYPE_ROTATE:
		return "rotate"
	case basactions.ActionType_ACTION_TYPE_SET_VARIABLE:
		return "setVariable"
	case basactions.ActionType_ACTION_TYPE_LOOP:
		return "loop"
	case basactions.ActionType_ACTION_TYPE_CONDITIONAL:
		return "conditional"
	default:
		return ""
	}
}

func actionToParams(action *basactions.ActionDefinition) map[string]any {
	if action == nil {
		return nil
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(action)
	if err != nil {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	delete(decoded, "type")
	delete(decoded, "metadata")

	for _, value := range decoded {
		if params, ok := value.(map[string]any); ok {
			wrapped := make(map[string]any, len(params))
			for key, param := range params {
				wrapped[key] = typeconv.WrapJsonValue(param)
			}
			return wrapped
		}
	}
	return nil
}

// decodeStepOutcome converts the driver response into a StepOutcome,
// decoding inline base64 screenshot/DOM payloads.
func decodeStepOutcome(r io.Reader) (contracts.StepOutcome, error) {
	var resp StepOutcomeResponse
	if err := json.NewDecoder(r).Decode(&resp); err != nil {
		return contracts.StepOutcome{}, fmt.Errorf("decode driver outcome: %w", err)
	}

	out := resp.StepOutcome
	out.SchemaVersion = contracts.StepOutcomeSchemaVersion
	out.PayloadVersion = contracts.PayloadVersion

	if resp.ScreenshotBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(resp.ScreenshotBase64)
		if err != nil {
			return contracts.StepOutcome{}, fmt.Errorf("decode screenshot: %w", err)
		}
		out.Screenshot = &contracts.Screenshot{
			Data:        data,
			MediaType:   resp.ScreenshotMediaType,
			CaptureTime: time.Now().UTC(),
			Width:       resp.ScreenshotWidth,
			Height:      resp.ScreenshotHeight,
		}
	}

	if resp.DOMHTML != "" {
		out.DOMSnapshot = &contracts.DOMSnapshot{
			HTML:        resp.DOMHTML,
			Preview:     resp.DOMPreview,
			CollectedAt: time.Now().UTC(),
		}
	}

	if resp.VideoPath != "" || resp.TracePath != "" {
		if out.Notes == nil {
			out.Notes = map[string]string{}
		}
		if resp.VideoPath != "" {
			out.Notes["video_path"] = resp.VideoPath
		}
		if resp.TracePath != "" {
			out.Notes["trace_path"] = resp.TracePath
		}
	}

	// Copy element snapshot if provided by driver (enables execution debugging)
	if resp.ElementSnapshot != nil {
		out.ElementSnapshot = resp.ElementSnapshot
	}

	return out, nil
}

// ResolveMaxConcurrentSessions returns the configured max concurrent sessions.
// Falls back to driver default (10) when no override is provided.
func ResolveMaxConcurrentSessions() int {
	const (
		driverDefaultMaxSessions = 10
		maxSessionsCeiling       = 100
		minSessionsFloor         = 1
	)

	envVal := strings.TrimSpace(os.Getenv("MAX_SESSIONS"))
	if envVal != "" {
		if parsed, err := strconv.Atoi(envVal); err == nil && parsed >= minSessionsFloor && parsed <= maxSessionsCeiling {
			return parsed
		}
	}

	return driverDefaultMaxSessions
}

// HTTP helper methods

func (c *Client) get(ctx context.Context, path string, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, response, "GET "+path)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, response interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, response, "POST "+path)
}

func (c *Client) postNoBody(ctx context.Context, path string, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, response, "POST "+path)
}

func (c *Client) postRaw(ctx context.Context, path string, body []byte, response interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doRequest(req, response, "POST "+path)
}

func (c *Client) doRequest(req *http.Request, response interface{}, operation string) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &Error{
			Op:      operation,
			URL:     c.baseURL,
			Message: "driver unavailable",
			Cause:   err,
			Hint:    "verify playwright-driver is running and PLAYWRIGHT_DRIVER_URL is correct",
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyStr := strings.TrimSpace(string(body))
		hint := "check playwright-driver logs for details"
		if strings.Contains(bodyStr, "Maximum concurrent sessions") {
			hint = "too many concurrent sessions - wait for other executions to complete or increase session limit"
		} else if strings.Contains(bodyStr, "browser") && strings.Contains(bodyStr, "launch") {
			hint = "browser failed to launch - check chromium installation and system resources"
		}
		return &Error{
			Op:      operation,
			URL:     c.baseURL,
			Status:  resp.StatusCode,
			Message: bodyStr,
			Hint:    hint,
		}
	}

	if response != nil && len(body) > 0 {
		if err := json.Unmarshal(body, response); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
	}

	return nil
}
