// Package workflow owns schema-native normalization for V2 workflow ingress.
package workflow

import (
	"encoding/json"
	"fmt"
	"strings"

	contractvalues "github.com/vrooli/browser-automation-studio/automation/contracts"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// NormalizeExecuteAdhocRequest applies V2 schema-native short-form
// normalization to raw JSON bytes for the ExecuteAdhocRequest endpoint.
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
		normalizeExecutionParameterAliases(params)
		NormalizeExecutionParameters(params)
		contractvalues.NormalizeJsonValueMaps(params, "initial_params", "initialParams", "initial_store", "initialStore", "env")
	}

	// 2. Normalize workflow definition fields for proto compatibility
	if flowDef, ok := raw["flow_definition"].(map[string]any); ok && flowDef != nil {
		if err := RejectLegacyWorkflowNodes(flowDef); err != nil {
			return nil, err
		}
		NormalizeWorkflowDefinitionV2(flowDef)
	}

	return json.Marshal(raw)
}

// executionParameterFieldNames is derived from the canonical descriptor so
// normalization cannot silently drift behind ExecutionParameters additions.
// Both proto field names and ProtoJSON names are accepted at ingress.
var executionParameterFieldNames = func() map[string]string {
	fields := (&basexecution.ExecutionParameters{}).ProtoReflect().Descriptor().Fields()
	names := make(map[string]string, fields.Len()*2)
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		jsonName := field.JSONName()
		names[string(field.Name())] = jsonName
		names[jsonName] = jsonName
	}
	return names
}()

// normalizeExecutionParameterAliases resolves snake_case/proto-JSON aliases
// before the JsonValue wrappers are rebuilt. If both spellings are supplied,
// the ProtoJSON JSON-name spelling wins.
func normalizeExecutionParameterAliases(params map[string]any) {
	for name, jsonName := range executionParameterFieldNames {
		if name == jsonName {
			continue
		}
		value, exists := params[name]
		if !exists {
			continue
		}
		if _, alreadyPresent := params[jsonName]; !alreadyPresent {
			params[jsonName] = value
		}
		delete(params, name)
	}
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
		if _, known := executionParameterFieldNames[key]; !known {
			unknownFields[key] = value
		}
	}

	// If no unknown fields, return as-is
	if len(unknownFields) == 0 {
		return
	}

	// Move unknown fields to initial_params
	initialParamsKey := "initial_params"
	var initialParams map[string]any
	if ip, ok := params["initialParams"].(map[string]any); ok {
		initialParams = ip
		initialParamsKey = "initialParams"
	} else if ip, ok := params["initial_params"].(map[string]any); ok {
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

	params[initialParamsKey] = initialParams
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

// NormalizeWorkflowDefinitionV2Bytes applies concise V2 schema field forms
// (short-form execution_mode and viewport settings) to a STANDALONE
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
	if err := RejectLegacyWorkflowNodes(doc); err != nil {
		return nil, err
	}
	NormalizeWorkflowDefinitionV2(doc)
	return json.Marshal(doc)
}

// RejectLegacyWorkflowNodes makes the V2-only workflow boundary explicit.
// Action-bearing nodes must use the typed action oneof; the historical
// node.type + node.data representation is never upgraded during execution.
func RejectLegacyWorkflowNodes(doc map[string]any) error {
	nodes, ok := doc["nodes"].([]any)
	if !ok {
		return nil
	}
	for index, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node == nil {
			continue
		}
		if _, hasAction := node["action"]; hasAction {
			continue
		}
		if _, hasType := node["type"]; hasType {
			return fmt.Errorf("workflow node %d uses legacy type/data shape; provide action instead", index)
		}
	}
	return nil
}

// NormalizeWorkflowDefinitionV2 applies schema-native short-form transformations
// to an already typed V2 workflow definition.
//
// Transformations:
//   - executionViewport → viewport_width/viewport_height in settings
//   - Removes defaultStepTimeoutMs
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

		// Map short-form execution_mode to proto enum name while preserving
		// the caller's field spelling; protojson accepts both proto and JSON names.
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

		// Handle subflow args wrapping for V2 nodes
		action, ok := node["action"].(map[string]any)
		if !ok || action == nil {
			continue
		}
		normalizeActionForProtoJSON(action)
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
			normalized[k] = contractvalues.WrapJsonValue(v)
		}
		subflow["args"] = normalized
	}
}

// normalizeActionForProtoJSON preserves the concise workflow-schema vocabulary
// while producing the enum and field forms protojson requires for typed V2 nodes.
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
			assertion["expected"] = contractvalues.WrapJsonValue(expected)
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
			settings["viewport_width"] = contractvalues.ToInt32Val(width)
		}
		if height, ok := viewport["height"]; ok {
			settings["viewport_height"] = contractvalues.ToInt32Val(height)
		}
		delete(settings, "executionViewport")
	}
	delete(settings, "defaultStepTimeoutMs")
}
