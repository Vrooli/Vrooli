// Package compiler provides workflow compilation utilities.
// This file provides adapters for converting compiler types to contracts types,
// centralizing the conversion logic that was previously scattered in executor/.
package compiler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
)

// CompileWorkflowToContracts compiles a workflow directly to contracts.ExecutionPlan.
// This is the preferred entry point for callers who need the canonical ExecutionPlan type.
// It internally calls CompileWorkflow and performs the type conversion.
func CompileWorkflowToContracts(ctx context.Context, executionID uuid.UUID, workflow *basapi.WorkflowSummary) (contracts.ExecutionPlan, []contracts.CompiledInstruction, error) {
	_ = ctx // Reserved for future use (e.g., context-aware compilation)

	plan, err := CompileWorkflow(workflow)
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	workflowID, err := uuid.Parse(workflow.GetId())
	if err != nil {
		return contracts.ExecutionPlan{}, nil, err
	}

	// Convert steps to compiled instructions (flat representation)
	instructions := make([]contracts.CompiledInstruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		instr := contracts.CompiledInstruction{
			Index:       step.Index,
			NodeID:      step.NodeID,
			PreloadHTML: "",
			Context:     map[string]any{},
			Metadata:    map[string]string{},
		}

		// Build typed Action field from step type/params
		action, actionErr := BuildActionDefinition(string(step.Type), step.Params)
		if actionErr != nil {
			logrus.WithFields(logrus.Fields{
				"execution_id": executionID,
				"workflow_id":  workflow.GetId(),
				"node_id":      step.NodeID,
				"step_index":   step.Index,
				"step_type":    step.Type,
				"error":        actionErr.Error(),
			}).Warn("Failed to build ActionDefinition for step")
		} else {
			instr.Action = action
		}

		instructions = append(instructions, instr)
	}

	// Build the contracts.ExecutionPlan with both flat and graph representations
	contractPlan := contracts.ExecutionPlan{
		SchemaVersion:  contracts.ExecutionPlanSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		Instructions:   instructions,
		Graph:          toContractsGraph(plan),
		Metadata:       plan.Metadata,
		CreatedAt:      time.Now().UTC(),
	}

	return contractPlan, instructions, nil
}

// toContractsGraph converts the compiler's ExecutionPlan into a contracts.PlanGraph.
// This preserves the graph structure (edges, loops) for control flow decisions.
func toContractsGraph(plan *ExecutionPlan) *contracts.PlanGraph {
	if plan == nil {
		return nil
	}

	steps := make([]contracts.PlanStep, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		// Convert outgoing edges
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
			Outgoing:  edges,
			Metadata:  map[string]string{},
			Context:   map[string]any{},
			Preload:   "",
			SourcePos: map[string]any{},
		}

		// Copy source position if available
		if step.SourcePosition != nil {
			converted.SourcePos["x"] = step.SourcePosition.X
			converted.SourcePos["y"] = step.SourcePosition.Y
		}

		// Recursively convert loop body if present
		if step.LoopPlan != nil {
			converted.Loop = toContractsGraph(step.LoopPlan)
		}

		// Build typed Action for type safety
		action, err := BuildActionDefinition(string(step.Type), step.Params)
		if err == nil {
			converted.Action = action
		}

		steps = append(steps, converted)
	}

	return &contracts.PlanGraph{Steps: steps}
}
