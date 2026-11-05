package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// Invocation is the generic Swarm-to-Agent-Manager workflow boundary. It has
// no domain-specific prompt, result, retry, or apply behavior.
type Invocation struct {
	Owner          string
	WorkflowKey    string
	ApprovalDigest string
	WorkflowDigest string
	GrantDigest    string
	Input          *structpb.Value
	IdempotencyKey string
	FirstRunNodeID string
	Activity       *WorkflowActivity
	// EngagementGrant is an owner-issued ceiling. It is copied verbatim into
	// Agent Manager's admission request and cannot be widened by this adapter.
	EngagementGrant      *domainpb.WorkflowEngagementGrant
	ExecutionPreferences *domainpb.ExecutionPreferences
}

// InvocationCompletion preserves the immutable input, terminal output, pinned
// revision, and attempt journal that Swarm needs for its domain-side checks.
type InvocationCompletion struct {
	ExecutionID      string
	WorkflowDigest   string
	ApprovalDigest   string
	GrantDigest      string
	DefinitionDigest string
	Status           domainpb.WorkflowExecutionStatus
	TerminalCode     string
	BudgetName       string
	Input            *structpb.Value
	Output           *structpb.Value
	Attempts         []*domainpb.WorkflowNodeAttempt
	BudgetUsage      *domainpb.WorkflowBudgetUsage
	ChargeReceipt    *domainpb.ChargeReceipt
	// WallSeconds is derived from the owner's durable created/ended timestamps.
	// It is conservative elapsed time, not a provider active-time meter.
	WallSeconds int64
}

// WorkflowStartGuard is injected at the composition edge. It lets Swarm apply
// transition-registry and integration preflight policy without teaching this
// transport package any domain policy.
type WorkflowStartGuard func(context.Context, string) error

// WorkflowInvoker is the minimal workflow boundary used by domain adapters.
// Keeping it small makes domain-side apply code testable without an HTTP client.
type WorkflowInvoker interface {
	StartWorkflow(context.Context, Invocation) (WorkflowStart, error)
	CollectWorkflow(context.Context, string) (InvocationCompletion, error)
}

// workflowWaitTimeoutSeconds bounds each server-side blocking wait. It stays
// below the HTTP client timeout so callers receive a durable not-ready result
// without cancelling the workflow itself.
const workflowWaitTimeoutSeconds = 15

func (s *WorkflowService) SetStartGuard(guard WorkflowStartGuard) { s.startGuard = guard }

func (s *WorkflowService) StartWorkflow(ctx context.Context, invocation Invocation) (WorkflowStart, error) {
	if strings.TrimSpace(invocation.Owner) == "" || strings.TrimSpace(invocation.WorkflowKey) == "" || invocation.Input == nil {
		return WorkflowStart{}, fmt.Errorf("%w: workflow owner, key, and input are required", ErrRequestFailed)
	}
	if s.startGuard != nil {
		if err := s.startGuard(ctx, invocation.WorkflowKey); err != nil {
			return WorkflowStart{}, err
		}
	}
	response, err := s.client.StartWorkflowExecution(ctx, &apipb.StartWorkflowExecutionRequest{
		Owner: invocation.Owner, WorkflowKey: invocation.WorkflowKey, DefinitionDigest: invocation.WorkflowDigest, Input: invocation.Input,
		IdempotencyKey: strings.TrimSpace(invocation.IdempotencyKey), ApprovalDigest: invocation.ApprovalDigest, GrantDigest: invocation.GrantDigest,
		EngagementGrant:      invocation.EngagementGrant,
		ExecutionPreferences: invocation.ExecutionPreferences,
	})
	if err != nil {
		return WorkflowStart{}, err
	}
	if response.Execution == nil || strings.TrimSpace(response.Execution.Id) == "" {
		return WorkflowStart{}, fmt.Errorf("%w: workflow start omitted execution", ErrRequestFailed)
	}
	trace, err := s.client.GetWorkflowExecutionTrace(ctx, response.Execution.Id)
	if err != nil {
		return WorkflowStart{}, err
	}
	start := WorkflowStart{ExecutionID: response.Execution.Id, WorkflowDigest: response.Execution.DefinitionDigest, DefinitionDigest: response.Execution.DefinitionDigest, ApprovalDigest: response.Execution.ApprovalDigest, GrantDigest: response.Execution.GrantDigest, Status: response.Execution.Status}
	for _, attempt := range trace.Attempts {
		if attempt.NodeId == invocation.FirstRunNodeID {
			start.RunID = strings.TrimSpace(attempt.RunId)
			break
		}
	}
	if invocation.Activity != nil && s.activityRecorder != nil {
		activity := *invocation.Activity
		activity.WorkflowKey = invocation.WorkflowKey
		if err := s.activityRecorder.RecordWorkflowStart(ctx, activity, start); err != nil {
			return WorkflowStart{}, fmt.Errorf("%w: record workflow activity: %v", ErrRequestFailed, err)
		}
	}
	return start, nil
}

