package agentmanager

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

const planRepairWorkflowKey = "swarm-manager/plan-repair"

// PlanRepairSnapshot is the bounded evidence Swarm gives a declared repair
// workflow. Plan Manager remains the validator of any candidate it returns.
type PlanRepairSnapshot struct {
	EntityKind         string
	EntityName         string
	EntityVersion      string
	PlanReference      string
	PlanContent        string
	FrontierDigest     string
	ValidationFindings []any
	CheckedAt          string
	MaxRepairAttempts  int
}

type PlanRepairWorkflowCompletion struct {
	ExecutionID      string
	DefinitionDigest string
	Status           string
	Result           json.RawMessage
}

type PlanRepairWorkflowService interface {
	StartPlanRepair(context.Context, PlanRepairSnapshot, string) (WorkflowStart, error)
	CollectPlanRepair(context.Context, string) (PlanRepairWorkflowCompletion, error)
}

func (s *WorkflowService) StartPlanRepair(ctx context.Context, snapshot PlanRepairSnapshot, idempotencyKey string) (WorkflowStart, error) {
	input, err := structpb.NewValue(map[string]any{
		"entity":      map[string]any{"kind": snapshot.EntityKind, "name": snapshot.EntityName, "version": snapshot.EntityVersion},
		"plan":        map[string]any{"reference": snapshot.PlanReference, "content": snapshot.PlanContent, "frontierDigest": snapshot.FrontierDigest},
		"validation":  map[string]any{"findings": snapshot.ValidationFindings, "checkedAt": snapshot.CheckedAt},
		"constraints": map[string]any{"maxRepairAttempts": snapshot.MaxRepairAttempts},
	})
	if err != nil {
		return WorkflowStart{}, fmt.Errorf("encode plan repair snapshot: %w", err)
	}
	return s.StartWorkflow(ctx, Invocation{Owner: "swarm-manager", WorkflowKey: planRepairWorkflowKey, Input: input, IdempotencyKey: idempotencyKey, FirstRunNodeID: "repair"})
}

func (s *WorkflowService) CollectPlanRepair(ctx context.Context, executionID string) (PlanRepairWorkflowCompletion, error) {
	invocation, err := s.CollectWorkflow(ctx, executionID)
	if err != nil {
		return PlanRepairWorkflowCompletion{}, err
	}
	if invocation.Output == nil {
		return PlanRepairWorkflowCompletion{}, fmt.Errorf("%w: plan repair output is missing", ErrRequestFailed)
	}
	output, ok := invocation.Output.AsInterface().(map[string]any)
	if !ok {
		return PlanRepairWorkflowCompletion{}, fmt.Errorf("%w: plan repair output is not an object", ErrRequestFailed)
	}
	result, err := json.Marshal(output["result"])
	if err != nil {
		return PlanRepairWorkflowCompletion{}, fmt.Errorf("encode plan repair result: %w", err)
	}
	return PlanRepairWorkflowCompletion{ExecutionID: invocation.ExecutionID, DefinitionDigest: invocation.DefinitionDigest, Status: invocation.Status.String(), Result: result}, nil
}
