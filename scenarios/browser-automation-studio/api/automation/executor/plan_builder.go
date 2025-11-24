package executor

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/database"
)

// PlanCompiler emits engine-agnostic execution plans and compiled instructions
// ready for orchestration.
type PlanCompiler interface {
	Compile(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error)
}

// DefaultPlanCompiler supplies the legacy browserless-backed compiler while we
// bring additional engines online.
var DefaultPlanCompiler PlanCompiler = &BrowserlessPlanCompiler{}

// BuildContractsPlan compiles a workflow into the engine-agnostic plan +
// instructions expected by the executor path.
func BuildContractsPlan(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	return BuildContractsPlanWithCompiler(ctx, executionID, workflow, DefaultPlanCompiler)
}

// BuildContractsPlanWithCompiler allows callers to inject a custom compiler
// (e.g., desktop automation) without altering executor orchestration.
func BuildContractsPlanWithCompiler(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow, compiler PlanCompiler) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	if compiler == nil {
		compiler = DefaultPlanCompiler
	}
	return compiler.Compile(ctx, executionID, workflow)
}

func toContractsGraph(plan *compiler.ExecutionPlan) *contracts.PlanGraph {
	if plan == nil {
		return nil
	}
	steps := make([]contracts.PlanStep, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		edges := make([]contracts.PlanEdge, 0, len(step.OutgoingEdges))
		for _, edge := range step.OutgoingEdges {
			edges = append(edges, contracts.PlanEdge{
				ID:         edge.ID,
				Target:     edge.TargetNode,
				Condition:  edge.Condition,
				SourcePort: edge.SourcePort,
				TargetPort: edge.TargetPort,
			})
		}
		converted := contracts.PlanStep{
			Index:     step.Index,
			NodeID:    step.NodeID,
			Type:      string(step.Type),
			Params:    step.Params,
			Outgoing:  edges,
			Metadata:  map[string]string{},
			Context:   map[string]any{},
			Preload:   "",
			SourcePos: nil,
		}
		if step.LoopPlan != nil {
			converted.Loop = toContractsGraph(step.LoopPlan)
		}
		steps = append(steps, converted)
	}
	return &contracts.PlanGraph{Steps: steps}
}
