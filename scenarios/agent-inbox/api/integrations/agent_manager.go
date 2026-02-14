// Package integrations provides clients for external services.
// Each integration is isolated behind a clean interface to enable testing
// and potential swapping of implementations.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"agent-inbox/config"
	"agent-inbox/resilience"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// protojson marshal/unmarshal options matching agent-manager's configuration.
// UseProtoNames ensures snake_case keys (matching agent-manager's serialization).
// DiscardUnknown on the consumer side ensures forward-compatibility.
var (
	protoMarshalOpts   = protojson.MarshalOptions{UseProtoNames: true}
	protoUnmarshalOpts = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// AgentManagerClientInterface defines the operations used by agent mode handlers.
// This interface enables dependency injection for testing.
type AgentManagerClientInterface interface {
	StartAgentChat(ctx context.Context, message string, cfg AgentChatConfig) (*AgentChatSession, error)
	ContinueChat(ctx context.Context, runID, message string) error
	GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error)
	GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error)
	StopRun(ctx context.Context, runID string) error
}

// AgentManagerClient provides direct REST API access to agent-manager.
// This client is used for reconciliation during server restart recovery.
// Tool execution flows through the Tool Execution Protocol (ProtocolHandler) instead.
//
// URL Resolution: The client resolves the agent-manager URL lazily and
// re-resolves on connection failures (e.g., after agent-manager restarts
// on a different port). This ensures resilience across service restarts.
type AgentManagerClient struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retryCfg   resilience.RetryConfig
	cb         *resilience.CircuitBreaker
}

// NewAgentManagerClient creates a new agent-manager client.
// The client lazily resolves the agent-manager URL and re-resolves on connection failure.
func NewAgentManagerClient() (*AgentManagerClient, error) {
	cfg := config.Default()
	return NewAgentManagerClientWithConfig(cfg.Integration.AgentManagerTimeout)
}

// NewAgentManagerClientWithConfig creates a new agent-manager client with explicit timeout.
// This enables testing and custom configuration injection.
//
// The initial URL resolution is best-effort: if agent-manager is not yet
// available, the client is still created and will attempt re-resolution
// on the first request.
func NewAgentManagerClientWithConfig(timeout time.Duration) (*AgentManagerClient, error) {
	baseURL, _ := getAgentManagerURL() // best-effort; re-resolved on failure

	cfg := config.Default()
	retryCfg := resilience.RetryConfig{
		MaxAttempts: cfg.Resilience.RetryAttempts,
		BaseDelay:   cfg.Resilience.RetryBaseDelay,
		MaxDelay:    cfg.Resilience.RetryMaxDelay,
		Jitter:      0.1,
	}
	cb := resilience.NewCircuitBreaker(resilience.CircuitBreakerConfig{
		FailureThreshold: cfg.Resilience.CircuitBreakerThreshold,
		Cooldown:         cfg.Resilience.CircuitBreakerCooldown,
	})

	return &AgentManagerClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:  timeout,
		retryCfg: retryCfg,
		cb:       cb,
	}, nil
}

// getAgentManagerURL discovers the agent-manager API URL.
func getAgentManagerURL() (string, error) {
	// Try environment variable first (set by lifecycle system)
	if url := os.Getenv("AGENT_MANAGER_API_URL"); url != "" {
		return url, nil
	}

	// Fall back to port discovery via CLI
	cmd := exec.Command("vrooli", "scenario", "port", "agent-manager", "API_PORT")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("agent-manager not available: %w", err)
	}

	port := strings.TrimSpace(string(output))
	return fmt.Sprintf("http://localhost:%s", port), nil
}

// reResolveURL attempts to re-discover the agent-manager URL.
// Called after a connection failure to handle port drift after restart.
func (c *AgentManagerClient) reResolveURL() error {
	newURL, err := getAgentManagerURL()
	if err != nil {
		return err
	}
	c.baseURL = newURL
	return nil
}

// getBaseURL returns the current base URL, attempting re-resolution if empty.
func (c *AgentManagerClient) getBaseURL() (string, error) {
	if c.baseURL != "" {
		return c.baseURL, nil
	}
	if err := c.reResolveURL(); err != nil {
		return "", fmt.Errorf("agent-manager not available: %w", err)
	}
	return c.baseURL, nil
}

