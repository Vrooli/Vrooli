package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// DurationToProtojson converts a Go time.Duration to the protojson string
// format for google.protobuf.Duration (e.g. "600s"). Returns "" for zero.
func DurationToProtojson(d time.Duration) string {
	if d == 0 {
		return ""
	}
	s := d.Seconds()
	if s == math.Trunc(s) {
		return fmt.Sprintf("%ds", int64(s))
	}
	return fmt.Sprintf("%gs", s)
}

// AgentManagerClient provides HTTP client for agent-manager API
type AgentManagerClient struct {
	httpClient  *http.Client
	testBaseURL string // override for tests; empty in production
}

// NewAgentManagerClient creates a new agent-manager client
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ProfileRef identifies a profile by key and optional defaults.
type ProfileRef struct {
	ProfileKey string        `json:"profile_key"`
	Defaults   *AgentProfile `json:"defaults,omitempty"`
}

// AgentProfile defines the configuration for running an agent.
// JSON tags use snake_case to match agent-manager's protojson schema.
type AgentProfile struct {
	Name                 string   `json:"name"`
	ProfileKey           string   `json:"profile_key"`
	Description          string   `json:"description,omitempty"`
	RunnerType           string   `json:"runner_type"`
	Model                string   `json:"model,omitempty"`
	ModelPreset          string   `json:"model_preset,omitempty"`
	MaxTurns             int32    `json:"max_turns,omitempty"`
	Timeout              string   `json:"timeout,omitempty"` // protojson Duration format, e.g. "600s"
	AllowedTools         []string `json:"allowed_tools,omitempty"`
	SkipPermissionPrompt bool     `json:"skip_permission_prompt,omitempty"`
	RequiresSandbox      bool     `json:"requires_sandbox,omitempty"`
	RequiresApproval     bool     `json:"requires_approval,omitempty"`
	CreatedBy            string   `json:"created_by,omitempty"`
}

// Task represents a task for agent execution.
// Field names and JSON tags must match the agent-manager proto schema
// (protojson with DiscardUnknown=false rejects any unrecognised field).
type Task struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title"`                  // Short label (required, 1-255 chars)
	Description string `json:"description"`            // Main prompt sent to the agent
	ScopePath   string `json:"scope_path"`             // Working directory (required, non-empty)
	ProjectRoot string `json:"project_root,omitempty"` // Optional project root
}

// CreateTaskRequest is the request for creating a task
type CreateTaskRequest struct {
	Task *Task `json:"task"`
}

// CreateTaskResponse is the response from creating a task
type CreateTaskResponse struct {
	Task *Task `json:"task"`
}

// CreateRunRequest is the request for creating a run.
// JSON tags use snake_case to match agent-manager's protojson schema.
type CreateRunRequest struct {
	TaskID     string      `json:"task_id"`
	ProfileRef *ProfileRef `json:"profile_ref,omitempty"`
	Tag        *string     `json:"tag,omitempty"`
	RunMode    string      `json:"run_mode,omitempty"`
}