// GetWorkflowExecutionByIdempotencyKey is the read side of start
// reconciliation. It never retries workflow creation.
func (s *WorkflowService) GetWorkflowExecutionByIdempotencyKey(ctx context.Context, key string) (WorkflowStart, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return WorkflowStart{}, fmt.Errorf("%w: idempotency key is required", ErrRequestFailed)
	}
	requestURL := "/api/v1/workflow-executions?idempotency_key=" + url.QueryEscape(key) + "&limit=1"
	resp, err := s.client.doRequest(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return WorkflowStart{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WorkflowStart{}, readErrorResponse(resp)
	}
	var result apipb.ListWorkflowExecutionsResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return WorkflowStart{}, err
	}
	if len(result.Executions) == 0 || result.Executions[0] == nil || strings.TrimSpace(result.Executions[0].Id) == "" {
		return WorkflowStart{}, ErrWorkflowNotFound
	}
	execution := result.Executions[0]
	return WorkflowStart{ExecutionID: execution.Id, WorkflowDigest: execution.DefinitionDigest, DefinitionDigest: execution.DefinitionDigest, ApprovalDigest: execution.ApprovalDigest, GrantDigest: execution.GrantDigest, Status: execution.Status}, nil
}

// InspectWorkflowStart resolves the original start without advancing or
// recreating it, including the input required to verify its consumer binding.
func (s *WorkflowService) InspectWorkflowStart(ctx context.Context, workflowKey, key string) (*domainpb.WorkflowExecution, error) {
	started, err := s.GetWorkflowExecutionByIdempotencyKey(ctx, key)
	if err != nil {
		return nil, err
	}
	response, err := s.client.GetWorkflowExecutionResult(ctx, started.ExecutionID)
	if err != nil {
		return nil, err
	}
	execution := response.GetExecution()
	if execution == nil || execution.Id != started.ExecutionID || execution.IdempotencyKey != key || execution.WorkflowKey != workflowKey {
		return nil, fmt.Errorf("workflow lookup returned a different start identity")
	}
	return execution, nil
}

