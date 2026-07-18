package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const phasedPlanWorkflowKey = "swarm-manager/phased-plan-drain"

// PhasedPlanSnapshot is the immutable, authorized execution frontier supplied
// by Swarm Manager. Agent Manager sees only generic workflow input values.
type PhasedPlanSnapshot struct {
	PlanReference  string
	FrontierDigest string
	ExecutionID    string
	EntityKind     string
	EntityName     string
	EntityVersion  string
	MaxSlices      int
	WriteScope     []string
}

type WorkflowAttemptProvenance struct {
	NodeID          string
	Ordinal         int32
	Strategy        string
	RunID           string
	ConversationID  string
	SourceAttemptID string
	ProfileIdentity string
}

type PhasedPlanWorkflowCompletion struct {
	ExecutionID      string
	DefinitionDigest string
	Status           domainpb.WorkflowExecutionStatus
	TerminalCode     string
	BudgetName       string
	ConsumerID       string
	EntityKind       string
	EntityName       string
	EntityVersion    string
	FrontierDigest   string
	Result           json.RawMessage
	Attempts         []WorkflowAttemptProvenance
}

// PhasedPlanWorkflowService is the execution-domain command/result surface.
// It cannot create or continue Agent Manager runs directly.
type PhasedPlanWorkflowService interface {
	StartPhasedPlan(context.Context, PhasedPlanSnapshot, string) (WorkflowStart, error)
	CollectPhasedPlan(context.Context, string) (PhasedPlanWorkflowCompletion, error)
	SignalPhasedPlanApproval(context.Context, string, string, string, string) error
	CancelPhasedPlan(context.Context, string, string) error
}

func (s *WorkflowService) StartPhasedPlan(ctx context.Context, snapshot PhasedPlanSnapshot, idempotencyKey string) (WorkflowStart, error) {
	// Workflow definitions are registered by agent-manager's startup declaration
	// sweep; the consumer no longer reconciles per start.
	writeScope := make([]any, len(snapshot.WriteScope))
	for i, path := range snapshot.WriteScope {
		writeScope[i] = path
	}
	input, err := structpb.NewValue(map[string]any{
		"plan": map[string]any{"reference": snapshot.PlanReference, "frontierDigest": snapshot.FrontierDigest},
		"consumer": map[string]any{
			"executionId": snapshot.ExecutionID, "entityKind": snapshot.EntityKind,
			"entityName": snapshot.EntityName, "entityVersion": snapshot.EntityVersion,
		},
		"constraints": map[string]any{"maxSlices": snapshot.MaxSlices, "writeScope": writeScope},
	})
	if err != nil {
		return WorkflowStart{}, fmt.Errorf("encode phased plan snapshot: %w", err)
	}
	return s.StartWorkflow(ctx, Invocation{Owner: "swarm-manager", WorkflowKey: phasedPlanWorkflowKey, Input: input, IdempotencyKey: idempotencyKey, FirstRunNodeID: "slice"})
}

func (s *WorkflowService) CollectPhasedPlan(ctx context.Context, executionID string) (PhasedPlanWorkflowCompletion, error) {
	executionID = strings.TrimSpace(executionID)
	if executionID == "" {
		return PhasedPlanWorkflowCompletion{}, fmt.Errorf("%w: execution id is required", ErrRequestFailed)
	}
	invocation, err := s.CollectWorkflow(ctx, executionID)
	if err != nil {
		return PhasedPlanWorkflowCompletion{}, err
	}
	if invocation.Input == nil {
		return PhasedPlanWorkflowCompletion{}, fmt.Errorf("%w: authorized workflow input is incomplete", ErrRequestFailed)
	}
	input, ok := invocation.Input.AsInterface().(map[string]any)
	if !ok {
		return PhasedPlanWorkflowCompletion{}, fmt.Errorf("%w: workflow input is not an object", ErrRequestFailed)
	}
	consumer, ok := input["consumer"].(map[string]any)
	if !ok {
		return PhasedPlanWorkflowCompletion{}, fmt.Errorf("%w: workflow consumer correlation is missing", ErrRequestFailed)
	}
	plan, ok := input["plan"].(map[string]any)
	if !ok {
		return PhasedPlanWorkflowCompletion{}, fmt.Errorf("%w: workflow plan frontier is missing", ErrRequestFailed)
	}
	completion := PhasedPlanWorkflowCompletion{
		ExecutionID: executionID, DefinitionDigest: invocation.DefinitionDigest, Status: invocation.Status,
		ConsumerID: stringValue(consumer["executionId"]), EntityKind: stringValue(consumer["entityKind"]),
		EntityName: stringValue(consumer["entityName"]), EntityVersion: stringValue(consumer["entityVersion"]),
		FrontierDigest: stringValue(plan["frontierDigest"]),
	}
	completion.TerminalCode, completion.BudgetName = invocation.TerminalCode, invocation.BudgetName
	if invocation.Output != nil {
		if output, ok := invocation.Output.AsInterface().(map[string]any); ok {
			completion.Result, err = json.Marshal(output["result"])
			if err != nil {
				return PhasedPlanWorkflowCompletion{}, fmt.Errorf("encode phased plan result: %w", err)
			}
		}
	}
	for _, attempt := range invocation.Attempts {
		completion.Attempts = append(completion.Attempts, WorkflowAttemptProvenance{
			NodeID: attempt.NodeId, Ordinal: attempt.Ordinal, Strategy: attempt.Strategy,
			RunID: attempt.RunId, ConversationID: attempt.ConversationId,
			SourceAttemptID: attempt.SourceAttemptId, ProfileIdentity: attempt.ProfileIdentity,
		})
	}
	return completion, nil
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

func (s *WorkflowService) SignalPhasedPlanApproval(ctx context.Context, executionID, consumerExecutionID, actor, idempotencyKey string) error {
	payload, err := structpb.NewValue(map[string]any{"executionId": consumerExecutionID, "actor": actor})
	if err != nil {
		return err
	}
	return s.SignalWorkflow(ctx, executionID, "slice_approved", payload, idempotencyKey)
}

func (s *WorkflowService) CancelPhasedPlan(ctx context.Context, executionID, reason string) error {
	return s.CancelWorkflow(ctx, executionID, "cancel-"+executionID, reason)
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
