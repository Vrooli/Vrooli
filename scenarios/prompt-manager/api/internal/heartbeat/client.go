package heartbeat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/owneridentity"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

// AgentManagerClient provides HTTP client for agent-manager API
type AgentManagerClient struct {
	createCallerVerifier owneridentity.Validator
	httpClient           *http.Client
	testBaseURL          string // override for tests; empty in production
	sleep                func(context.Context, time.Duration) error
	// serviceToken mints Prompt Manager's own service-principal bearer for an
	// unattended heartbeat. Its authority mode may be retained in an in-memory
	// queue, but the credential is request-local and refreshed before owner I/O.
	serviceToken interface {
		Token(context.Context) (string, error)
	}
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// NewAgentManagerClient creates a new agent-manager client
func NewAgentManagerClient(timeout time.Duration) *AgentManagerClient {
	return &AgentManagerClient{
		createCallerVerifier: owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})}),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		serviceToken: defaultServiceTokenSource(),
	}
}

// ProfileRef identifies a reconciled scenario-owned profile by key.
type ProfileRef struct {
	ProfileKey string `json:"profile_key"`
}

// AgentProfile retains the complete owner schema, including execution controls and provenance.
type AgentProfile = domainpb.AgentProfile

// Task represents a task for agent execution.
// Field names and JSON tags must match the agent-manager proto schema
// (protojson with DiscardUnknown=false rejects any unrecognised field).
type Task struct {
	ID                 string          `json:"id,omitempty"`
	Title              string          `json:"title"`                         // Short label (required, 1-255 chars)
	Description        string          `json:"description"`                   // Main prompt sent to the agent
	ScopePath          string          `json:"scope_path"`                    // Working directory (required, non-empty)
	ProjectRoot        string          `json:"project_root,omitempty"`        // Optional project root
	PhasePromptIDs     []string        `json:"phase_prompt_ids,omitempty"`    // Existing phase prompts are part of the owner task identity.
	Status             string          `json:"status,omitempty"`              // Owner lifecycle status, when returned
	CurrentRunID       string          `json:"current_run_id,omitempty"`      // Owner attempt identity; never inferred from task status.
	ContextAttachments json.RawMessage `json:"context_attachments,omitempty"` // Retained so finite transport can refuse unsupported attached tasks.
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
//
// Environment is a per-request map of `VROOLI_*`-prefixed env vars that
// agent-manager merges into the spawned agent process's environment via
// phases.MergeEnvVars. Agent-manager validates the prefix and a 4096-byte
// total-size cap; see scenarios/agent-manager/api/internal/handlers/handlers.go
// validateCustomEnvironment. Heartbeat callers use this to propagate
// VROOLI_PROMPT_MANAGER_ATTRIBUTION (see spawn_attribution.go); the
// channel is generic and other prompt-manager spawn sites may add additional
// VROOLI_-prefixed vars without a contract change.
type CreateRunRequest struct {
	ParentRunID     *string                  `json:"parent_run_id,omitempty"`
	RequestedScopes *RunIdentityScopeRequest `json:"requested_scopes,omitempty"`
	WorkReferences  []*eventpb.WorkReference `json:"work_references,omitempty"`
	IdempotencyKey  string                   `json:"idempotency_key,omitempty"`
	TaskID          string                   `json:"task_id"`
	ProfileRef      *ProfileRef              `json:"profile_ref,omitempty"`
	Tag             *string                  `json:"tag,omitempty"`
	RunMode         string                   `json:"run_mode,omitempty"`
	Environment     map[string]string        `json:"environment,omitempty"`
}

// RunIdentityScopeRequest is a narrowing declaration, never a grant.
type RunIdentityScopeRequest struct {
	Scopes               []string `json:"scopes"`
	ExpectedOwnerSubject string   `json:"expected_owner_subject,omitempty"`
}

// Run represents an agent run.
// JSON tags use snake_case to match agent-manager's protojson UseProtoNames output.
type Run struct {
	ParentRunID           string `json:"parent_run_id,omitempty"`
	RequestedModel        string `json:"requested_model,omitempty"`
	ActualModel           string `json:"actual_model,omitempty"`
	HarnessKind           string `json:"harness_kind,omitempty"`
	ImportSourceHarness   string `json:"import_source_harness,omitempty"`
	ImportSourceSessionID string `json:"import_source_session_id,omitempty"`
	CreatedAt             string `json:"created_at,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
	TerminalClass         string `json:"terminal_class,omitempty"`
	StopReason            string `json:"stop_reason,omitempty"`
	// Keep owner JSON presence and open reference enums. A summary is reported
	// evidence, not a qualified per-attempt usage or acceptance receipt.
	Summary        json.RawMessage `json:"summary,omitempty"`
	WorkReferences json.RawMessage `json:"work_references,omitempty"`
	ID             string          `json:"id"`
	TaskID         string          `json:"task_id"`
	ProfileID      string          `json:"agent_profile_id,omitempty"`
	Status         string          `json:"status"`
	StartedAt      string          `json:"started_at,omitempty"`
	EndedAt        string          `json:"ended_at,omitempty"`
	Error          string          `json:"error_msg,omitempty"`
	Tag            string          `json:"tag,omitempty"`
	SessionID      string          `json:"session_id,omitempty"`
	Actions        *RunActions     `json:"actions,omitempty"`
	Result         *RunResult      `json:"result,omitempty"`
}

// RunResult is the provenance-bearing final-output projection returned by
// agent-manager. Heartbeats consume the selected candidate directly instead
// of reconstructing a result from transcript position.
type RunResult struct {
	FinalOutput string               `json:"final_output,omitempty"`
	Selection   RunResultSelection   `json:"selection,omitempty"`
	Candidates  []RunResultCandidate `json:"candidates,omitempty"`
}

type RunResultSelection struct {
	Status              string `json:"status,omitempty"`
	SelectedCandidateID string `json:"selected_candidate_id,omitempty"`
	Rule                string `json:"rule,omitempty"`
}

type RunResultCandidate struct {
	ID      string `json:"id,omitempty"`
	Content string `json:"content,omitempty"`
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
	AgentProfileID            string
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

// ContinueRunRequest is the request for continuing a paused run. A stable
// IdempotencyKey lets the owner (agent-manager) deduplicate a retried send or a
// reload that replays the same accepted turn; the caller owns generation and
// must not reuse one key for a different message.
type ContinueRunRequest struct {
	Message        string `json:"message"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
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

// Health checks the agent-manager service health
func (c *AgentManagerClient) Health(ctx context.Context) (bool, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", "/health", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// ResolveProfile reads the current unpaginated owner list. A positive exact
// identity does not require catalog completeness; absence or ambiguity refuses
// the affected consumer instead of silently dropping its filter or installing.
func (c *AgentManagerClient) ResolveProfile(ctx context.Context, key string) (*AgentProfile, error) {
	if key == "" || strings.TrimSpace(key) != key {
		return nil, fmt.Errorf("profile key must be exact and nonempty")
	}
	resp, err := c.doRequestWithRetry(ctx, http.MethodGet, "/api/v1/profiles", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read profile list: %w", err)
	}
	var result apipb.ListProfilesResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unsupported profile list: %w", err)
	}
	var match *AgentProfile
	for _, profile := range result.Profiles {
		if profile.GetProfileKey() != key {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("profile key %q is ambiguous", key)
		}
		id, err := uuid.Parse(profile.Id)
		if err != nil || id == uuid.Nil {
			return nil, fmt.Errorf("profile key %q has invalid owner identity", key)
		}
		match = profile
	}
	if match == nil {
		return nil, fmt.Errorf("profile key %q is not installed", key)
	}
	return match, nil
}

// CreateTask submits once. Possible owner effects survive a lost response;
// only provably unsent requests and definitive refusals release the obligation.
func (c *AgentManagerClient) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, NewDispatchRejectedError(err)
	}
	body, err := json.Marshal(CreateTaskRequest{Task: task})
	if err != nil {
		return nil, NewDispatchRejectedError(fmt.Errorf("marshal request: %w", err))
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/tasks", body)
	if err != nil {
		if IsDispatchRejected(err) {
			return nil, err
		}
		return nil, NewDispatchUncertainError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusAccepted {
			if cause, ok := approvalPendingResponse(resp); ok {
				return nil, NewApprovalPendingError("creating_task", cause)
			}
			return nil, NewDispatchUncertainError(fmt.Errorf("task creation returned unrecognized 202 response"))
		}
		cause := c.parseError(resp)
		if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < http.StatusInternalServerError {
			return nil, NewDispatchRejectedError(cause)
		}
		return nil, NewDispatchUncertainError(cause)
	}
	var result CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, NewDispatchUncertainError(err)
	}
	if result.Task == nil {
		return nil, NewDispatchUncertainError(fmt.Errorf("task creation returned no task identity"))
	}
	id, identityErr := uuid.Parse(result.Task.ID)
	if identityErr != nil || id == uuid.Nil || strings.TrimSpace(result.Task.ID) != result.Task.ID || (task != nil && task.ID != "" && result.Task.ID != task.ID) {
		return nil, NewDispatchUncertainError(fmt.Errorf("task creation returned invalid or mismatched task identity"))
	}
	return result.Task, nil
}

