package executor

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/compiler"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
)

// BuildContractsPlan compiles a workflow into the engine-agnostic plan +
// instructions expected by the new executor path.
func BuildContractsPlan(ctx context.Context, executionID uuid.UUID, workflow *database.Workflow) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	plan, err := compiler.CompileWorkflow(workflow)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}
	instructions, err := runtime.InstructionsFromPlan(ctx, plan)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	compiled := make([]contracts.CompiledInstruction, 0, len(instructions))
	for _, instr := range instructions {
		params := map[string]any{}
		raw, _ := json.Marshal(instr.Params)
		_ = json.Unmarshal(raw, &params)

		contextMap := map[string]any{}
		if len(instr.Context) > 0 {
			rawCtx, _ := json.Marshal(instr.Context)
			_ = json.Unmarshal(rawCtx, &contextMap)
		}

		compiled = append(compiled, contracts.CompiledInstruction{
			Index:       instr.Index,
			NodeID:      instr.NodeID,
			Type:        instr.Type,
			Params:      params,
			PreloadHTML: instr.PreloadHTML,
			Context:     contextMap,
			Metadata:    map[string]string{},
		})
	}

	return contracts.ExecutionPlan{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflow.ID,
		Instructions:   compiled,
		Graph:          toContractsGraph(plan),
		Metadata:       plan.Metadata,
		CreatedAt:      time.Now().UTC(),
	}, compiled, nil
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
