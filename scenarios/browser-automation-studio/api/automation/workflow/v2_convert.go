// Package workflow provides conversion utilities between workflow formats.
package workflow

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// V1NodeToWorkflowNodeV2 converts a legacy V1 node to the proto WorkflowNodeV2 format.
func V1NodeToWorkflowNodeV2(node V1Node) (*basworkflows.WorkflowNodeV2, error) {
	action, err := v1DataToActionDefinition(node.Type, node.Data)
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

// WorkflowNodeV2ToV1Node converts a proto WorkflowNodeV2 to the legacy V1 format.
func WorkflowNodeV2ToV1Node(node *basworkflows.WorkflowNodeV2) (V1Node, error) {
	if node == nil {
		return V1Node{}, fmt.Errorf("node is nil")
	}

	nodeType, data := actionDefinitionToV1Data(node.Action)

	// Merge execution settings into data
	if node.ExecutionSettings != nil {
		mergeExecutionSettingsToData(node.ExecutionSettings, data)
	}

	result := V1Node{
		ID:   node.Id,
		Type: nodeType,
		Data: data,
	}

	if node.Position != nil {
		result.Position = &V1Position{
			X: node.Position.X,
			Y: node.Position.Y,
		}
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

// WorkflowEdgeV2ToV1Edge converts a proto edge to the legacy V1 format.
func WorkflowEdgeV2ToV1Edge(edge *basworkflows.WorkflowEdgeV2) V1Edge {
	if edge == nil {
		return V1Edge{}
	}

	result := V1Edge{
		ID:     edge.Id,
		Source: edge.Source,
		Target: edge.Target,
	}

	if edge.Type != nil {
		result.Type = workflowEdgeTypeToString(*edge.Type)
	}
	if edge.Label != nil {
		result.Label = *edge.Label
	}
	if edge.SourceHandle != nil {
		result.SourceHandle = *edge.SourceHandle
	}
	if edge.TargetHandle != nil {
		result.TargetHandle = *edge.TargetHandle
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

// WorkflowDefinitionV2ToV1 converts a V2 definition to the legacy V1 format.
func WorkflowDefinitionV2ToV1(def *basworkflows.WorkflowDefinitionV2) (V1FlowDefinition, error) {
	if def == nil {
		return V1FlowDefinition{}, fmt.Errorf("definition is nil")
	}

	result := V1FlowDefinition{
		Nodes:    make([]V1Node, 0, len(def.Nodes)),
		Edges:    make([]V1Edge, 0, len(def.Edges)),
		Settings: v2SettingsToMap(def.Settings),
		Metadata: v2MetadataToMap(def.Metadata),
	}

	for _, node := range def.Nodes {
		v1Node, err := WorkflowNodeV2ToV1Node(node)
		if err != nil {
			return V1FlowDefinition{}, err
		}
		result.Nodes = append(result.Nodes, v1Node)
	}

	for _, edge := range def.Edges {
		result.Edges = append(result.Edges, WorkflowEdgeV2ToV1Edge(edge))
	}

	return result, nil
}

// v1DataToActionDefinition converts V1 node data to an ActionDefinition.
func v1DataToActionDefinition(nodeType string, data map[string]any) (*basactions.ActionDefinition, error) {
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

// actionDefinitionToV1Data converts an ActionDefinition to V1 node type and data.
func actionDefinitionToV1Data(action *basactions.ActionDefinition) (string, map[string]any) {
	if action == nil {
		return "unknown", map[string]any{}
	}

	nodeType := mapActionTypeToV1Type(action.Type)
	data := make(map[string]any)

	// Extract params based on type
	switch p := action.Params.(type) {
	case *basactions.ActionDefinition_Navigate:
		if p.Navigate != nil {
			data["url"] = p.Navigate.Url
			if p.Navigate.WaitForSelector != nil {
				data["waitForSelector"] = *p.Navigate.WaitForSelector
			}
			if p.Navigate.TimeoutMs != nil {
				data["timeoutMs"] = *p.Navigate.TimeoutMs
			}
			if p.Navigate.WaitUntil != nil {
				data["waitUntil"] = *p.Navigate.WaitUntil
			}
		}
	case *basactions.ActionDefinition_Click:
		if p.Click != nil {
			data["selector"] = p.Click.Selector
			if p.Click.Button != nil {
				data["button"] = *p.Click.Button
			}
			if p.Click.ClickCount != nil {
				data["clickCount"] = *p.Click.ClickCount
			}
			if p.Click.DelayMs != nil {
				data["delayMs"] = *p.Click.DelayMs
			}
			if len(p.Click.Modifiers) > 0 {
				data["modifiers"] = p.Click.Modifiers
			}
			if p.Click.Force != nil {
				data["force"] = *p.Click.Force
			}
		}
	case *basactions.ActionDefinition_Input:
		if p.Input != nil {
			data["selector"] = p.Input.Selector
			data["value"] = p.Input.Value
			if p.Input.IsSensitive != nil {
				data["isSensitive"] = *p.Input.IsSensitive
			}
			if p.Input.Submit != nil {
				data["submit"] = *p.Input.Submit
			}
			if p.Input.ClearFirst != nil {
				data["clearFirst"] = *p.Input.ClearFirst
			}
		}
	case *basactions.ActionDefinition_Wait:
		if p.Wait != nil {
			switch w := p.Wait.WaitFor.(type) {
			case *basactions.WaitParams_DurationMs:
				data["durationMs"] = w.DurationMs
			case *basactions.WaitParams_Selector:
				data["selector"] = w.Selector
			}
			if p.Wait.State != nil {
				data["state"] = *p.Wait.State
			}
			if p.Wait.TimeoutMs != nil {
				data["timeoutMs"] = *p.Wait.TimeoutMs
			}
		}
	case *basactions.ActionDefinition_Assert:
		if p.Assert != nil {
			data["selector"] = p.Assert.Selector
			data["mode"] = p.Assert.Mode
			data["assertMode"] = p.Assert.Mode // Legacy field
			if p.Assert.Expected != nil {
				data["expected"] = jsonValueToAny(p.Assert.Expected)
			}
			if p.Assert.Negated != nil {
				data["negated"] = *p.Assert.Negated
			}
		}
	case *basactions.ActionDefinition_Scroll:
		if p.Scroll != nil {
			if p.Scroll.Selector != nil {
				data["selector"] = *p.Scroll.Selector
			}
			if p.Scroll.X != nil {
				data["x"] = *p.Scroll.X
			}
			if p.Scroll.Y != nil {
				data["y"] = *p.Scroll.Y
			}
			if p.Scroll.DeltaX != nil {
				data["deltaX"] = *p.Scroll.DeltaX
			}
			if p.Scroll.DeltaY != nil {
				data["deltaY"] = *p.Scroll.DeltaY
			}
		}
	case *basactions.ActionDefinition_SelectOption:
		if p.SelectOption != nil {
			data["selector"] = p.SelectOption.Selector
			switch s := p.SelectOption.SelectBy.(type) {
			case *basactions.SelectParams_Value:
				data["value"] = s.Value
			case *basactions.SelectParams_Label:
				data["label"] = s.Label
			case *basactions.SelectParams_Index:
				data["index"] = s.Index
			}
		}
	case *basactions.ActionDefinition_Evaluate:
		if p.Evaluate != nil {
			data["expression"] = p.Evaluate.Expression
			if p.Evaluate.StoreResult != nil {
				data["storeResult"] = *p.Evaluate.StoreResult
			}
		}
	case *basactions.ActionDefinition_Keyboard:
		if p.Keyboard != nil {
			if p.Keyboard.Key != nil {
				data["key"] = *p.Keyboard.Key
			}
			if len(p.Keyboard.Keys) > 0 {
				data["keys"] = p.Keyboard.Keys
			}
			if len(p.Keyboard.Modifiers) > 0 {
				data["modifiers"] = p.Keyboard.Modifiers
			}
			if p.Keyboard.Action != nil {
				data["action"] = *p.Keyboard.Action
			}
		}
	case *basactions.ActionDefinition_Hover:
		if p.Hover != nil {
			data["selector"] = p.Hover.Selector
			if p.Hover.TimeoutMs != nil {
				data["timeoutMs"] = *p.Hover.TimeoutMs
			}
		}
	case *basactions.ActionDefinition_Screenshot:
		if p.Screenshot != nil {
			if p.Screenshot.FullPage != nil {
				data["fullPage"] = *p.Screenshot.FullPage
			}
			if p.Screenshot.Selector != nil {
				data["selector"] = *p.Screenshot.Selector
			}
			if p.Screenshot.Quality != nil {
				data["quality"] = *p.Screenshot.Quality
			}
		}
	case *basactions.ActionDefinition_Focus:
		if p.Focus != nil {
			data["selector"] = p.Focus.Selector
		}
	case *basactions.ActionDefinition_Blur:
		if p.Blur != nil && p.Blur.Selector != nil {
			data["selector"] = *p.Blur.Selector
		}
	}

	return nodeType, data
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

// mergeExecutionSettingsToData merges NodeExecutionSettings into V1 node data.
func mergeExecutionSettingsToData(settings *basworkflows.NodeExecutionSettings, data map[string]any) {
	if settings == nil {
		return
	}
	if settings.TimeoutMs != nil {
		data["timeoutMs"] = *settings.TimeoutMs
	}
	if settings.WaitAfterMs != nil {
		data["waitAfterMs"] = *settings.WaitAfterMs
	}
	if settings.ContinueOnError != nil {
		data["continueOnError"] = *settings.ContinueOnError
	}
	if settings.Resilience != nil {
		res := make(map[string]any)
		if settings.Resilience.MaxAttempts != nil {
			res["maxAttempts"] = *settings.Resilience.MaxAttempts
		}
		if settings.Resilience.DelayMs != nil {
			res["delayMs"] = *settings.Resilience.DelayMs
		}
		if settings.Resilience.BackoffFactor != nil {
			res["backoffFactor"] = *settings.Resilience.BackoffFactor
		}
		if settings.Resilience.PreconditionSelector != nil {
			res["preconditionSelector"] = *settings.Resilience.PreconditionSelector
		}
		if settings.Resilience.SuccessSelector != nil {
			res["successSelector"] = *settings.Resilience.SuccessSelector
		}
		data["resilience"] = res
	}
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

// v2MetadataToMap converts V2 metadata to a V1 map.
func v2MetadataToMap(meta *basworkflows.WorkflowMetadataV2) map[string]any {
	if meta == nil {
		return nil
	}
	result := make(map[string]any)
	if meta.Name != nil {
		result["name"] = *meta.Name
	}
	if meta.Description != nil {
		result["description"] = *meta.Description
	}
	if meta.Version != nil {
		result["version"] = *meta.Version
	}
	if len(meta.Labels) > 0 {
		result["labels"] = meta.Labels
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// v2SettingsToMap converts V2 settings to a V1 map.
func v2SettingsToMap(settings *basworkflows.WorkflowSettingsV2) map[string]any {
	if settings == nil {
		return nil
	}
	result := make(map[string]any)
	if settings.ViewportWidth != nil || settings.ViewportHeight != nil {
		viewport := make(map[string]any)
		if settings.ViewportWidth != nil {
			viewport["width"] = *settings.ViewportWidth
		}
		if settings.ViewportHeight != nil {
			viewport["height"] = *settings.ViewportHeight
		}
		result["executionViewport"] = viewport
	}
	if settings.UserAgent != nil {
		result["user_agent"] = *settings.UserAgent
	}
	if settings.Locale != nil {
		result["locale"] = *settings.Locale
	}
	if settings.TimeoutMs != nil {
		// Convert milliseconds back to seconds for legacy format
		result["timeout_seconds"] = *settings.TimeoutMs / 1000
	}
	if settings.Headless != nil {
		result["headless"] = *settings.Headless
	}
	if settings.EntrySelector != nil {
		result["entrySelector"] = *settings.EntrySelector
	}
	if settings.EntrySelectorTimeoutMs != nil {
		result["entrySelectorTimeoutMs"] = *settings.EntrySelectorTimeoutMs
	}
	if len(result) == 0 {
		return nil
	}
	return result
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

// workflowEdgeTypeToString converts the proto enum to a string edge type.
func workflowEdgeTypeToString(t basbase.WorkflowEdgeType) string {
	switch t {
	case basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_DEFAULT:
		return "default"
	case basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_SMOOTHSTEP:
		return "smoothstep"
	case basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_STEP:
		return "step"
	case basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_STRAIGHT:
		return "straight"
	case basbase.WorkflowEdgeType_WORKFLOW_EDGE_TYPE_BEZIER:
		return "bezier"
	default:
		return ""
	}
}