func (s *WorkflowService) CollectWorkflow(ctx context.Context, executionID string) (InvocationCompletion, error) {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return InvocationCompletion{}, fmt.Errorf("%w: execution id is required", ErrRequestFailed)
	}
	waited, err := s.client.WaitWorkflowExecution(ctx, executionID, workflowWaitTimeoutSeconds)
	if err != nil {
		return InvocationCompletion{}, err
	}
	if waited.Execution == nil || waited.TimedOut || !terminalWorkflowStatus(waited.Execution.Status) {
		return InvocationCompletion{}, fmt.Errorf("%w: workflow execution is not terminal", ErrWorkflowNotReady)
	}
	response, err := s.client.GetWorkflowExecutionResult(ctx, executionID)
	if err != nil {
		return InvocationCompletion{}, err
	}
	if response.Execution == nil || !terminalWorkflowStatus(response.Execution.Status) {
		return InvocationCompletion{}, fmt.Errorf("%w: workflow execution is not terminal", ErrWorkflowNotReady)
	}
	trace, err := s.client.GetWorkflowExecutionTrace(ctx, executionID)
	if err != nil {
		return InvocationCompletion{}, err
	}
	completion := InvocationCompletion{ExecutionID: executionID, WorkflowDigest: response.Execution.DefinitionDigest, DefinitionDigest: response.Execution.DefinitionDigest, ApprovalDigest: response.Execution.ApprovalDigest, GrantDigest: response.Execution.GrantDigest, Status: response.Execution.Status, Input: response.Execution.Input, Output: response.Execution.Output, Attempts: trace.Attempts, BudgetUsage: response.Execution.BudgetUsage, ChargeReceipt: response.Execution.ChargeReceipt}
	if response.Execution.CreatedAt != nil && response.Execution.EndedAt != nil {
		d := response.Execution.EndedAt.AsTime().Sub(response.Execution.CreatedAt.AsTime())
		if d > 0 {
			completion.WallSeconds = int64((d + time.Second - 1) / time.Second)
		}
	}
	if reason := response.Execution.TerminalReason; reason != nil {
		completion.TerminalCode, completion.BudgetName = reason.Code, reason.BudgetName
	}
	return completion, nil
}

func (s *WorkflowService) SignalWorkflow(ctx context.Context, executionID, signal string, payload *structpb.Value, idempotencyKey string) error {
	_, err := s.client.workflowOperationPost(ctx, "/api/v1/workflow-executions/"+url.PathEscape(executionID)+"/signals", &apipb.SignalWorkflowExecutionRequest{ExecutionId: executionID, Signal: signal, Payload: payload, IdempotencyKey: idempotencyKey})
	return err
}

func (s *WorkflowService) CancelWorkflow(ctx context.Context, executionID, idempotencyKey, reason string) error {
	_, err := s.client.workflowOperationPost(ctx, "/api/v1/workflow-executions/"+url.PathEscape(executionID)+"/cancel", &apipb.WorkflowExecutionOperationRequest{ExecutionId: executionID, IdempotencyKey: idempotencyKey, Reason: reason})
	return err
}

