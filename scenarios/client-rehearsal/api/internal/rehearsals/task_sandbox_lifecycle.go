package rehearsals

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	taskSandboxProfileName    = "workflow-apis"
	taskSandboxProfileVersion = "1.0.0"
)

// WorkspaceSandboxLifecycleEvidence is intentionally a compact receipt. It
// records the owner-bound task, the exact synthetic profile digest and the
// teardown result without retaining service URLs or workflow data.
type WorkspaceSandboxLifecycleEvidence struct {
	SandboxID          string
	Owner              string
	ScopeSHA256        string
	ProfileSHA256      string
	OperationCount     int
	SyntheticTargets   bool
	IsolationClaim     string
	RouteHitsStatus    string
	RouteHits          *HTTPMockRouteHitsReceipt
	RouteHitsError     string
	N8NRuntimeStatus   string
	N8NExecution       *N8NWorkflowExecutionEvidence
	N8NRuntimeError    string
	TeardownStatus     string
	TeardownError      string
	CleanupConfirmedAt time.Time
}

// CreateAndRunTaskScoped is the internal service entry point for a single
// synthetic companion profile. Workspace Sandbox derives the task scope and
// owner from server configuration; imported workflow data can never choose it. The selected
// Engine still executes in-process, so this path proves Sandbox profile
// lifecycle and ownership only; it does not prove workflow traffic reached
// the companion or that the Sandbox network is isolated from egress.
func (s *service) CreateAndRunTaskScoped(ctx context.Context, input CreateInput, task WorkspaceSandboxTaskRequest) (TaskScopedRun, error) {
	if s == nil || s.sandbox == nil {
		return TaskScopedRun{}, errors.New("task-scoped rehearsal requires a configured Workspace Sandbox client")
	}
	if err := validateWorkspaceSandboxTaskScope(task); err != nil {
		return TaskScopedRun{}, err
	}
	input.taskSandbox = &task
	rehearsal, err := s.CreateAndRun(ctx, input)
	if err != nil {
		return TaskScopedRun{}, err
	}
	evidence := lifecycleEvidenceFromTrace(rehearsal.Trace)
	return TaskScopedRun{Rehearsal: rehearsal, Lifecycle: evidence}, nil
}

func validateWorkspaceSandboxTaskScope(task WorkspaceSandboxTaskRequest) error {
	if !taskScopeIDPattern.MatchString(task.TaskID) || task.TaskID == "." || task.TaskID == ".." || (task.Name != "" && !sandboxNamePattern.MatchString(task.Name)) {
		return errors.New("task-scoped rehearsal requires a bounded logical task ID and optional name")
	}
	return nil
}

func (s *service) beginTaskSandbox(ctx context.Context, task WorkspaceSandboxTaskRequest, profile HTTPMockProfile) (WorkspaceSandboxProfileLease, WorkspaceSandboxLifecycleEvidence, error) {
	var lease WorkspaceSandboxProfileLease
	var err error
	if s.n8n == nil {
		lease, err = s.sandbox.CreateAndStartTaskProfile(ctx, task, taskSandboxProfileName, taskSandboxProfileVersion, profile)
	} else {
		created, createErr := s.sandbox.CreateTaskSandbox(ctx, task)
		if createErr != nil {
			return WorkspaceSandboxProfileLease{}, WorkspaceSandboxLifecycleEvidence{}, createErr
		}
		runtime, startErr := s.n8n.Start(ctx, created.ID)
		if startErr != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			cleanupErr := s.sandbox.DeleteTaskSandbox(cleanupCtx, created.ID)
			return WorkspaceSandboxProfileLease{}, WorkspaceSandboxLifecycleEvidence{}, errors.Join(fmt.Errorf("start task-owned n8n runtime: %w", startErr), cleanupErr)
		}
		lease, err = s.sandbox.StartTaskProfileOnSandbox(ctx, created, taskSandboxProfileName, taskSandboxProfileVersion, profile)
		lease.Runtime = runtime
		if err != nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			stopErr := s.n8n.Stop(cleanupCtx, runtime)
			var cleanupErr error
			if stopErr == nil {
				cleanupErr = s.sandbox.DeleteTaskSandbox(cleanupCtx, created.ID)
			}
			return WorkspaceSandboxProfileLease{}, WorkspaceSandboxLifecycleEvidence{}, errors.Join(fmt.Errorf("start task Sandbox companion profile: %w", err), stopErr, cleanupErr)
		}
	}
	if err != nil {
		return WorkspaceSandboxProfileLease{}, WorkspaceSandboxLifecycleEvidence{}, err
	}
	resolvedTask := lease.Task
	scopeHash := sha256.Sum256([]byte(resolvedTask.ScopePath))
	evidence := WorkspaceSandboxLifecycleEvidence{
		SandboxID:        lease.SandboxID,
		Owner:            resolvedTask.Owner,
		ScopeSHA256:      hex.EncodeToString(scopeHash[:]),
		ProfileSHA256:    lease.Receipt.ProfileSHA256,
		OperationCount:   len(lease.Receipt.OperationAddresses),
		SyntheticTargets: allOperationTargetsSynthetic(lease.Receipt.OperationAddresses),
		IsolationClaim:   "Workspace Sandbox owner verified profile digest and synthetic-only operation routes; no live network isolation claim is inferred",
		N8NRuntimeStatus: "not-started",
		TeardownStatus:   "pending",
	}
	if lease.Runtime.RuntimeID != "" {
		evidence.N8NRuntimeStatus = "running"
	}
	if !evidence.SyntheticTargets || evidence.OperationCount != len(profile.Operations) {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		cleanupErr := s.stopAndDeleteTaskSandbox(cleanupCtx, lease, resolvedTask)
		if cleanupErr != nil {
			return lease, evidence, fmt.Errorf("Sandbox profile receipt failed synthetic route validation (%v); cleanup failed: %w", errors.New("unexpected operation address"), cleanupErr)
		}
		return lease, evidence, errors.New("Workspace Sandbox profile receipt failed synthetic route validation; created task was stopped and deleted")
	}
	return lease, evidence, nil
}

