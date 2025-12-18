package compiler

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// ========================================================================
// Execution Plan Conversion (WorkflowNodeV2 -> CompiledInstruction)
// ========================================================================

// WorkflowNodeV2ToCompiledInstruction converts a WorkflowNodeV2 proto to a
// CompiledInstruction that can be executed by the automation engine.
func WorkflowNodeV2ToCompiledInstruction(node *basworkflows.WorkflowNodeV2, index int) contracts.CompiledInstruction {
	if node == nil {
		return contracts.CompiledInstruction{Index: index}
	}

	instruction := contracts.CompiledInstruction{
		Index:  index,
		NodeID: node.Id,
	}

	// Populate the typed Action field directly
	if node.Action != nil {
		instruction.Action = node.Action
	}

	// Add execution settings to context
	if node.ExecutionSettings != nil {
		instruction.Context = executionSettingsToContext(node.ExecutionSettings)
	}

	// Add metadata from action metadata
	if node.Action != nil && node.Action.Metadata != nil && node.Action.Metadata.Label != nil {
		instruction.Metadata = map[string]string{
			"label": *node.Action.Metadata.Label,
		}
	}

	return instruction
}

// WorkflowDefinitionV2ToExecutionPlan converts a complete V2 workflow definition
// to an execution plan with CompiledInstructions and PlanGraph.
func WorkflowDefinitionV2ToExecutionPlan(def *basworkflows.WorkflowDefinitionV2) ([]contracts.CompiledInstruction, *contracts.PlanGraph) {
	if def == nil {
		return nil, nil
	}

	// Convert nodes to instructions
	instructions := make([]contracts.CompiledInstruction, 0, len(def.Nodes))
	for i, node := range def.Nodes {
		instructions = append(instructions, WorkflowNodeV2ToCompiledInstruction(node, i))
	}

	// Build plan graph from edges
	planSteps := make([]contracts.PlanStep, 0, len(def.Nodes))
	for i, node := range def.Nodes {
		step := contracts.PlanStep{
			Index:    i,
			NodeID:   node.Id,
			Outgoing: findOutgoingEdges(node.Id, def.Edges),
		}
		// Populate typed Action field
		if node.Action != nil {
			step.Action = node.Action
		}
		// Copy execution settings to context
		if node.ExecutionSettings != nil {
			step.Context = executionSettingsToContext(node.ExecutionSettings)
		}
		planSteps = append(planSteps, step)
	}

	graph := &contracts.PlanGraph{
		Steps: planSteps,
	}

	return instructions, graph
}

// executionSettingsToContext converts NodeExecutionSettings to a context map.
func executionSettingsToContext(settings *basworkflows.NodeExecutionSettings) map[string]any {
	if settings == nil {
		return nil
	}

	ctx := make(map[string]any)

	if settings.TimeoutMs != nil {
		ctx["timeoutMs"] = int(*settings.TimeoutMs)
	}
	if settings.WaitAfterMs != nil {
		ctx["waitAfterMs"] = int(*settings.WaitAfterMs)
	}
	if settings.ContinueOnError != nil {
		ctx["continueOnError"] = *settings.ContinueOnError
	}

	if settings.Resilience != nil {
		res := make(map[string]any)
		if settings.Resilience.MaxAttempts != nil {
			res["maxAttempts"] = int(*settings.Resilience.MaxAttempts)
		}
		if settings.Resilience.DelayMs != nil {
			res["delayMs"] = int(*settings.Resilience.DelayMs)
		}
		if settings.Resilience.BackoffFactor != nil {
			res["backoffFactor"] = *settings.Resilience.BackoffFactor
		}
		if settings.Resilience.PreconditionSelector != nil {
			res["preconditionSelector"] = *settings.Resilience.PreconditionSelector
		}
		if settings.Resilience.PreconditionTimeoutMs != nil {
			res["preconditionTimeoutMs"] = int(*settings.Resilience.PreconditionTimeoutMs)
		}
		if settings.Resilience.SuccessSelector != nil {
			res["successSelector"] = *settings.Resilience.SuccessSelector
		}
		if settings.Resilience.SuccessTimeoutMs != nil {
			res["successTimeoutMs"] = int(*settings.Resilience.SuccessTimeoutMs)
		}

		if len(res) > 0 {
			ctx["resilience"] = res
		}
	}

	if len(ctx) == 0 {
		return nil
	}
	return ctx
}

