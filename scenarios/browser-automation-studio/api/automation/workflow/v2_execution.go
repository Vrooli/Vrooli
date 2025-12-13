package workflow

import (
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
)

// ========================================================================
// Execution Plan Conversion (WorkflowNodeV2 -> CompiledInstruction)
// ========================================================================

// WorkflowNodeV2ToCompiledInstruction converts a WorkflowNodeV2 proto to a
// CompiledInstruction that can be executed by the automation engine.
func WorkflowNodeV2ToCompiledInstruction(node *basv1.WorkflowNodeV2, index int) contracts.CompiledInstruction {
	if node == nil {
		return contracts.CompiledInstruction{Index: index}
	}

	instruction := contracts.CompiledInstruction{
		Index:  index,
		NodeID: node.Id,
		Params: make(map[string]any),
	}

	// Extract type and params from ActionDefinition
	if node.Action != nil {
		instruction.Type = mapActionTypeToV1Type(node.Action.Type)
		instruction.Params = actionDefinitionToParams(node.Action)
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
func WorkflowDefinitionV2ToExecutionPlan(def *basv1.WorkflowDefinitionV2) ([]contracts.CompiledInstruction, *contracts.PlanGraph) {
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
			Type:     mapActionTypeToV1Type(node.Action.Type),
			Params:   actionDefinitionToParams(node.Action),
			Outgoing: findOutgoingEdges(node.Id, def.Edges),
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

// actionDefinitionToParams extracts params from an ActionDefinition to a map.
func actionDefinitionToParams(action *basv1.ActionDefinition) map[string]any {
	if action == nil {
		return make(map[string]any)
	}

	params := make(map[string]any)

	switch p := action.Params.(type) {
	case *basv1.ActionDefinition_Navigate:
		if p.Navigate != nil {
			params["url"] = p.Navigate.Url
			if p.Navigate.WaitForSelector != nil {
				params["waitForSelector"] = *p.Navigate.WaitForSelector
			}
			if p.Navigate.TimeoutMs != nil {
				params["timeoutMs"] = *p.Navigate.TimeoutMs
			}
			if p.Navigate.WaitUntil != nil {
				params["waitUntil"] = *p.Navigate.WaitUntil
			}
		}
	case *basv1.ActionDefinition_Click:
		if p.Click != nil {
			params["selector"] = p.Click.Selector
			if p.Click.Button != nil {
				params["button"] = *p.Click.Button
			}
			if p.Click.ClickCount != nil {
				params["clickCount"] = int(*p.Click.ClickCount)
			}
			if p.Click.DelayMs != nil {
				params["delayMs"] = int(*p.Click.DelayMs)
			}
			if len(p.Click.Modifiers) > 0 {
				params["modifiers"] = p.Click.Modifiers
			}
			if p.Click.Force != nil {
				params["force"] = *p.Click.Force
			}
		}
	case *basv1.ActionDefinition_Input:
		if p.Input != nil {
			params["selector"] = p.Input.Selector
			params["value"] = p.Input.Value
			if p.Input.IsSensitive != nil {
				params["isSensitive"] = *p.Input.IsSensitive
			}
			if p.Input.Submit != nil {
				params["submit"] = *p.Input.Submit
			}
			if p.Input.ClearFirst != nil {
				params["clearFirst"] = *p.Input.ClearFirst
			}
			if p.Input.DelayMs != nil {
				params["delayMs"] = int(*p.Input.DelayMs)
			}
		}
	case *basv1.ActionDefinition_Wait:
		if p.Wait != nil {
			switch w := p.Wait.WaitFor.(type) {
			case *basv1.WaitParams_DurationMs:
				params["durationMs"] = int(w.DurationMs)
			case *basv1.WaitParams_Selector:
				params["selector"] = w.Selector
			}
			if p.Wait.State != nil {
				params["state"] = *p.Wait.State
			}
			if p.Wait.TimeoutMs != nil {
				params["timeoutMs"] = int(*p.Wait.TimeoutMs)
			}
		}
	case *basv1.ActionDefinition_Assert:
		if p.Assert != nil {
			params["selector"] = p.Assert.Selector
			params["mode"] = p.Assert.Mode
			if p.Assert.Expected != nil {
				params["expected"] = jsonValueToAny(p.Assert.Expected)
			}
			if p.Assert.Negated != nil {
				params["negated"] = *p.Assert.Negated
			}
			if p.Assert.CaseSensitive != nil {
				params["caseSensitive"] = *p.Assert.CaseSensitive
			}
			if p.Assert.AttributeName != nil {
				params["attributeName"] = *p.Assert.AttributeName
			}
		}
	case *basv1.ActionDefinition_Scroll:
		if p.Scroll != nil {
			if p.Scroll.Selector != nil {
				params["selector"] = *p.Scroll.Selector
			}
			if p.Scroll.X != nil {
				params["x"] = int(*p.Scroll.X)
			}
			if p.Scroll.Y != nil {
				params["y"] = int(*p.Scroll.Y)
			}
			if p.Scroll.DeltaX != nil {
				params["deltaX"] = int(*p.Scroll.DeltaX)
			}
			if p.Scroll.DeltaY != nil {
				params["deltaY"] = int(*p.Scroll.DeltaY)
			}
			if p.Scroll.Behavior != nil {
				params["behavior"] = *p.Scroll.Behavior
			}
		}
	case *basv1.ActionDefinition_SelectOption:
		if p.SelectOption != nil {
			params["selector"] = p.SelectOption.Selector
			switch s := p.SelectOption.SelectBy.(type) {
			case *basv1.SelectParams_Value:
				params["value"] = s.Value
			case *basv1.SelectParams_Label:
				params["label"] = s.Label
			case *basv1.SelectParams_Index:
				params["index"] = int(s.Index)
			}
			if p.SelectOption.TimeoutMs != nil {
				params["timeoutMs"] = int(*p.SelectOption.TimeoutMs)
			}
		}
	case *basv1.ActionDefinition_Evaluate:
		if p.Evaluate != nil {
			params["expression"] = p.Evaluate.Expression
			if p.Evaluate.StoreResult != nil {
				params["storeResult"] = *p.Evaluate.StoreResult
			}
		}
	case *basv1.ActionDefinition_Keyboard:
		if p.Keyboard != nil {
			if p.Keyboard.Key != nil {
				params["key"] = *p.Keyboard.Key
			}
			if len(p.Keyboard.Keys) > 0 {
				params["keys"] = p.Keyboard.Keys
			}
			if len(p.Keyboard.Modifiers) > 0 {
				params["modifiers"] = p.Keyboard.Modifiers
			}
			if p.Keyboard.Action != nil {
				params["action"] = *p.Keyboard.Action
			}
		}
	case *basv1.ActionDefinition_Hover:
		if p.Hover != nil {
			params["selector"] = p.Hover.Selector
			if p.Hover.TimeoutMs != nil {
				params["timeoutMs"] = int(*p.Hover.TimeoutMs)
			}
		}
	case *basv1.ActionDefinition_Screenshot:
		if p.Screenshot != nil {
			if p.Screenshot.FullPage != nil {
				params["fullPage"] = *p.Screenshot.FullPage
			}
			if p.Screenshot.Selector != nil {
				params["selector"] = *p.Screenshot.Selector
			}
			if p.Screenshot.Quality != nil {
				params["quality"] = int(*p.Screenshot.Quality)
			}
		}
	case *basv1.ActionDefinition_Focus:
		if p.Focus != nil {
			params["selector"] = p.Focus.Selector
			if p.Focus.TimeoutMs != nil {
				params["timeoutMs"] = int(*p.Focus.TimeoutMs)
			}
		}
	case *basv1.ActionDefinition_Blur:
		if p.Blur != nil {
			if p.Blur.Selector != nil {
				params["selector"] = *p.Blur.Selector
			}
			if p.Blur.TimeoutMs != nil {
				params["timeoutMs"] = int(*p.Blur.TimeoutMs)
			}
		}
	}

	return params
}

// executionSettingsToContext converts NodeExecutionSettings to a context map.
func executionSettingsToContext(settings *basv1.NodeExecutionSettings) map[string]any {
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
func findOutgoingEdges(sourceID string, edges []*basv1.WorkflowEdgeV2) []contracts.PlanEdge {
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
func CompiledInstructionToWorkflowNodeV2(instr contracts.CompiledInstruction) (*basv1.WorkflowNodeV2, error) {
	node := &basv1.WorkflowNodeV2{
		Id: instr.NodeID,
	}

	// Build ActionDefinition from type and params
	action, err := v1DataToActionDefinition(instr.Type, instr.Params)
	if err != nil {
		return nil, err
	}
	node.Action = action

	// Add metadata from instruction
	if instr.Metadata != nil {
		if label, ok := instr.Metadata["label"]; ok {
			if action.Metadata == nil {
				action.Metadata = &basv1.ActionMetadata{}
			}
			action.Metadata.Label = &label
		}
	}

	// Convert context back to execution settings
	if instr.Context != nil {
		node.ExecutionSettings = contextToExecutionSettings(instr.Context)
	}

	return node, nil
}

// contextToExecutionSettings converts a context map back to NodeExecutionSettings.
func contextToExecutionSettings(ctx map[string]any) *basv1.NodeExecutionSettings {
	if ctx == nil {
		return nil
	}

	settings := &basv1.NodeExecutionSettings{}
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
		res := &basv1.ResilienceConfig{}
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