func (s *service) stopAndDeleteTaskSandbox(ctx context.Context, lease WorkspaceSandboxProfileLease, task WorkspaceSandboxTaskRecord) error {
	if err := s.sandbox.verifyTaskSandbox(ctx, lease.SandboxID, task); err != nil {
		return fmt.Errorf("verify task ownership before teardown: %w", err)
	}
	if err := s.sandbox.StopProfile(ctx, lease.SandboxID, lease.Name); err != nil {
		return fmt.Errorf("stop task companion: %w", err)
	}
	if err := s.sandbox.deleteTaskSandboxNoContent(ctx, lease.SandboxID); err != nil {
		return fmt.Errorf("delete task Sandbox: %w", err)
	}
	if err := s.sandbox.confirmTaskSandboxDeleted(ctx, lease.SandboxID); err != nil {
		return fmt.Errorf("confirm task Sandbox deletion: %w", err)
	}
	return nil
}

func (c *WorkspaceSandboxProfileClient) verifyTaskSandbox(ctx context.Context, sandboxID string, task WorkspaceSandboxTaskRecord) error {
	var found struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		ProjectRoot    string `json:"projectRoot"`
		ScopePath      string `json:"scopePath"`
		Owner          string `json:"owner"`
		OwnerType      string `json:"ownerType"`
		IdempotencyKey string `json:"idempotencyKey"`
		Status         string `json:"status"`
	}
	if err := c.requestJSON(ctx, http.MethodGet, "/api/v1/sandboxes/"+url.PathEscape(sandboxID), nil, &found); err != nil {
		return err
	}
	if found.ID != sandboxID || found.Name != task.Name || found.ProjectRoot != task.ProjectRoot || found.ScopePath != task.ScopePath || found.Owner != task.Owner || found.OwnerType != "task" || found.IdempotencyKey != task.IdempotencyKey || (found.Status != "active" && found.Status != "stopped") {
		return errors.New("Workspace Sandbox owner, scope, or lifecycle state changed before teardown")
	}
	return nil
}

func (c *WorkspaceSandboxProfileClient) deleteTaskSandboxNoContent(ctx context.Context, sandboxID string) error {
	response, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/sandboxes/"+url.PathEscape(sandboxID))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Workspace Sandbox delete returned HTTP %d; expected 204", response.StatusCode)
	}
	return nil
}

func (c *WorkspaceSandboxProfileClient) confirmTaskSandboxDeleted(ctx context.Context, sandboxID string) error {
	response, err := c.doRequest(ctx, http.MethodGet, "/api/v1/sandboxes/"+url.PathEscape(sandboxID))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		return fmt.Errorf("Workspace Sandbox task remains addressable after delete (HTTP %d)", response.StatusCode)
	}
	return nil
}

func (c *WorkspaceSandboxProfileClient) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	endpoint := *c.baseURL
	endpoint.Path += path
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Sandbox request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Workspace Sandbox: %w", err)
	}
	return response, nil
}

func allOperationTargetsSynthetic(addresses []HTTPMockOperationAddress) bool {
	if len(addresses) == 0 {
		return false
	}
	for _, address := range addresses {
		parsed, err := url.Parse(address.URL)
		if err != nil || parsed.Scheme != "http" || !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".synthetic.invalid") {
			return false
		}
	}
	return true
}

func lifecycleEvidenceFromTrace(trace []TraceEvent) WorkspaceSandboxLifecycleEvidence {
	var evidence WorkspaceSandboxLifecycleEvidence
	for _, event := range trace {
		if event.Operation == "sandbox task lifecycle receipt" {
			_ = json.Unmarshal([]byte(event.ResponseSummary), &evidence)
		}
	}
	return evidence
}
