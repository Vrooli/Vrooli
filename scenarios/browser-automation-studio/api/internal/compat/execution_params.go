// Package compat provides backwards compatibility adapters for legacy client formats.
// These functions normalize incoming JSON payloads to match the expected proto schema
// before unmarshaling, centralizing compatibility logic that was previously scattered
// across handler implementations.
package compat

import (
	"encoding/json"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/typeconv"
)

// NormalizeExecuteAdhocRequest applies backwards compatibility transformations
// to raw JSON bytes for the ExecuteAdhocRequest endpoint.
//
// Transformations:
//   - JsonValue wrapping for initial_params, initial_store, env
//   - executionViewport (camelCase) → viewport_width/viewport_height
//   - Removes unsupported UI settings (defaultStepTimeoutMs)
//   - Unknown parameter fields are moved to initial_params
//
// Removed transformations:
//   - execution_params → parameters: Removed 2024-12 after confirming no clients
//     use this field. All known callers (test-genie, UI) use "parameters" directly.
func NormalizeExecuteAdhocRequest(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return []byte("{}"), nil
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		// If it doesn't unmarshal, return as-is and let proto unmarshal handle the error
		return body, nil
	}

	// 1. Normalize execution parameters (move unknown fields to initial_params)
	if params, ok := raw["parameters"].(map[string]any); ok && params != nil {
		NormalizeExecutionParameters(params)
		typeconv.NormalizeJsonValueMaps(params, "initial_params", "initial_store", "env")
	}

	// 2. Normalize workflow definition fields for proto compatibility
	if flowDef, ok := raw["flow_definition"].(map[string]any); ok && flowDef != nil {
		NormalizeWorkflowDefinitionV2(flowDef)
	}

	// 3. Ensure protojson-compatible field names (lowerCamelCase)
	raw = normalizeProtoJSONKeys(raw).(map[string]any)

	return json.Marshal(raw)
}

// knownExecutionParamFields is the set of fields that are part of the ExecutionParameters proto.
var knownExecutionParamFields = map[string]bool{
	"initial_params":          true,
	"initialParams":           true,
	"initial_store":           true,
	"initialStore":            true,
	"env":                     true,
	"projectRoot":             true,
	"project_root":            true,
	"startUrl":                true,
	"start_url":               true,
	"session_profile_id":      true,
	"sessionProfileId":        true,
	"save_session_profile_id": true,
	"saveSessionProfileId":    true,
}

// NormalizeExecutionParameters moves unknown fields in params to initial_params.
// This allows users to pass custom workflow parameters directly in params
// without needing to nest them in initial_params.
//
// Example:
//
//	Input:  {"username": "test", "initial_params": {"existing": "value"}}
//	Output: {"initial_params": {"existing": "value", "username": "test"}}
func NormalizeExecutionParameters(params map[string]any) {
	if params == nil {
		return
	}

	// Find unknown fields
	unknownFields := make(map[string]any)
	for key, value := range params {
		if !knownExecutionParamFields[key] {
			unknownFields[key] = value
		}
	}

	// If no unknown fields, return as-is
	if len(unknownFields) == 0 {
		return
	}

	// Move unknown fields to initial_params
	var initialParams map[string]any
	if ip, ok := params["initial_params"].(map[string]any); ok {
		initialParams = ip
	} else if ip, ok := params["initialParams"].(map[string]any); ok {
		initialParams = ip
	}
	if initialParams == nil {
		initialParams = make(map[string]any)
	}

	for key, value := range unknownFields {
		// Don't overwrite existing values in initial_params
		if _, exists := initialParams[key]; !exists {
			initialParams[key] = value
		}
		delete(params, key)
	}

	params["initial_params"] = initialParams
}

// NormalizeWorkflowSettings normalizes UI-oriented settings in a workflow definition
// to match the expected proto schema.
//
// Transformations:
//   - executionViewport.width → viewport_width
//   - executionViewport.height → viewport_height
//   - Removes unsupported UI settings (defaultStepTimeoutMs, executionViewport)
func NormalizeWorkflowSettings(flowDef map[string]any) {
	settings, ok := flowDef["settings"].(map[string]any)
	if !ok || settings == nil {
		return
	}
	normalizeViewportSettings(settings)
}

// executionModeMapping maps human-friendly short-form execution_mode values
// to their proto ExecutionMode enum names. Workflow JSON files use the short
// form ("observer") for readability; protojson requires the full enum name.
var executionModeMapping = map[string]string{
	"observer":    "EXECUTION_MODE_OBSERVER",
	"mutating":    "EXECUTION_MODE_MUTATING",
	"destructive": "EXECUTION_MODE_DESTRUCTIVE",
}

