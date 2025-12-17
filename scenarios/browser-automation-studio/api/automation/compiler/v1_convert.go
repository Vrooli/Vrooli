// Package compiler provides workflow compilation and format conversion utilities.
package compiler

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// V1NodeToWorkflowNodeV2 converts a legacy V1 node to the proto WorkflowNodeV2 format.
func V1NodeToWorkflowNodeV2(node V1Node) (*basworkflows.WorkflowNodeV2, error) {
	action, err := V1DataToActionDefinition(node.Type, node.Data)
	if err != nil {
		return nil, fmt.Errorf("convert node %s: %w", node.ID, err)
	}

	result := &basworkflows.WorkflowNodeV2{
		Id:     node.ID,
		Action: action,
	}

	if node.Position != nil {
		result.Position = &basbase.NodePosition{
			X: node.Position.X,
			Y: node.Position.Y,
		}
	}

	// Extract execution settings if present
	if execSettings := extractExecutionSettings(node.Data); execSettings != nil {
		result.ExecutionSettings = execSettings
	}

	return result, nil
}


// V1EdgeToWorkflowEdgeV2 converts a legacy V1 edge to the proto format.
func V1EdgeToWorkflowEdgeV2(edge V1Edge) *basworkflows.WorkflowEdgeV2 {
	result := &basworkflows.WorkflowEdgeV2{
		Id:     edge.ID,
		Source: edge.Source,
		Target: edge.Target,
	}

	if edge.Type != "" {
		edgeType := stringToWorkflowEdgeType(edge.Type)
		result.Type = &edgeType
	}
	if edge.Label != "" {
		result.Label = &edge.Label
	}
	if edge.SourceHandle != "" {
		result.SourceHandle = &edge.SourceHandle
	}
	if edge.TargetHandle != "" {
		result.TargetHandle = &edge.TargetHandle
	}

	return result
}


// V1FlowDefinitionToV2 converts a complete V1 flow definition to V2 format.
func V1FlowDefinitionToV2(def V1FlowDefinition) (*basworkflows.WorkflowDefinitionV2, error) {
	result := &basworkflows.WorkflowDefinitionV2{
		Metadata: extractV2Metadata(def.Metadata),
		Settings: extractV2Settings(def.Settings),
		Nodes:    make([]*basworkflows.WorkflowNodeV2, 0, len(def.Nodes)),
		Edges:    make([]*basworkflows.WorkflowEdgeV2, 0, len(def.Edges)),
	}

	for _, node := range def.Nodes {
		v2Node, err := V1NodeToWorkflowNodeV2(node)
		if err != nil {
			return nil, err
		}
		result.Nodes = append(result.Nodes, v2Node)
	}

	for _, edge := range def.Edges {
		result.Edges = append(result.Edges, V1EdgeToWorkflowEdgeV2(edge))
	}

	return result, nil
}


