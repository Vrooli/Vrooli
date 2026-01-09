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

	// 1. JsonValue wrapping for parameter maps
	if params, ok := raw["parameters"].(map[string]any); ok && params != nil {
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

// NormalizeWorkflowDefinitionV2 applies V2 compatibility transformations to a workflow definition.
// This handles both settings normalization and node-level subflow args wrapping.
//
// Transformations:
//   - executionViewport → viewport_width/viewport_height in settings
//   - Removes defaultStepTimeoutMs
//   - Wraps subflow args into JsonValue oneof shape
func NormalizeWorkflowDefinitionV2(doc map[string]any) {
	// Normalize settings
	settings, ok := doc["settings"].(map[string]any)
	if ok && settings != nil {
		normalizeViewportSettings(settings)
	}

	// Remove non-proto metadata fields (used by test-genie/playbooks metadata)
	if metadata, ok := doc["metadata"].(map[string]any); ok && metadata != nil {
		delete(metadata, "reset")
	}

	// Normalize subflow args in nodes
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return
	}
	for _, rawNode := range nodes {
		node, ok := rawNode.(map[string]any)
		if !ok || node == nil {
			continue
		}
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