// NormalizeWorkflowDefinitionV2Bytes applies the same compatibility transforms
// the flow-file loader uses (short-form execution_mode → enum, viewport
// settings, V1→V2 node shape, lowerCamelCase keys) to a STANDALONE
// WorkflowDefinitionV2 JSON body, returning protojson-ready bytes. Used by the
// capture path, which splices a raw bas/flows body that must parse identically
// to one fed through `execute-adhoc --flow-file`.
func NormalizeWorkflowDefinitionV2Bytes(body []byte) ([]byte, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	NormalizeWorkflowDefinitionV2(doc)
	normalized, ok := normalizeProtoJSONKeys(doc).(map[string]any)
	if !ok {
		return json.Marshal(doc)
	}
	return json.Marshal(normalized)
}

// NormalizeWorkflowDefinitionV2 applies V2 compatibility transformations to a workflow definition.
// This handles both settings normalization and node-level transformations.
//
// Transformations:
//   - executionViewport → viewport_width/viewport_height in settings
//   - Removes defaultStepTimeoutMs
//   - Converts V1 nodes (type+data) to V2 format (action oneof)
//   - Wraps subflow args into JsonValue oneof shape
//   - Maps short-form execution_mode values to proto enum names
func NormalizeWorkflowDefinitionV2(doc map[string]any) {
	// Normalize settings
	settings, ok := doc["settings"].(map[string]any)
	if ok && settings != nil {
		normalizeViewportSettings(settings)
	}

	// Normalize metadata
	if metadata, ok := doc["metadata"].(map[string]any); ok && metadata != nil {
		// Remove non-proto metadata fields (used by test-genie/playbooks metadata)
		delete(metadata, "reset")

		// Map short-form execution_mode to proto enum name.
		// Check both snake_case and camelCase since normalizeProtoJSONKeys runs later.
		for _, key := range []string{"execution_mode", "executionMode"} {
			if val, ok := metadata[key].(string); ok {
				if mapped, found := executionModeMapping[val]; found {
					metadata[key] = mapped
				}
			}
		}
	}

	// Normalize nodes
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node == nil {
			continue
		}

		// Transform V1 nodes to V2 format
		normalizeNodeV1ToV2(node)

		// Handle subflow args wrapping for V2 nodes
		action, ok := node["action"].(map[string]any)
		if !ok || action == nil {
			continue
		}
		subflow, ok := action["subflow"].(map[string]any)
		if !ok || subflow == nil {
			continue
		}
		args, ok := subflow["args"].(map[string]any)
		if !ok || args == nil {
			continue
		}

		normalized := make(map[string]any, len(args))
		for k, v := range args {
			normalized[k] = typeconv.WrapJsonValue(v)
		}
		subflow["args"] = normalized
	}
}

// normalizeNodeV1ToV2 converts a V1-format node (type+data) to V2 format (action oneof).
// V1: { "id": "x", "type": "click", "data": { "selector": "#btn", "label": "My Label" } }
// V2: { "id": "x", "action": { "type": "ACTION_TYPE_CLICK", "click": { "selector": "#btn" }, "metadata": { "label": "My Label" } } }
func normalizeNodeV1ToV2(node map[string]any) {
	// Check if this is a V1 node (has "type" field but no "action" field)
	stepType, hasType := node["type"].(string)
	if !hasType || stepType == "" {
		return
	}
	if _, hasAction := node["action"]; hasAction {
		// Already V2 format
		return
	}

	// Extract V1 data
	data, _ := node["data"].(map[string]any)
	if data == nil {
		data = make(map[string]any)
	}

	// Build V2 action structure
	actionType := stepTypeToActionType(stepType)
	paramsKey := stepTypeToParamsKey(stepType)
	params := normalizeParamsData(stepType, data)

	// Extract label for metadata
	var metadata map[string]any
	if label, ok := data["label"].(string); ok && label != "" {
		metadata = map[string]any{"label": label}
		delete(params, "label")
	}

	// Build the action map
	action := map[string]any{
		"type": actionType,
	}
	if len(params) > 0 {
		action[paramsKey] = params
	}
	normalizeActionForProtoJSON(action)
	if metadata != nil {
		action["metadata"] = metadata
	}

	// Replace V1 fields with V2 structure
	delete(node, "type")
	delete(node, "data")
	node["action"] = action
}

// stepTypeToActionType maps V1 step type strings to V2 ACTION_TYPE enum names.
func stepTypeToActionType(stepType string) string {
	mapping := map[string]string{
		"navigate":   "ACTION_TYPE_NAVIGATE",
		"click":      "ACTION_TYPE_CLICK",
		"type":       "ACTION_TYPE_INPUT",
		"input":      "ACTION_TYPE_INPUT",
		"assert":     "ACTION_TYPE_ASSERT",
		"wait":       "ACTION_TYPE_WAIT",
		"screenshot": "ACTION_TYPE_SCREENSHOT",
		"evaluate":   "ACTION_TYPE_EVALUATE",
		"hover":      "ACTION_TYPE_HOVER",
		"focus":      "ACTION_TYPE_FOCUS",
		"blur":       "ACTION_TYPE_BLUR",
		"select":     "ACTION_TYPE_SELECT",
		"keyboard":   "ACTION_TYPE_KEYBOARD",
		"shortcut":   "ACTION_TYPE_SHORTCUT",
		"extract":    "ACTION_TYPE_EXTRACT",
		"subflow":    "ACTION_TYPE_SUBFLOW",
		"dragDrop":   "ACTION_TYPE_DRAG_DROP",
		"drag_drop":  "ACTION_TYPE_DRAG_DROP",
		"loop":       "ACTION_TYPE_LOOP",
		"condition":  "ACTION_TYPE_CONDITION",
		"set":        "ACTION_TYPE_SET",
		"scroll":     "ACTION_TYPE_SCROLL",
	}
	if actionType, ok := mapping[stepType]; ok {
		return actionType
	}
	// Fallback: uppercase with prefix
	return "ACTION_TYPE_" + strings.ToUpper(stepType)
}

