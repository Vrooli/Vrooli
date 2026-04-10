package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// ErrInvalidWorkflowFormat is returned when a workflow is missing required V2 structure.
// All workflows must use V2 format where nodes have an "action" field with typed action definitions.
var ErrInvalidWorkflowFormat = errors.New("invalid workflow format: nodes must have 'action' field with typed action definitions")

// BuildFlowDefinitionV2ForWrite converts an incoming flow definition map to a WorkflowDefinitionV2.
// Validates that the workflow uses V2 format (nodes with "action" field) and normalizes
// UI-oriented fields for proto compatibility.
func BuildFlowDefinitionV2ForWrite(flow map[string]any, metadata map[string]any, settings map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	if flow == nil {
		return &basworkflows.WorkflowDefinitionV2{}, nil
	}

	// Merge metadata and settings into the flow if provided separately.
	merged := make(map[string]any, len(flow))
	for k, v := range flow {
		merged[k] = v
	}
	if metadata != nil && merged["metadata"] == nil {
		merged["metadata"] = metadata
	}
	if settings != nil && merged["settings"] == nil {
		merged["settings"] = settings
	}

	// Validate V2 format - all nodes must have action field.
	if !isV2Format(merged) {
		return nil, ErrInvalidWorkflowFormat
	}

	// Normalize for proto serialization.
	normalizeForProto(merged)
	body, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal flow definition: %w", err)
	}

	var pb basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &pb); err != nil {
		return nil, fmt.Errorf("parse V2 flow definition: %w", err)
	}

	return &pb, nil
}

// validateFlowDefinitionV2OnWrite validates a WorkflowDefinitionV2 proto message.
// Returns an error if the definition is invalid for writing.
func validateFlowDefinitionV2OnWrite(def *basworkflows.WorkflowDefinitionV2) error {
	if def == nil {
		return nil // Empty definition is valid.
	}

	// Validate nodes have required fields.
	for i, node := range def.Nodes {
		if node == nil {
			return fmt.Errorf("node at index %d is nil", i)
		}
		if node.Id == "" {
			return fmt.Errorf("node at index %d has empty id", i)
		}
		if node.Action == nil {
			return fmt.Errorf("node %q has no action defined", node.Id)
		}
	}

	// Validate edges reference valid nodes.
	nodeIDs := make(map[string]bool, len(def.Nodes))
	for _, node := range def.Nodes {
		nodeIDs[node.Id] = true
	}

	for i, edge := range def.Edges {
		if edge == nil {
			return fmt.Errorf("edge at index %d is nil", i)
		}
		if edge.Id == "" {
			return fmt.Errorf("edge at index %d has empty id", i)
		}
		if edge.Source == "" {
			return fmt.Errorf("edge %q has empty source", edge.Id)
		}
		if edge.Target == "" {
			return fmt.Errorf("edge %q has empty target", edge.Id)
		}
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("edge %q references unknown source node %q", edge.Id, edge.Source)
		}
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("edge %q references unknown target node %q", edge.Id, edge.Target)
		}
	}

	return nil
}

// isV2Format validates that a flow definition uses V2 format.
// V2 format requires nodes to have "action" fields with typed action definitions.
// Returns true for empty workflows (valid V2) or workflows with action fields.
func isV2Format(doc map[string]any) bool {
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return true // Empty workflow is valid V2
	}
	first, ok := nodes[0].(map[string]any)
	if !ok || first == nil {
		return false
	}
	_, hasAction := first["action"]
	return hasAction
}

