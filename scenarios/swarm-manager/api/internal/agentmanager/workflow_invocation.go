package agentmanager

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// Invocation is the generic Swarm-to-Agent-Manager workflow boundary. It has
// no domain-specific prompt, result, retry, or apply behavior.
type Invocation struct {
	Owner          string
	WorkflowKey    string
	Input          *structpb.Value
	IdempotencyKey string
	FirstRunNodeID string
}

// InvocationCompletion preserves the immutable input, terminal output, pinned
// revision, and attempt journal that Swarm needs for its domain-side checks.
type InvocationCompletion struct {
	ExecutionID      string
	DefinitionDigest string
	Status           domainpb.WorkflowExecutionStatus
	TerminalCode     string
	BudgetName       string
	Input            *structpb.Value
	Output           *structpb.Value
	Attempts         []*domainpb.WorkflowNodeAttempt
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
		Owner: invocation.Owner, WorkflowKey: invocation.WorkflowKey, Input: invocation.Input,
		IdempotencyKey: strings.TrimSpace(invocation.IdempotencyKey),
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
	start := WorkflowStart{ExecutionID: response.Execution.Id, DefinitionDigest: response.Execution.DefinitionDigest, Status: response.Execution.Status}
	for _, attempt := range trace.Attempts {
		if attempt.NodeId == invocation.FirstRunNodeID {
			start.RunID = strings.TrimSpace(attempt.RunId)
			break
		}
	}
	return start, nil
}

func (s *WorkflowService) CollectWorkflow(ctx context.Context, executionID string) (InvocationCompletion, error) {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return InvocationCompletion{}, fmt.Errorf("%w: execution id is required", ErrRequestFailed)
	}
	waited, err := s.client.WaitWorkflowExecution(ctx, executionID, workshopWaitTimeoutSeconds)
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
	completion := InvocationCompletion{ExecutionID: executionID, DefinitionDigest: response.Execution.DefinitionDigest, Status: response.Execution.Status, Input: response.Execution.Input, Output: response.Execution.Output, Attempts: trace.Attempts}
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