// GetTask returns nil only when the task does not exist. Conversation retries
// reuse the authoritative task after an ambiguous create response.
func (c *AgentManagerClient) GetTask(ctx context.Context, taskID string) (*Task, error) {
	resp, err := c.doRequestWithRetry(ctx, "GET", "/api/v1/tasks/"+taskID, nil)
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
	var result CreateTaskResponse
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, NewDispatchUncertainError(fmt.Errorf("read task after an ambiguous create: %w", err))
	}
	if result.Task == nil || strings.TrimSpace(result.Task.ID) == "" || result.Task.ID != taskID {
		return nil, NewDispatchUncertainError(fmt.Errorf("task lookup returned missing or mismatched identity for %q", taskID))
	}
	return result.Task, nil
}

// CancelTask transitions a queued or running task to cancelled so it can be
// removed by the compensating cleanup after a failed run creation.
func (c *AgentManagerClient) CancelTask(ctx context.Context, taskID string) error {
	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/tasks/"+taskID+"/cancel", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return c.parseError(resp)
	}
	return nil
}

// DeleteTask removes a cancelled task that could not be associated with a run.
// Heartbeat creation is a two-step API operation, so this compensating action
// prevents a failed run request from leaving an orphaned task in Agent Manager.
func (c *AgentManagerClient) DeleteTask(ctx context.Context, taskID string) error {
	resp, err := c.doRequestWithRetry(ctx, "DELETE", "/api/v1/tasks/"+taskID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return c.parseError(resp)
	}
	return nil
}