// normalizeForProto normalizes a V2 flow definition for proto serialization.
// Handles:
//   - UI-oriented viewport settings -> proto field names
//   - Primitive subflow args -> JsonValue wrappers
func normalizeForProto(doc map[string]any) {
	settings, ok := doc["settings"].(map[string]any)
	if ok && settings != nil {
		// Translate UI-oriented viewport settings to proto field names.
		if viewport, ok := settings["executionViewport"].(map[string]any); ok && viewport != nil {
			if width, ok := viewport["width"]; ok {
				switch v := width.(type) {
				case float64:
					settings["viewport_width"] = int32(v)
				case int:
					settings["viewport_width"] = int32(v)
				case int32:
					settings["viewport_width"] = v
				case int64:
					settings["viewport_width"] = int32(v)
				}
			}
			if height, ok := viewport["height"]; ok {
				switch v := height.(type) {
				case float64:
					settings["viewport_height"] = int32(v)
				case int:
					settings["viewport_height"] = int32(v)
				case int32:
					settings["viewport_height"] = v
				case int64:
					settings["viewport_height"] = int32(v)
				}
			}
			delete(settings, "executionViewport")
		}
		delete(settings, "defaultStepTimeoutMs")
	}

	// Process nodes: wrap subflow args and remove non-proto fields.
	nodes, ok := doc["nodes"].([]any)
	if ok && len(nodes) > 0 {
		for _, rawNode := range nodes {
			node, ok := rawNode.(map[string]any)
			if !ok || node == nil {
				continue
			}

			// Normalize action and its params for proto compatibility.
			if action, ok := node["action"].(map[string]any); ok && action != nil {
				normalizeActionForProto(action)
			}

			// Remove fields not in WorkflowNodeV2 proto.
			// 'type' is a ReactFlow field (e.g., "navigate", "click") - action.type captures this.
			// 'data' is a legacy V1 field - action contains all needed data.
			delete(node, "type")
			delete(node, "data")
		}
	}

	// Sanitize edges - normalize fields for WorkflowEdgeV2 proto.
	edges, ok := doc["edges"].([]any)
	if ok && len(edges) > 0 {
		for _, rawEdge := range edges {
			edge, ok := rawEdge.(map[string]any)
			if !ok || edge == nil {
				continue
			}

			// Convert camelCase handle fields to snake_case.
			if sh, ok := edge["sourceHandle"]; ok {
				edge["source_handle"] = sh
				delete(edge, "sourceHandle")
			}
			if th, ok := edge["targetHandle"]; ok {
				edge["target_handle"] = th
				delete(edge, "targetHandle")
			}

			// Convert edge type from UI format to proto enum.
			if t, ok := edge["type"].(string); ok {
				edge["type"] = normalizeEdgeType(t)
			}

			// Remove non-proto fields (ReactFlow visual/state properties).
			delete(edge, "markerEnd")
			delete(edge, "markerStart")
			delete(edge, "style")
			delete(edge, "animated")
			delete(edge, "data")
			delete(edge, "labelStyle")
			delete(edge, "labelBgStyle")
			delete(edge, "labelBgPadding")
			delete(edge, "labelBgBorderRadius")
			delete(edge, "className")
			delete(edge, "zIndex")
			delete(edge, "ariaLabel")
			delete(edge, "interactionWidth")
			delete(edge, "focusable")
			delete(edge, "hidden")
			delete(edge, "deletable")
			delete(edge, "selectable")
			delete(edge, "updatable")
			delete(edge, "pathOptions")
		}
	}
}

// normalizeEdgeType converts UI edge type strings to proto enum names.
func normalizeEdgeType(uiType string) string {
	switch strings.ToLower(uiType) {
	case "smoothstep":
		return "WORKFLOW_EDGE_TYPE_SMOOTHSTEP"
	case "step":
		return "WORKFLOW_EDGE_TYPE_STEP"
	case "straight":
		return "WORKFLOW_EDGE_TYPE_STRAIGHT"
	case "bezier":
		return "WORKFLOW_EDGE_TYPE_BEZIER"
	case "default", "":
		return "WORKFLOW_EDGE_TYPE_DEFAULT"
	default:
		return "WORKFLOW_EDGE_TYPE_DEFAULT"
	}
}

// normalizeActionForProto normalizes an action definition for proto serialization.
// Handles enum conversions, field name normalization, and subflow args wrapping.
func normalizeActionForProto(action map[string]any) {
	// Normalize navigate params
	if nav, ok := action["navigate"].(map[string]any); ok && nav != nil {
		normalizeNavigateParams(nav)
	}

	// Normalize click params
	if click, ok := action["click"].(map[string]any); ok && click != nil {
		normalizeClickParams(click)
	}

	// Normalize wait params
	if wait, ok := action["wait"].(map[string]any); ok && wait != nil {
		normalizeWaitParams(wait)
	}

	// Normalize assert params
	if assert, ok := action["assert"].(map[string]any); ok && assert != nil {
		normalizeAssertParams(assert)
	}

	// Wrap primitive subflow args into expected JsonValue shape.
	if subflow, ok := action["subflow"].(map[string]any); ok && subflow != nil {
		if args, ok := subflow["args"].(map[string]any); ok && args != nil {
			normalized := make(map[string]any, len(args))
			for k, v := range args {
				normalized[k] = typeconv.WrapJsonValue(v)
			}
			subflow["args"] = normalized
		}
	}
}

// normalizeNavigateParams converts UI-oriented navigate params to proto format.
func normalizeNavigateParams(nav map[string]any) {
	// Convert destinationType enum
	if dt, ok := nav["destinationType"].(string); ok {
		nav["destination_type"] = normalizeNavigateDestinationType(dt)
		delete(nav, "destinationType")
	}
	if dt, ok := nav["destination_type"].(string); ok {
		nav["destination_type"] = normalizeNavigateDestinationType(dt)
	}

	// Convert waitUntil enum
	if wu, ok := nav["waitUntil"].(string); ok {
		nav["wait_until"] = normalizeNavigateWaitEvent(wu)
		delete(nav, "waitUntil")
	}
	if wu, ok := nav["wait_until"].(string); ok {
		nav["wait_until"] = normalizeNavigateWaitEvent(wu)
	}

	// Convert camelCase field names to snake_case
	renameCamelToSnake(nav, "waitForSelector", "wait_for_selector")
	renameCamelToSnake(nav, "timeoutMs", "timeout_ms")
	renameCamelToSnake(nav, "scenarioPath", "scenario_path")
}

