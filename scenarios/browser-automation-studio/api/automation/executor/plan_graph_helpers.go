package executor

import (
	"log"

	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	autoworkflow "github.com/vrooli/browser-automation-studio/automation/workflow"
)

// toContractsGraphFromAutomation converts the automation compiler plan into the contract graph shape.
func toContractsGraphFromAutomation(plan *autocompiler.ExecutionPlan) *contracts.PlanGraph {
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
			SourcePos: map[string]any{},
		}

		// MIGRATION: Populate typed Action field from V1 type/params
		action, actionErr := autoworkflow.V1DataToActionDefinition(string(step.Type), step.Params)
		if actionErr != nil {
			log.Printf("[GRAPH] Warning: failed to build ActionDefinition for step %d (%s): %v", step.Index, step.Type, actionErr)
		} else {
			converted.Action = action
		}

		if step.SourcePosition != nil {
			converted.SourcePos["x"] = step.SourcePosition.X
			converted.SourcePos["y"] = step.SourcePosition.Y
		}
		if step.LoopPlan != nil {
			converted.Loop = toContractsGraphFromAutomation(step.LoopPlan)
		}
		steps = append(steps, converted)
	}
	return &contracts.PlanGraph{Steps: steps}
}