// Run represents an agent run.
// JSON tags use snake_case to match agent-manager's protojson UseProtoNames output.
type Run struct {
	ID        string      `json:"id"`
	TaskID    string      `json:"task_id"`
	ProfileID string      `json:"agent_profile_id,omitempty"`
	Status    string      `json:"status"`
	StartedAt string      `json:"started_at,omitempty"`
	EndedAt   string      `json:"ended_at,omitempty"`
	Error     string      `json:"error_msg,omitempty"`
	Tag       string      `json:"tag,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Actions   *RunActions `json:"actions,omitempty"`
}

// RunActions describes which actions are available for a run.
type RunActions struct {
	CanInvestigate        bool `json:"can_investigate"`
	CanApplyInvestigation bool `json:"can_apply_investigation"`
	CanDelete             bool `json:"can_delete"`
	CanStop               bool `json:"can_stop"`
	CanRetry              bool `json:"can_retry"`
	CanContinue           bool `json:"can_continue"`
}

// ListRunsOptions configures filtering for ListRuns.
type ListRunsOptions struct {
	Status                    string
	TagPrefix                 string
	ProfileKey                string
	TaskID                    string
	InvestigatesRunID         string
	AppliesInvestigationRunID string
	Limit                     int
	Offset                    int
}

// ListRunsResponse is the response from listing runs.
type ListRunsResponse struct {
	Runs    []*Run `json:"runs"`
	Total   int    `json:"total"`
	HasMore bool   `json:"has_more"`
}

// ContinueRunRequest is the request for continuing a paused run.
type ContinueRunRequest struct {
	Message string `json:"message"`
}

// InvestigateRunRequest is the request for creating an investigation run.
type InvestigateRunRequest struct {
	RunIDs        []string `json:"runIds"`
	Depth         string   `json:"depth"`
	CustomContext string   `json:"customContext"`
}

// InvestigationApplyRequest is the request for applying an investigation.
type InvestigationApplyRequest struct {
	InvestigationRunID string `json:"investigationRunId"`
	CustomContext      string `json:"customContext"`
}

// CreateRunResponse is the response from creating a run
type CreateRunResponse struct {
	Run *Run `json:"run"`
}

// GetRunResponse is the response from getting a run
type GetRunResponse struct {
	Run *Run `json:"run"`
}

// EnsureProfileRequest requests a profile by key.
type EnsureProfileRequest struct {
	ProfileKey     string        `json:"profile_key"`
	Defaults       *AgentProfile `json:"defaults,omitempty"`
	UpdateExisting bool          `json:"update_existing,omitempty"`
}

// EnsureProfileResponse is the response from ensure profile
type EnsureProfileResponse struct {
	Profile *AgentProfile `json:"profile"`
	Created bool          `json:"created"`
	Updated bool          `json:"updated"`
}

// Health checks the agent-manager service health
func (c *AgentManagerClient) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed
func (c *AgentManagerClient) EnsureProfile(ctx context.Context, req *EnsureProfileRequest) (*EnsureProfileResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/profiles/ensure", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result EnsureProfileResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTask creates a new task
func (c *AgentManagerClient) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	req := CreateTaskRequest{Task: task}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/tasks", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Task, nil
}

// CreateRun starts a new run for a task
func (c *AgentManagerClient) CreateRun(ctx context.Context, req *CreateRunRequest) (*Run, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/runs", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// GetRun retrieves a run by ID
func (c *AgentManagerClient) GetRun(ctx context.Context, runID string) (*Run, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", fmt.Sprintf("/api/v1/runs/%s", runID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result GetRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// WaitForRun polls until a run reaches a terminal state
func (c *AgentManagerClient) WaitForRun(ctx context.Context, runID string, pollInterval time.Duration) (*Run, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			run, err := c.GetRun(ctx, runID)
			if err != nil {
				return nil, err
			}
			if run == nil {
				return nil, fmt.Errorf("run %s not found", runID)
			}

			if IsTerminalStatus(run.Status) {
				return run, nil
			}
		}
	}
}

// StopRun requests agent-manager to stop a running run.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) error {
	resp, err := c.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/runs/%s/stop", runID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return c.parseError(resp)
	}
	return nil
}

// GetRunEvents fetches raw event JSON for a run from agent-manager.
// afterSequence <= -1 and limit <= 0 are treated as "not set" and omitted
// from the forwarded query string so that agent-manager applies its own defaults.
func (c *AgentManagerClient) GetRunEvents(ctx context.Context, runID string, afterSequence int64, limit int) ([]byte, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events", runID)
	var params []string
	if afterSequence >= 0 {
		params = append(params, fmt.Sprintf("after_sequence=%d", afterSequence))
	}
	if limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", limit))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	resp, err := c.doRequestWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("run %s not found", runID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	return io.ReadAll(resp.Body)
}

// ListRuns retrieves runs with optional filtering.
func (c *AgentManagerClient) ListRuns(ctx context.Context, opts ListRunsOptions) (*ListRunsResponse, error) {
	path := "/api/v1/runs"
	var params []string
	if opts.Status != "" {
		params = append(params, "status="+opts.Status)
	}
	if opts.TagPrefix != "" {
		params = append(params, "tag_prefix="+opts.TagPrefix)
	}
	if opts.ProfileKey != "" {
		params = append(params, "profile_key="+opts.ProfileKey)
	}
	if opts.TaskID != "" {
		params = append(params, "task_id="+opts.TaskID)
	}
	if opts.InvestigatesRunID != "" {
		params = append(params, "investigates_run_id="+opts.InvestigatesRunID)
	}
	if opts.AppliesInvestigationRunID != "" {
		params = append(params, "applies_investigation_run_id="+opts.AppliesInvestigationRunID)
	}
	if opts.Limit > 0 {
		params = append(params, fmt.Sprintf("limit=%d", opts.Limit))
	}
	if opts.Offset > 0 {
		params = append(params, fmt.Sprintf("offset=%d", opts.Offset))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := c.doRequestWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result ListRunsResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ContinueRun sends a continue message to a paused run.
func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, message string) (*Run, error) {
	body, err := json.Marshal(ContinueRunRequest{Message: message})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", fmt.Sprintf("/api/v1/runs/%s/continue", runID), body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// CreateInvestigationRun creates an investigation run for the given run IDs.
func (c *AgentManagerClient) CreateInvestigationRun(ctx context.Context, runIDs []string, depth string, customContext string) (*Run, error) {
	body, err := json.Marshal(InvestigateRunRequest{
		RunIDs:        runIDs,
		Depth:         depth,
		CustomContext: customContext,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/runs/investigate", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// CreateInvestigationApplyRun creates a run that applies an investigation's recommendations.
func (c *AgentManagerClient) CreateInvestigationApplyRun(ctx context.Context, investigationRunID string, customContext string) (*Run, error) {
	body, err := json.Marshal(InvestigationApplyRequest{
		InvestigationRunID: investigationRunID,
		CustomContext:      customContext,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/runs/investigation-apply", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Run, nil
}

// isRetryable returns true for transport errors and 5xx status codes.
func isRetryable(resp *http.Response, err error) bool {
	if err != nil {
		return true // connection refused, timeout, DNS, etc.
	}
	return resp.StatusCode >= 500
}

// doRequestWithRetry wraps doRequest with bounded retry for transient failures.
// Max 3 attempts with backoff: 500ms, 1s, 2s. Only retries transport errors
// and 5xx responses. The URL is re-resolved on each attempt.
func (c *AgentManagerClient) doRequestWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	backoff := []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	var lastErr error

	for attempt := 0; attempt <= len(backoff); attempt++ {
		resp, err := c.doRequest(ctx, method, path, body)
		if !isRetryable(resp, err) {
			return resp, err
		}

		// Close body from failed attempt to avoid leaking connections
		if resp != nil {
			resp.Body.Close()
		}
		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("agent-manager returned %d", resp.StatusCode)
		}

		// Don't sleep after the last attempt
		if attempt < len(backoff) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff[attempt]):
			}
		}
	}
	return nil, fmt.Errorf("agent-manager unreachable after %d attempts: %w", len(backoff)+1, lastErr)
}

// doRequest performs an HTTP request to agent-manager
func (c *AgentManagerClient) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *AgentManagerClient) resolveBaseURL(ctx context.Context) (string, error) {
	if c.testBaseURL != "" {
		return c.testBaseURL, nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("resolve agent-manager url: %w", err)
	}
	return url, nil
}

func (c *AgentManagerClient) parseResponse(resp *http.Response, result interface{}) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	return nil
}

func (c *AgentManagerClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("agent-manager error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("agent-manager error: status %d, body: %s", resp.StatusCode, string(body))
}