// normalizeClickParams converts UI-oriented click params to proto format.
func normalizeClickParams(click map[string]any) {
	// Convert button enum
	if btn, ok := click["button"].(string); ok {
		click["button"] = normalizeMouseButton(btn)
	}

	// Convert camelCase field names
	renameCamelToSnake(click, "clickCount", "click_count")
	renameCamelToSnake(click, "timeoutMs", "timeout_ms")
	renameCamelToSnake(click, "offsetX", "offset_x")
	renameCamelToSnake(click, "offsetY", "offset_y")
}

// normalizeWaitParams converts UI-oriented wait params to proto format.
func normalizeWaitParams(wait map[string]any) {
	// Convert state enum
	if state, ok := wait["state"].(string); ok {
		wait["state"] = normalizeWaitState(state)
	}

	// Convert camelCase field names
	renameCamelToSnake(wait, "timeoutMs", "timeout_ms")
	renameCamelToSnake(wait, "durationMs", "duration_ms")
}

// normalizeAssertParams converts UI-oriented assert params to proto format.
func normalizeAssertParams(assert map[string]any) {
	// Convert mode enum
	if mode, ok := assert["mode"].(string); ok {
		assert["mode"] = normalizeAssertionMode(mode)
	}

	// Convert camelCase field names
	renameCamelToSnake(assert, "timeoutMs", "timeout_ms")
	renameCamelToSnake(assert, "expectedValue", "expected_value")
}

// renameCamelToSnake renames a camelCase key to snake_case if present.
func renameCamelToSnake(m map[string]any, camel, snake string) {
	if v, ok := m[camel]; ok {
		m[snake] = v
		delete(m, camel)
	}
}

// normalizeNavigateDestinationType converts UI destination type to proto enum.
func normalizeNavigateDestinationType(dt string) string {
	switch strings.ToLower(dt) {
	case "url":
		return "NAVIGATE_DESTINATION_TYPE_URL"
	case "scenario":
		return "NAVIGATE_DESTINATION_TYPE_SCENARIO"
	default:
		return "NAVIGATE_DESTINATION_TYPE_UNSPECIFIED"
	}
}

// normalizeNavigateWaitEvent converts UI wait event to proto enum.
func normalizeNavigateWaitEvent(we string) string {
	switch strings.ToLower(we) {
	case "load":
		return "NAVIGATE_WAIT_EVENT_LOAD"
	case "domcontentloaded":
		return "NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED"
	case "networkidle":
		return "NAVIGATE_WAIT_EVENT_NETWORKIDLE"
	case "commit":
		return "NAVIGATE_WAIT_EVENT_COMMIT"
	default:
		return "NAVIGATE_WAIT_EVENT_UNSPECIFIED"
	}
}

// normalizeMouseButton converts UI button name to proto enum.
func normalizeMouseButton(btn string) string {
	switch strings.ToLower(btn) {
	case "left":
		return "MOUSE_BUTTON_LEFT"
	case "right":
		return "MOUSE_BUTTON_RIGHT"
	case "middle":
		return "MOUSE_BUTTON_MIDDLE"
	default:
		return "MOUSE_BUTTON_UNSPECIFIED"
	}
}

// normalizeWaitState converts UI wait state to proto enum.
func normalizeWaitState(state string) string {
	switch strings.ToLower(state) {
	case "attached":
		return "WAIT_STATE_ATTACHED"
	case "detached":
		return "WAIT_STATE_DETACHED"
	case "visible":
		return "WAIT_STATE_VISIBLE"
	case "hidden":
		return "WAIT_STATE_HIDDEN"
	default:
		return "WAIT_STATE_UNSPECIFIED"
	}
}

// normalizeAssertionMode converts UI assertion mode to proto enum.
func normalizeAssertionMode(mode string) string {
	switch strings.ToLower(mode) {
	case "exists":
		return "ASSERTION_MODE_EXISTS"
	case "not_exists", "notexists":
		return "ASSERTION_MODE_NOT_EXISTS"
	case "visible":
		return "ASSERTION_MODE_VISIBLE"
	case "hidden":
		return "ASSERTION_MODE_HIDDEN"
	case "text_contains", "textcontains":
		return "ASSERTION_MODE_TEXT_CONTAINS"
	case "text_equals", "textequals":
		return "ASSERTION_MODE_TEXT_EQUALS"
	case "attribute_equals", "attributeequals":
		return "ASSERTION_MODE_ATTRIBUTE_EQUALS"
	case "url_contains", "urlcontains":
		return "ASSERTION_MODE_URL_CONTAINS"
	case "url_equals", "urlequals":
		return "ASSERTION_MODE_URL_EQUALS"
	case "title_contains", "titlecontains":
		return "ASSERTION_MODE_TITLE_CONTAINS"
	case "title_equals", "titleequals":
		return "ASSERTION_MODE_TITLE_EQUALS"
	default:
		return "ASSERTION_MODE_UNSPECIFIED"
	}
}