// CreateRun starts a new run for a task.
//
// Run creation can leave a durable
// effect without returning an identity. The outcome is therefore classified so
// the caller can retain an uncertain obligation instead of releasing the
// member's slot on a response that may simply have been lost:
//
//   - marshal failure / non-2xx response -> definitive rejection (no run accepted)
//   - transport failure or malformed 2xx body -> uncertain (run may exist)
func (c *AgentManagerClient) CreateRun(ctx context.Context, req *CreateRunRequest) (*Run, error) {
	if err := ctx.Err(); err != nil {
		return nil, NewDispatchRejectedError(err)
	}
	proof := createRunCaller(ctx)
	if proof.authorization == "" && proof.runIdentity == "" && owneridentity.CreateRunCallerEnforced() {
		return nil, NewDispatchRejectedError(fmt.Errorf("CreateRun caller proof is missing"))
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, NewDispatchRejectedError(fmt.Errorf("marshal request: %w", err))
	}

	// Request-local proof never falls back to anonymous retries or redirects.
	resp, err := c.doRequest(ctx, "POST", "/api/v1/runs", body)
	if err != nil {
		// The request may have reached the owner before the transport failed.
		return nil, NewDispatchUncertainError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusAccepted {
			if cause, ok := approvalPendingResponse(resp); ok {
				return nil, NewApprovalPendingError("creating_run", redactCreateRunCallerError(cause, proof))
			}
			return nil, NewDispatchUncertainError(fmt.Errorf("run creation returned unrecognized 202 response"))
		}
		safeError := redactCreateRunCallerError(c.parseError(resp), proof)
		var ownerError *AgentManagerHTTPError
		if errors.As(safeError, &ownerError) && ownerError != nil && resp.StatusCode == http.StatusConflict && ownerError.Code == "STATE_IDEMPOTENCY_REPLAY_CONFLICT" {
			// Agent Manager emits this exact response only after locating the
			// original run for the key and finding changed replay terms. Preserve
			// the durable obligation so an operator can reconcile the live run.
			return nil, NewDispatchUncertainError(safeError)
		}
		return nil, NewDispatchRejectedError(safeError)
	}

	var result CreateRunResponse
	if err := c.parseResponse(resp, &result); err != nil {
		// The owner answered successfully but the run identity is unreadable.
		return nil, NewDispatchUncertainError(err)
	}
	if result.Run == nil || strings.TrimSpace(result.Run.ID) == "" {
		return nil, NewDispatchUncertainError(fmt.Errorf("run creation returned no run identity"))
	}
	return result.Run, nil
}