func terminalWorkflowStatus(status domainpb.WorkflowExecutionStatus) bool {
	switch status {
	case domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BUDGET_EXHAUSTED,
		domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}

func (c *HTTPClient) workflowOperationPost(ctx context.Context, path string, req proto.Message) (*apipb.WorkflowExecutionOperationResponse, error) {
	body, err := protoJSONMarshal.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readErrorResponse(resp)
	}
	var result apipb.WorkflowExecutionOperationResponse
	if err := decodeProtoResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ReviewWorkflowSnapshot reads the terminal owner graph into a bounded immutable
// review input. Run summaries remain agent reports; owner status, identities and
// usage are observations, not evidence that every product criterion passed.
func (s *WorkflowService) ReviewWorkflowSnapshot(ctx context.Context, executionID, consumerID, approvalDigest, grantDigest string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	type pendingWorkflow struct{ id, parent, attempt string }
	queue := []pendingWorkflow{{id: executionID}}
	seen := map[string]bool{}
	workflows := []any{}
	reads := 0
	for len(queue) > 0 {
		pending := queue[0]
		queue = queue[1:]
		if seen[pending.id] {
			return nil, fmt.Errorf("review evidence repeats workflow %s", pending.id)
		}
		seen[pending.id] = true
		reads += 2
		if reads > 1024 {
			return nil, fmt.Errorf("review evidence exceeds 1024 owner reads; inspect workflow %s separately", executionID)
		}
		result, err := s.client.GetWorkflowExecutionResult(ctx, pending.id)
		if err != nil {
			return nil, fmt.Errorf("review workflow %s result: %w", pending.id, err)
		}
		wf := result.GetExecution()
		if wf == nil || wf.Id != pending.id || !terminalWorkflowStatus(wf.Status) {
			return nil, fmt.Errorf("review workflow %s has no matching terminal result", pending.id)
		}
		if pending.parent == "" {
			consumer, _ := wf.GetInput().GetStructValue().GetFields()["consumer"].GetStructValue().GetFields()["executionId"].AsInterface().(string)
			if consumer != consumerID || wf.ApprovalDigest != approvalDigest || wf.GrantDigest != grantDigest {
				return nil, fmt.Errorf("review workflow %s does not match the execution approval, grant and consumer", pending.id)
			}
		} else if wf.ParentExecutionId != pending.parent || wf.ParentAttemptId != pending.attempt || wf.ApprovalDigest != approvalDigest {
			return nil, fmt.Errorf("review child workflow %s does not match its parent attempt and approval", pending.id)
		}
		trace, err := s.client.GetWorkflowExecutionTrace(ctx, pending.id)
		if err != nil {
			return nil, fmt.Errorf("review workflow %s trace: %w", pending.id, err)
		}
		attempts := []any{}
		for _, attempt := range trace.Attempts {
			if attempt == nil || attempt.ExecutionId != pending.id {
				return nil, fmt.Errorf("review workflow %s has a mismatched attempt", pending.id)
			}
			facts := map[string]any{"id": attempt.Id, "node": attempt.NodeId, "status": attempt.Status, "strategy": attempt.Strategy, "runId": attempt.RunId, "conversationId": attempt.ConversationId, "childExecutionId": attempt.ChildExecutionId}
			if attempt.RunId != "" {
				reads++
				if reads > 1024 {
					return nil, fmt.Errorf("review evidence exceeds 1024 owner reads; inspect workflow %s separately", executionID)
				}
				run, err := s.client.GetRun(ctx, attempt.RunId)
				if err != nil {
					return nil, fmt.Errorf("review run %s: %w", attempt.RunId, err)
				}
				if run == nil || run.Id != attempt.RunId || run.IdempotencyKey != attempt.IdempotencyKey {
					return nil, fmt.Errorf("review run %s does not match the owner attempt", attempt.RunId)
				}
				facts["run"] = map[string]any{"id": run.Id, "status": run.Status.String(), "sandboxId": run.SandboxId, "summary": run.Summary, "finalizationStatus": run.FinalizationStatus.String(), "finalizationError": run.FinalizationError, "finalizedAt": run.FinalizedAt, "reportedFinalOutput": run.GetResult().GetFinalOutput()}
			}
			if attempt.ChildExecutionId != "" {
				queue = append(queue, pendingWorkflow{id: attempt.ChildExecutionId, parent: pending.id, attempt: attempt.Id})
			}
			attempts = append(attempts, facts)
		}
		workflows = append(workflows, map[string]any{"id": wf.Id, "workflowKey": wf.WorkflowKey, "definitionDigest": wf.DefinitionDigest, "approvalDigest": wf.ApprovalDigest, "grantDigest": wf.GrantDigest, "status": wf.Status.String(), "output": wf.Output, "terminalReason": wf.TerminalReason, "usage": wf.BudgetUsage, "chargeReceipt": wf.ChargeReceipt, "edgeTraversals": wf.EdgeTraversals, "attempts": attempts})
	}
	encoded, err := json.Marshal(map[string]any{"state": "available", "source": "agent-manager terminal result, trace and run APIs", "observedAt": time.Now().UTC().Format(time.RFC3339Nano), "workflowExecutionId": executionID, "trust": "Owner lifecycle, identity and accounting observations; run summaries and final outputs are agent reports. These do not replace test-producer or applied-change provenance receipts.", "workflows": workflows})
	if err != nil {
		return nil, err
	}
	if len(encoded) > 512*1024 {
		return nil, fmt.Errorf("review evidence exceeds 512 KiB; inspect workflow %s separately", executionID)
	}
	return encoded, nil
}