// findOutgoingEdges finds the outgoing edges for a given source node.
func findOutgoingEdges(sourceID string, edges []*basworkflows.WorkflowEdgeV2) []contracts.PlanEdge {
	var outgoing []contracts.PlanEdge
	for _, edge := range edges {
		if edge.Source == sourceID {
			planEdge := contracts.PlanEdge{
				ID:     edge.Id,
				Target: edge.Target,
			}
			if edge.Label != nil {
				planEdge.Condition = *edge.Label
			}
			if edge.SourceHandle != nil {
				planEdge.SourcePort = *edge.SourceHandle
			}
			if edge.TargetHandle != nil {
				planEdge.TargetPort = *edge.TargetHandle
			}
			outgoing = append(outgoing, planEdge)
		}
	}
	return outgoing
}

// CompiledInstructionToWorkflowNodeV2 converts a CompiledInstruction back to
// a WorkflowNodeV2 proto for storage or serialization.
func CompiledInstructionToWorkflowNodeV2(instr contracts.CompiledInstruction) (*basworkflows.WorkflowNodeV2, error) {
	node := &basworkflows.WorkflowNodeV2{
		Id: instr.NodeID,
	}

	// Use the Action field directly
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		node.Action = instr.Action
	} else {
		return nil, fmt.Errorf("instruction %s has no Action defined", instr.NodeID)
	}

	// Add metadata from instruction (merge with existing action metadata)
	if instr.Metadata != nil {
		if label, ok := instr.Metadata["label"]; ok {
			if node.Action.Metadata == nil {
				node.Action.Metadata = &basactions.ActionMetadata{}
			}
			node.Action.Metadata.Label = &label
		}
	}

	// Convert context back to execution settings
	if instr.Context != nil {
		node.ExecutionSettings = contextToExecutionSettings(instr.Context)
	}

	return node, nil
}

// contextToExecutionSettings converts a context map back to NodeExecutionSettings.
func contextToExecutionSettings(ctx map[string]any) *basworkflows.NodeExecutionSettings {
	if ctx == nil {
		return nil
	}

	settings := &basworkflows.NodeExecutionSettings{}
	hasSettings := false

	if tm, ok := toInt32(ctx["timeoutMs"]); ok {
		settings.TimeoutMs = &tm
		hasSettings = true
	}
	if wa, ok := toInt32(ctx["waitAfterMs"]); ok {
		settings.WaitAfterMs = &wa
		hasSettings = true
	}
	if coe, ok := ctx["continueOnError"].(bool); ok {
		settings.ContinueOnError = &coe
		hasSettings = true
	}

	if resData, ok := ctx["resilience"].(map[string]any); ok {
		res := &basworkflows.ResilienceConfig{}
		hasResilience := false

		if ma, ok := toInt32(resData["maxAttempts"]); ok {
			res.MaxAttempts = &ma
			hasResilience = true
		}
		if dm, ok := toInt32(resData["delayMs"]); ok {
			res.DelayMs = &dm
			hasResilience = true
		}
		if bf, ok := toFloat64(resData["backoffFactor"]); ok {
			res.BackoffFactor = &bf
			hasResilience = true
		}
		if ps, ok := resData["preconditionSelector"].(string); ok {
			res.PreconditionSelector = &ps
			hasResilience = true
		}
		if ptm, ok := toInt32(resData["preconditionTimeoutMs"]); ok {
			res.PreconditionTimeoutMs = &ptm
			hasResilience = true
		}
		if ss, ok := resData["successSelector"].(string); ok {
			res.SuccessSelector = &ss
			hasResilience = true
		}
		if stm, ok := toInt32(resData["successTimeoutMs"]); ok {
			res.SuccessTimeoutMs = &stm
			hasResilience = true
		}

		if hasResilience {
			settings.Resilience = res
			hasSettings = true
		}
	}

	if !hasSettings {
		return nil
	}
	return settings
}