// approvalPendingResponse recognizes the exact typed 202 contract used by
// Agent Manager. Unknown 202 bodies are not treated as safe-to-retry because
// their side effects are unspecified.
func approvalPendingResponse(resp *http.Response) (error, bool) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read approval response: %w", err), false
	}
	var result struct {
		Status string `json:"status"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(body, &result) != nil || result.Status != "approval_pending" {
		return fmt.Errorf("unexpected 202 response: %s", strings.TrimSpace(string(body))), false
	}
	if result.Detail == "" {
		result.Detail = "operator approval pending"
	}
	return fmt.Errorf("%s", result.Detail), true
}

// CreateRunDelegated uses explicitly provisioned authority for this request only.
// It never exchanges machine credentials or changes the default client's identity.
func (c *AgentManagerClient) CreateRunDelegated(ctx context.Context, req *CreateRunRequest, ownerToken string) (*Run, error) {
	if req == nil || req.RequestedScopes == nil || strings.TrimSpace(req.RequestedScopes.ExpectedOwnerSubject) == "" || strings.TrimSpace(ownerToken) == "" || strings.ContainsAny(ownerToken, "\r\n\t ") {
		return nil, fmt.Errorf("delegated run requires an explicit owner credential, expected owner and scope narrowing")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return nil, fmt.Errorf("delegated run requires an idempotency key")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal delegated run request")
	}
	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve delegated run owner")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/runs", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create delegated run request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+ownerToken)
	client := *c.httpClient
	// Even a same-origin redirect must not turn this CreateRun credential into
	// authority for a different operation. Caller owns uncertain dispatch recovery.
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("delegated run transport failed; reconcile the original idempotency key")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("delegated run refused: HTTP %d", response.StatusCode)
	}
	var result CreateRunResponse
	if err := c.parseResponse(response, &result); err != nil || result.Run == nil || result.Run.ID == "" {
		return nil, fmt.Errorf("delegated run response unavailable; reconcile the original idempotency key")
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
	check := func() (*Run, error) {
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
		return nil, nil
	}

	// Check immediately before waiting so an already-terminal run does not
	// pay a full polling interval. This also keeps completion latency bounded
	// by the remote request rather than the configured cadence.
	if run, err := check(); err != nil || run != nil {
		return run, err
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if run, err := check(); err != nil || run != nil {
				return run, err
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
	if opts.AgentProfileID != "" {
		params = append(params, "agent_profile_id="+opts.AgentProfileID)
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

// GetEffortEnrollment reads the exact canonical enrollment from Agent
// Manager's existing effort board. The enrollment is declarative identity;
// this read does not grant authority.
func (c *AgentManagerClient) GetEffortEnrollment(ctx context.Context, effortRef string) (*domainpb.EffortEnrollment, error) {
	if strings.TrimSpace(effortRef) == "" {
		return nil, errors.New("effort reference is required")
	}
	body, err := protojson.Marshal(&domainpb.GetEffortBoardRequest{EffortRef: strings.TrimSpace(effortRef), PageSize: 1})
	if err != nil {
		return nil, fmt.Errorf("marshal effort board request: %w", err)
	}
	resp, err := c.doRequestWithRetry(ctx, http.MethodPost, apiconnect.AgentManagerServiceGetEffortBoardProcedure, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read effort board: %w", err)
	}
	var board domainpb.EffortBoard
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(responseBody, &board); err != nil {
		return nil, fmt.Errorf("decode effort board: %w", err)
	}
	for _, row := range board.GetRows() {
		if enrollment := row.GetEnrollment(); enrollment != nil && enrollment.GetEffortRef() == strings.TrimSpace(effortRef) {
			return enrollment, nil
		}
	}
	return nil, fmt.Errorf("agent-manager effort enrollment %q was not present in its board response", effortRef)
}

// ContinueRun sends a continue message to a paused run. A non-empty
// idempotencyKey is forwarded so agent-manager can deduplicate a retried or
// reloaded turn instead of advancing the conversation twice.
func (c *AgentManagerClient) ContinueRun(ctx context.Context, runID string, message string, idempotencyKey string) (*Run, error) {
	body, err := json.Marshal(ContinueRunRequest{Message: message, IdempotencyKey: idempotencyKey})
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

// CreateTypedInvestigation admits a caller-neutral, diagnosis-only
// investigation through Agent Manager's finite lifecycle. The raw JSON is
// preserved so this adapter does not reinterpret the owner's typed result.
func (c *AgentManagerClient) CreateTypedInvestigation(ctx context.Context, request []byte) ([]byte, error) {
	resp, err := c.doRequestWithRetry(ctx, "POST", "/api/v1/investigations", request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}
	return io.ReadAll(resp.Body)
}

// ListTypedInvestigations returns the owner's durable typed records without
// projecting them into legacy Run values.
func (c *AgentManagerClient) ListTypedInvestigations(ctx context.Context, limit int) ([]byte, error) {
	path := "/api/v1/investigations"
	if limit > 0 {
		path += fmt.Sprintf("?limit=%d", limit)
	}
	resp, err := c.doRequestWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}
	return io.ReadAll(resp.Body)
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
			sleep := c.sleep
			if sleep == nil {
				sleep = waitForRetry
			}
			if err := sleep(ctx, backoff[attempt]); err != nil {
				return nil, err
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
		return nil, NewDispatchRejectedError(err)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bodyReader)
	if err != nil {
		return nil, NewDispatchRejectedError(fmt.Errorf("create request: %w", err))
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	if method == http.MethodPost && path == "/api/v1/tasks" {
		if err := ctx.Err(); err != nil {
			return nil, NewDispatchRejectedError(err)
		}
		// The heartbeat creates its task BEFORE its run, in the same qualified
		// context. The task leg must carry the SAME caller proof as the run leg
		// (an unattended heartbeat's service bearer, or a forwarded human/run
		// proof) — otherwise it arrives anonymous and agent-manager 401s before
		// the run is ever reached. Effort callers forward no header on the task
		// leg (their proof is minted per-run at the runs channel).
		proof := createRunCaller(ctx)
		if proof.authorization != "" {
			req.Header.Set("Authorization", proof.authorization)
		}
		if proof.runIdentity != "" {
			req.Header.Set("X-Agent-Identity-Token", proof.runIdentity)
		}
		client := *c.httpClient
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		return client.Do(req)
	}

	if method == http.MethodPost && path == "/api/v1/runs" {
		proof := createRunCaller(ctx)
		if proof.authorization != "" || proof.runIdentity != "" {
			if proof.authorization != "" {
				req.Header.Set("Authorization", proof.authorization)
			}
			if proof.runIdentity != "" {
				req.Header.Set("X-Agent-Identity-Token", proof.runIdentity)
			}
			client := *c.httpClient
			client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
			return client.Do(req)
		}
	}
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
		Code            string `json:"code"`
		Error           string `json:"error"`
		Message         string `json:"message"`
		AdmissionReason string `json:"admission_reason"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		admissionReason := normalizeAgentManagerAdmissionReason(resp.StatusCode, errResp.AdmissionReason)
		if errResp.Error != "" {
			return &AgentManagerHTTPError{StatusCode: resp.StatusCode, Code: errResp.Code, Message: errResp.Error, AdmissionReason: admissionReason}
		}
		if errResp.Message != "" {
			return &AgentManagerHTTPError{StatusCode: resp.StatusCode, Code: errResp.Code, Message: errResp.Message, AdmissionReason: admissionReason}
		}
	}
	return &AgentManagerHTTPError{StatusCode: resp.StatusCode, Code: errResp.Code, Message: fmt.Sprintf("body: %s", string(body))}
}