// V1DataToActionDefinition converts V1 node data to an ActionDefinition.
// This is the bridge between legacy untyped params and new typed action.
// Used by ContractPlanCompiler to populate CompiledInstruction.Action.
func V1DataToActionDefinition(nodeType string, data map[string]any) (*basactions.ActionDefinition, error) {
	action := &basactions.ActionDefinition{}

	// Map node type to ActionType
	actionType := mapV1TypeToActionType(nodeType)
	action.Type = actionType

	// Build params based on type
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		action.Params = &basactions.ActionDefinition_Navigate{
			Navigate: buildNavigateParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		action.Params = &basactions.ActionDefinition_Click{
			Click: buildClickParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		action.Params = &basactions.ActionDefinition_Input{
			Input: buildInputParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		action.Params = &basactions.ActionDefinition_Wait{
			Wait: buildWaitParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		action.Params = &basactions.ActionDefinition_Assert{
			Assert: buildAssertParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		action.Params = &basactions.ActionDefinition_Scroll{
			Scroll: buildScrollParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		action.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: buildSelectParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		action.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: buildEvaluateParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		action.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: buildKeyboardParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		action.Params = &basactions.ActionDefinition_Hover{
			Hover: buildHoverParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		action.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: buildScreenshotParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		action.Params = &basactions.ActionDefinition_Focus{
			Focus: buildFocusParams(data),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		action.Params = &basactions.ActionDefinition_Blur{
			Blur: buildBlurParams(data),
		}
	}

	// Build metadata if recording data is present
	action.Metadata = buildActionMetadata(data)

	return action, nil
}


// mapV1TypeToActionType converts V1 node type string to ActionType enum.
// Delegates to typeconv.StringToActionType for the canonical implementation.
func mapV1TypeToActionType(nodeType string) basactions.ActionType {
	return typeconv.StringToActionType(nodeType)
}

// mapActionTypeToV1Type converts ActionType enum to V1 node type string.
// Delegates to typeconv.ActionTypeToString for the canonical implementation.
func mapActionTypeToV1Type(actionType basactions.ActionType) string {
	return typeconv.ActionTypeToString(actionType)
}

// extractExecutionSettings extracts NodeExecutionSettings from V1 node data.
func extractExecutionSettings(data map[string]any) *basworkflows.NodeExecutionSettings {
	settings := &basworkflows.NodeExecutionSettings{}
	hasSettings := false

	if tm, ok := toInt32(data["timeoutMs"]); ok {
		settings.TimeoutMs = &tm
		hasSettings = true
	}
	if wa, ok := toInt32(data["waitAfterMs"]); ok {
		settings.WaitAfterMs = &wa
		hasSettings = true
	}
	if coe, ok := data["continueOnError"].(bool); ok {
		settings.ContinueOnError = &coe
		hasSettings = true
	}

	// Extract resilience config
	if resData, ok := data["resilience"].(map[string]any); ok {
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
		if ss, ok := resData["successSelector"].(string); ok {
			res.SuccessSelector = &ss
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


// extractV2Metadata extracts V2 metadata from V1 metadata map.
func extractV2Metadata(v1Meta map[string]any) *basworkflows.WorkflowMetadataV2 {
	if v1Meta == nil {
		return nil
	}
	meta := &basworkflows.WorkflowMetadataV2{
		Labels: make(map[string]string),
	}
	hasData := false

	if name, ok := v1Meta["name"].(string); ok {
		meta.Name = &name
		hasData = true
	}
	if desc, ok := v1Meta["description"].(string); ok {
		meta.Description = &desc
		hasData = true
	}
	if ver, ok := v1Meta["version"].(string); ok {
		meta.Version = &ver
		hasData = true
	}
	if labels, ok := v1Meta["labels"].(map[string]any); ok {
		for k, v := range labels {
			if s, ok := v.(string); ok {
				meta.Labels[k] = s
				hasData = true
			}
		}
	}

	if !hasData {
		return nil
	}
	return meta
}

// extractV2Settings extracts V2 settings from V1 settings map.
func extractV2Settings(v1Settings map[string]any) *basworkflows.WorkflowSettingsV2 {
	if v1Settings == nil {
		return nil
	}
	settings := &basworkflows.WorkflowSettingsV2{}
	hasData := false

	// Try executionViewport first, then viewport_width/viewport_height
	if viewport, ok := v1Settings["executionViewport"].(map[string]any); ok {
		if w, ok := toInt32(viewport["width"]); ok {
			settings.ViewportWidth = &w
			hasData = true
		}
		if h, ok := toInt32(viewport["height"]); ok {
			settings.ViewportHeight = &h
			hasData = true
		}
	}
	if w, ok := toInt32(v1Settings["viewport_width"]); ok {
		settings.ViewportWidth = &w
		hasData = true
	}
	if h, ok := toInt32(v1Settings["viewport_height"]); ok {
		settings.ViewportHeight = &h
		hasData = true
	}

	if ua, ok := v1Settings["user_agent"].(string); ok {
		settings.UserAgent = &ua
		hasData = true
	}
	if locale, ok := v1Settings["locale"].(string); ok {
		settings.Locale = &locale
		hasData = true
	}
	// Handle both legacy timeout_seconds and new timeout_ms
	if ts, ok := toInt32(v1Settings["timeout_seconds"]); ok {
		// Convert seconds to milliseconds for the proto field
		timeoutMs := ts * 1000
		settings.TimeoutMs = &timeoutMs
		hasData = true
	} else if tm, ok := toInt32(v1Settings["timeout_ms"]); ok {
		settings.TimeoutMs = &tm
		hasData = true
	}
	if headless, ok := v1Settings["headless"].(bool); ok {
		settings.Headless = &headless
		hasData = true
	}
	if es, ok := v1Settings["entrySelector"].(string); ok {
		settings.EntrySelector = &es
		hasData = true
	}
	if estm, ok := toInt32(v1Settings["entrySelectorTimeoutMs"]); ok {
		settings.EntrySelectorTimeoutMs = &estm
		hasData = true
	}

	if !hasData {
		return nil
	}
	return settings
}



// stringToWorkflowEdgeType converts a string edge type to the proto enum.
func stringToWorkflowEdgeType(s string) basbase.WorkflowEdgeType {
	switch s {
	case "default":
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_DEFAULT
	case "smoothstep":
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_SMOOTHSTEP
	case "step":
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_STEP
	case "straight":
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_STRAIGHT
	case "bezier":
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_BEZIER
	default:
		return basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_UNSPECIFIED
	}
}