// stepTypeToParamsKey maps V1 step type to the V2 params field name.
// Most map directly (click -> click), but "type" -> "input".
func stepTypeToParamsKey(stepType string) string {
	mapping := map[string]string{
		"type":      "input",
		"dragDrop":  "dragDrop",
		"drag_drop": "dragDrop",
	}
	if key, ok := mapping[stepType]; ok {
		return key
	}
	return stepType
}

// normalizeParamsData handles field renames between V1 data and V2 params.
func normalizeParamsData(stepType string, data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = v
	}

	// Handle "type" step (input action) field renames
	if stepType == "type" || stepType == "input" {
		// V1 uses "text", V2 uses "value"
		if text, ok := result["text"]; ok {
			if _, hasValue := result["value"]; !hasValue {
				result["value"] = text
			}
			delete(result, "text")
		}
	}

	// Remove "label" as it moves to metadata
	delete(result, "label")

	return result
}

// normalizeActionForProtoJSON preserves the concise workflow-schema vocabulary
// while producing the enum and field forms protojson requires. This runs for
// V1-shaped workflow files after their action wrapper is constructed, so every
// file-loading entry point accepts the same authored form.
func normalizeActionForProtoJSON(action map[string]any) {
	if navigate, ok := action["navigate"].(map[string]any); ok {
		if destination, ok := navigate["destinationType"].(string); ok {
			navigate["destination_type"] = normalizeNavigateDestinationType(destination)
			delete(navigate, "destinationType")
		}
		if waitUntil, ok := navigate["waitUntil"].(string); ok {
			navigate["wait_until"] = normalizeNavigateWaitEvent(waitUntil)
			delete(navigate, "waitUntil")
		}
	}
	if assertion, ok := action["assert"].(map[string]any); ok {
		if mode, ok := assertion["assertMode"].(string); ok {
			assertion["mode"] = normalizeAssertionMode(mode)
			delete(assertion, "assertMode")
		}
		if expected, ok := assertion["expectedValue"]; ok {
			assertion["expected"] = typeconv.WrapJsonValue(expected)
			delete(assertion, "expectedValue")
		}
	}
}

func normalizeNavigateDestinationType(value string) string {
	switch strings.ToLower(value) {
	case "url":
		return "NAVIGATE_DESTINATION_TYPE_URL"
	case "scenario":
		return "NAVIGATE_DESTINATION_TYPE_SCENARIO"
	default:
		return "NAVIGATE_DESTINATION_TYPE_UNSPECIFIED"
	}
}

func normalizeNavigateWaitEvent(value string) string {
	switch strings.ToLower(value) {
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

func normalizeAssertionMode(value string) string {
	switch strings.ToLower(value) {
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
	default:
		return "ASSERTION_MODE_UNSPECIFIED"
	}
}

// normalizeViewportSettings handles the executionViewport camelCase → snake_case conversion.
func normalizeViewportSettings(settings map[string]any) {
	viewport, ok := settings["executionViewport"].(map[string]any)
	if ok && viewport != nil {
		if width, ok := viewport["width"]; ok {
			settings["viewport_width"] = typeconv.ToInt32Val(width)
		}
		if height, ok := viewport["height"]; ok {
			settings["viewport_height"] = typeconv.ToInt32Val(height)
		}
		delete(settings, "executionViewport")
	}
	delete(settings, "defaultStepTimeoutMs")
}

func normalizeProtoJSONKeys(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, raw := range v {
			normalizedKey := toLowerCamel(key)
			normalizedVal := normalizeProtoJSONKeys(raw)
			if _, exists := out[normalizedKey]; exists && normalizedKey != key {
				// Preserve existing key if both snake_case and camelCase are present.
				continue
			}
			out[normalizedKey] = normalizedVal
		}
		return out
	case []any:
		for i := range v {
			v[i] = normalizeProtoJSONKeys(v[i])
		}
		return v
	default:
		return value
	}
}

func toLowerCamel(input string) string {
	if !strings.ContainsRune(input, '_') {
		return input
	}
	parts := strings.Split(input, "_")
	if len(parts) == 0 {
		return input
	}
	var b strings.Builder
	b.WriteString(parts[0])
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