// prepareCreateRunCaller qualifies the two-step executor before local state or
// task writes. An original service mode must always be freshly minted and can
// never fall back to caller or anonymous authority, independent of AUTH-01.
// Direct raw proxy requests retain the owner's exact-parent verification at
// CreateRun.
func (c *AgentManagerClient) prepareCreateRunCaller(ctx context.Context) (context.Context, error) {
	proof := createRunCaller(ctx)
	if proof.origin == createRunCallerOriginService {
		if proof.runIdentity != "" || proof.binding != "" {
			return nil, fmt.Errorf("service caller mode contains caller-only proof")
		}
		return c.mintServiceCreateRunCaller(ctx)
	}
	// An unattended heartbeat carries no human owner token and no run identity.
	// If a service source is configured, failure to mint is a hard refusal: it
	// must not silently turn a scheduled service dispatch into anonymous work.
	if proof.authorization == "" && proof.runIdentity == "" && c.serviceToken != nil {
		return c.mintServiceCreateRunCaller(ctx)
	}
	headers := http.Header{}
	if proof.authorization != "" {
		headers.Set("Authorization", proof.authorization)
	}
	if proof.runIdentity != "" {
		headers.Set("X-Agent-Identity-Token", proof.runIdentity)
	}
	verifier := c.createCallerVerifier
	if verifier == nil {
		verifier = owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})})
	}
	qualified, err := owneridentity.AuthorizeCreateRunCaller(ctx, headers, verifier, time.Now(), owneridentity.CreateRunCallerEnforced())
	if err != nil {
		return nil, err
	}
	qualifiedProof := createRunCaller(qualified)
	if qualifiedProof.origin == "" {
		origin := createRunCallerOriginUnqualified
		if proof.authorization != "" || proof.runIdentity != "" {
			origin = createRunCallerOriginCaller
		}
		qualified = withCreateRunCallerOrigin(qualified, proof.authorization, proof.runIdentity, origin)
	}
	if createRunCaller(qualified).origin == createRunCallerOriginCaller {
		binding, bindingErr := owneridentity.CreateRunCallerBinding(qualified)
		if bindingErr != nil {
			return nil, fmt.Errorf("verified caller identity binding unavailable: %w", bindingErr)
		}
		qualified = withCreateRunCallerBinding(qualified, binding)
	}
	return qualified, nil
}

func (c *AgentManagerClient) mintServiceCreateRunCaller(ctx context.Context) (context.Context, error) {
	if c == nil || c.serviceToken == nil {
		return nil, fmt.Errorf("prompt-manager service credential source is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	token, err := c.serviceToken.Token(ctx)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("prompt-manager service credential could not be refreshed")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("prompt-manager service credential could not be refreshed")
	}
	return withCreateRunCallerOrigin(ctx, "Bearer "+token, "", createRunCallerOriginService), nil
}