// doWithRetry performs an HTTP request with retry, circuit breaker, and URL re-resolution.
// On retry attempts > 1, it re-resolves the agent-manager URL to handle port drift.
// 4xx responses are marked as permanent (non-retryable) errors.
func (c *AgentManagerClient) doWithRetry(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var resp *http.Response

	err := resilience.Retry(ctx, c.retryCfg, func(ctx context.Context, attempt int) error {
		// Re-resolve URL on retries to handle port drift
		if attempt > 1 {
			_ = c.reResolveURL()
		}

		baseURL, err := c.getBaseURL()
		if err != nil {
			return err
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reqBody)
		if err != nil {
			return resilience.Permanent(err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		doReq := func(ctx context.Context) error {
			var doErr error
			resp, doErr = c.httpClient.Do(req)
			return doErr
		}

		if c.cb != nil {
			err = c.cb.Execute(ctx, doReq)
		} else {
			err = doReq(ctx)
		}
		if err != nil {
			return err
		}

		// Mark 4xx responses as permanent (non-retryable)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return resilience.Permanent(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}

		return nil
	})

	return resp, err
}

// =============================================================================
// Proto Status Helpers
// =============================================================================

// ProtoRunStatusToLocal maps a proto RunStatus enum to the local RunStatus string type.
func ProtoRunStatusToLocal(s domainpb.RunStatus) RunStatus {
	switch s {
	case domainpb.RunStatus_RUN_STATUS_PENDING:
		return RunStatusPending
	case domainpb.RunStatus_RUN_STATUS_STARTING:
		return RunStatusStarting
	case domainpb.RunStatus_RUN_STATUS_RUNNING:
		return RunStatusRunning
	case domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW:
		return RunStatusNeedsReview
	case domainpb.RunStatus_RUN_STATUS_COMPLETE:
		return RunStatusComplete
	case domainpb.RunStatus_RUN_STATUS_FAILED:
		return RunStatusFailed
	case domainpb.RunStatus_RUN_STATUS_CANCELLED:
		return RunStatusCancelled
	default:
		return RunStatus(s.String())
	}
}

// localRunnerTypeToProto maps a local RunnerType string to the proto enum.
func localRunnerTypeToProto(rt RunnerType) domainpb.RunnerType {
	switch rt {
	case RunnerTypeClaudeCode:
		return domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE
	case RunnerTypeCodex:
		return domainpb.RunnerType_RUNNER_TYPE_CODEX
	case RunnerTypeOpenCode:
		return domainpb.RunnerType_RUNNER_TYPE_OPENCODE
	default:
		return domainpb.RunnerType_RUNNER_TYPE_UNSPECIFIED
	}
}

// protoRunPhaseToString maps a proto RunPhase enum to a simple string.
func protoRunPhaseToString(p domainpb.RunPhase) string {
	switch p {
	case domainpb.RunPhase_RUN_PHASE_QUEUED:
		return "queued"
	case domainpb.RunPhase_RUN_PHASE_INITIALIZING:
		return "initializing"
	case domainpb.RunPhase_RUN_PHASE_SANDBOX_CREATING:
		return "sandbox_creating"
	case domainpb.RunPhase_RUN_PHASE_RUNNER_ACQUIRING:
		return "runner_acquiring"
	case domainpb.RunPhase_RUN_PHASE_EXECUTING:
		return "executing"
	case domainpb.RunPhase_RUN_PHASE_COLLECTING_RESULTS:
		return "collecting_results"
	case domainpb.RunPhase_RUN_PHASE_AWAITING_REVIEW:
		return "awaiting_review"
	case domainpb.RunPhase_RUN_PHASE_APPLYING:
		return "applying"
	case domainpb.RunPhase_RUN_PHASE_CLEANING_UP:
		return "cleaning_up"
	case domainpb.RunPhase_RUN_PHASE_COMPLETED:
		return "completed"
	default:
		return ""
	}
}

// protoEventTypeToString maps a proto RunEventType enum to the simple string the UI expects.
func protoEventTypeToString(et domainpb.RunEventType) string {
	switch et {
	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE:
		return "message"
	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL:
		return "tool_call"
	case domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT:
		return "tool_result"
	case domainpb.RunEventType_RUN_EVENT_TYPE_STATUS:
		return "status"
	case domainpb.RunEventType_RUN_EVENT_TYPE_ERROR:
		return "error"
	case domainpb.RunEventType_RUN_EVENT_TYPE_LOG:
		return "log"
	case domainpb.RunEventType_RUN_EVENT_TYPE_METRIC:
		return "metric"
	case domainpb.RunEventType_RUN_EVENT_TYPE_ARTIFACT:
		return "artifact"
	case domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE_DELETED:
		return "message_deleted"
	default:
		return et.String()
	}
}

// CheckAgentStatus gets the status of a coding agent run as a proto Run.
// This is used by the reconciliation service to verify agent state after server restarts.
// On connection failure, re-resolves the agent-manager URL and retries once.
func (c *AgentManagerClient) CheckAgentStatus(ctx context.Context, runID string) (*domainpb.Run, error) {
	resp, err := c.doWithRetry(ctx, "GET", "/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent-manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
	}

	return result.GetRun(), nil
}

// =============================================================================
// Agent Mode Types - For agent-inbox integration
// =============================================================================

// RunnerType identifies which agent runner to use.
type RunnerType string

const (
	RunnerTypeClaudeCode RunnerType = "claude-code"
	RunnerTypeCodex      RunnerType = "codex"
	RunnerTypeOpenCode   RunnerType = "opencode"
)

// AgentChatConfig contains configuration for starting an agent chat.
type AgentChatConfig struct {
	// RunnerType specifies which runner to use (claude-code, codex, opencode)
	RunnerType RunnerType `json:"runner_type"`

	// ProjectPath is the directory where the agent will operate
	ProjectPath string `json:"project_path"`

	// Model optionally specifies the model to use (e.g., "claude-opus-4")
	Model string `json:"model,omitempty"`

	// MaxTurns optionally limits the number of conversation turns
	MaxTurns int `json:"max_turns,omitempty"`
}

// AgentChatSession contains the session info returned after starting agent mode.
type AgentChatSession struct {
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id,omitempty"` // Runner-specific session ID for continuation
}

// RunStatus represents the current state of a run.
type RunStatus string

const (
	RunStatusPending     RunStatus = "pending"
	RunStatusStarting    RunStatus = "starting"
	RunStatusRunning     RunStatus = "running"
	RunStatusNeedsReview RunStatus = "needs_review"
	RunStatusComplete    RunStatus = "complete"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
)

// IsActiveRunStatus returns true if the status indicates the run is still in progress.
func IsActiveRunStatus(status string) bool {
	switch RunStatus(status) {
	case RunStatusPending, RunStatusStarting, RunStatusRunning, RunStatusNeedsReview:
		return true
	default:
		return false
	}
}

// IsTerminalRunStatus returns true if the status indicates the run has finished.
func IsTerminalRunStatus(status string) bool {
	return !IsActiveRunStatus(status) && status != ""
}

// AgentRunStatus contains the status info for a run.
type AgentRunStatus struct {
	RunID           string    `json:"run_id"`
	Status          RunStatus `json:"status"`
	Phase           string    `json:"phase"`
	ProgressPercent int       `json:"progress_percent"`
	SessionID       string    `json:"session_id,omitempty"`
	ErrorMsg        string    `json:"error_msg,omitempty"`
}

// TranslatedEvent represents an agent-manager event translated for inbox rendering.
type TranslatedEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // message, tool_call, tool_result, status, error, log, metric, artifact, message_deleted, etc.
	Role      string    `json:"role"` // user, assistant, system, tool
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int64     `json:"sequence"`

	// Tool fields (for tool_call and tool_result types)
	ToolName    string `json:"tool_name,omitempty"`
	ToolCallID  string `json:"tool_call_id,omitempty"` // Correlation ID linking tool_call ↔ tool_result
	ToolInput   string `json:"tool_input,omitempty"`   // JSON string of tool input
	ToolOutput  string `json:"tool_output,omitempty"`  // Tool result output
	ToolSuccess bool   `json:"tool_success,omitempty"`

	// Status fields (for status type)
	RunStatus string `json:"run_status,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Progress  int    `json:"progress,omitempty"`

	// RawData holds the original event data as JSON for event types that
	// don't have dedicated fields. The UI can display this in a generic card.
	RawData string `json:"raw_data,omitempty"`
}

// =============================================================================
// Agent Mode Methods
// =============================================================================

// StartAgentChat creates a task and run for agent mode chat.
// This is the entry point for starting an agentic coding session.
// On connection failure, re-resolves the agent-manager URL and retries once.
func (c *AgentManagerClient) StartAgentChat(ctx context.Context, message string, cfg AgentChatConfig) (*AgentChatSession, error) {
	// First create a task using proto types
	taskReq := &apipb.CreateTaskRequest{
		Task: &domainpb.Task{
			Title:       "Agent Chat Session",
			Description: message,
			ScopePath:   cfg.ProjectPath,
			ProjectRoot: cfg.ProjectPath,
		},
	}
	taskBody, err := protoMarshalOpts.Marshal(taskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/tasks", taskBody)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create task: %d: %s", resp.StatusCode, string(respBody))
	}

	var taskResult apipb.CreateTaskResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &taskResult); err != nil {
		return nil, fmt.Errorf("failed to parse task response: %w", err)
	}

	task := taskResult.GetTask()
	if task == nil {
		return nil, fmt.Errorf("task response missing task")
	}
	taskID := task.GetId()
	if taskID == "" {
		return nil, fmt.Errorf("task response missing id")
	}

	// Now create a run for this task using proto types
	runMode := domainpb.RunMode_RUN_MODE_IN_PLACE
	profileKey := "agent-inbox-" + string(cfg.RunnerType)
	runReq := &apipb.CreateRunRequest{
		TaskId:  taskID,
		RunMode: &runMode,
		ProfileRef: &apipb.ProfileRef{
			ProfileKey: profileKey,
			Defaults: &domainpb.AgentProfile{
				ProfileKey: profileKey,
				Name:       profileKey,
				RunnerType: localRunnerTypeToProto(cfg.RunnerType),
				Model:      cfg.Model,
				MaxTurns:   int32(cfg.MaxTurns),
			},
		},
	}
	runBody, err := protoMarshalOpts.Marshal(runReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run request: %w", err)
	}

	resp2, err := c.doWithRetry(ctx, "POST", "/api/v1/runs", runBody)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp2.Body.Close()

	respBody, _ = io.ReadAll(resp2.Body)
	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create run: %d: %s", resp2.StatusCode, string(respBody))
	}

	var runResult apipb.CreateRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &runResult); err != nil {
		return nil, fmt.Errorf("failed to parse run response: %w", err)
	}

	run := runResult.GetRun()
	if run == nil {
		return nil, fmt.Errorf("run response missing run")
	}

	return &AgentChatSession{
		TaskID:    taskID,
		RunID:     run.GetId(),
		SessionID: run.GetSessionId(),
	}, nil
}

// ContinueChat sends a follow-up message to an existing agent run.
// Uses the continuation API to maintain conversation state.
func (c *AgentManagerClient) ContinueChat(ctx context.Context, runID, message string) error {
	reqProto := &domainpb.ContinueRunRequest{
		RunId:   runID,
		Message: message,
	}
	body, err := protoMarshalOpts.Marshal(reqProto)
	if err != nil {
		return fmt.Errorf("failed to marshal continue request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/runs/"+runID+"/continue", body)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("continue failed: %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetEvents retrieves events for a run, optionally filtered by sequence.
func (c *AgentManagerClient) GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error) {
	path := fmt.Sprintf("/api/v1/runs/%s/events?after_sequence=%d", runID, afterSequence)

	resp, err := c.doWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get events failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunEventsResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse events response: %w", err)
	}

	// Translate proto events to inbox format
	events := make([]*TranslatedEvent, 0, len(result.GetEvents()))
	for _, protoEvent := range result.GetEvents() {
		event := TranslateProtoEvent(protoEvent)
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// TranslateProtoEvent converts a proto RunEvent to TranslatedEvent format.
func TranslateProtoEvent(ev *domainpb.RunEvent) *TranslatedEvent {
	if ev == nil {
		return nil
	}

	event := &TranslatedEvent{
		ID:       ev.GetId(),
		Sequence: ev.GetSequence(),
		Type:     protoEventTypeToString(ev.GetEventType()),
	}

	if ts := ev.GetTimestamp(); ts != nil {
		event.Timestamp = ts.AsTime()
	}

	switch d := ev.Data.(type) {
	case *domainpb.RunEvent_Message:
		msg := d.Message
		event.Role = msg.GetRole()
		event.Content = msg.GetContent()

	case *domainpb.RunEvent_ToolCall:
		tc := d.ToolCall
		event.Role = "assistant"
		event.ToolName = tc.GetToolName()
		event.ToolCallID = tc.GetToolCallId()
		if input := tc.GetInput(); input != nil {
			inputBytes, _ := json.Marshal(input.AsMap())
			event.ToolInput = string(inputBytes)
		}

	case *domainpb.RunEvent_ToolResult:
		tr := d.ToolResult
		event.Role = "tool"
		event.ToolName = tr.GetToolName()
		event.ToolCallID = tr.GetToolCallId()
		event.ToolOutput = tr.GetOutput()
		event.ToolSuccess = tr.GetSuccess()
		if errMsg := tr.GetError(); errMsg != "" {
			event.ToolOutput = errMsg
			event.ToolSuccess = false
		}

	case *domainpb.RunEvent_Status:
		st := d.Status
		event.Role = "system"
		event.RunStatus = st.GetNewStatus()
		event.Content = st.GetReason()

	case *domainpb.RunEvent_Error:
		errData := d.Error
		event.Role = "system"
		event.Content = errData.GetMessage()

	case *domainpb.RunEvent_Log:
		logData := d.Log
		event.Role = "system"
		event.Content = logData.GetMessage()
		rawBytes, _ := protoMarshalOpts.Marshal(logData)
		event.RawData = string(rawBytes)

	case *domainpb.RunEvent_Metric:
		metricData := d.Metric
		event.Role = "system"
		event.Content = metricData.GetName()
		rawBytes, _ := protoMarshalOpts.Marshal(metricData)
		event.RawData = string(rawBytes)

	case *domainpb.RunEvent_Artifact:
		artData := d.Artifact
		event.Role = "system"
		event.Content = artData.GetType()
		rawBytes, _ := protoMarshalOpts.Marshal(artData)
		event.RawData = string(rawBytes)

	case *domainpb.RunEvent_MessageDeleted:
		mdData := d.MessageDeleted
		event.Role = "system"
		event.Content = mdData.GetTargetEventId()
		rawBytes, _ := protoMarshalOpts.Marshal(mdData)
		event.RawData = string(rawBytes)

	default:
		// Unknown/unhandled oneof variant (progress, cost, rate_limit, etc.)
		event.Role = "system"
	}

	return event
}

// StopRun stops a running agent run.
func (c *AgentManagerClient) StopRun(ctx context.Context, runID string) error {
	resp, err := c.doWithRetry(ctx, "POST", "/api/v1/runs/"+runID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop failed: %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetRunStatus gets the current status of a run.
func (c *AgentManagerClient) GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error) {
	resp, err := c.doWithRetry(ctx, "GET", "/api/v1/runs/"+runID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to agent-manager: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get status failed: %d: %s", resp.StatusCode, string(respBody))
	}

	var result apipb.GetRunResponse
	if err := protoUnmarshalOpts.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse status response: %w", err)
	}

	run := result.GetRun()
	if run == nil {
		return nil, fmt.Errorf("run response missing run")
	}

	return &AgentRunStatus{
		RunID:           runID,
		Status:          ProtoRunStatusToLocal(run.GetStatus()),
		Phase:           protoRunPhaseToString(run.GetPhase()),
		ProgressPercent: int(run.GetProgressPercent()),
		SessionID:       run.GetSessionId(),
		ErrorMsg:        run.GetErrorMsg(),
	}, nil
}

// GetWebSocketURL returns the WebSocket URL for connecting to agent-manager events.
// Re-resolves the URL if not yet available.
func (c *AgentManagerClient) GetWebSocketURL() string {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return "" // Caller should handle empty URL
	}
	// Convert http:// to ws://
	wsURL := strings.Replace(baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	return wsURL + "/api/v1/ws"
}
