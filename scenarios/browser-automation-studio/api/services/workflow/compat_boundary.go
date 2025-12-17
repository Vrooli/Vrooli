package workflow

import (
	"encoding/json"
	"fmt"

	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// BuildFlowDefinitionV2ForWrite converts an incoming flow definition map to a WorkflowDefinitionV2.
// This is the write compat boundary: it first attempts strict V2 protojson parsing, and falls back
// to V1 nodes/edges conversion only when the payload looks like legacy format.
func BuildFlowDefinitionV2ForWrite(flow map[string]any, metadata map[string]any, settings map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	if flow == nil {
		return &basworkflows.WorkflowDefinitionV2{}, nil
	}

	// Merge metadata and settings into the flow for conversion if provided separately.
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

	// First attempt: strict V2 protojson parsing.
	if looksLikeV2(merged) {
		normalizeV2Compat(merged)
		body, err := json.Marshal(merged)
		if err == nil {
			var pb basworkflows.WorkflowDefinitionV2
			if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(body, &pb); err == nil {
				return &pb, nil
			}
		}
	}

	// Fallback: treat as V1 nodes/edges and convert to V2.
	nodes := ToInterfaceSlice(merged["nodes"])
	edges := ToInterfaceSlice(merged["edges"])

	v2, err := autocompiler.V1FlowDefinitionToV2(autocompiler.V1FlowDefinition{
		Nodes:    v1NodesFromAny(nodes),
		Edges:    v1EdgesFromAny(edges),
		Metadata: extractMapAny(merged, "metadata"),
		Settings: extractMapAny(merged, "settings"),
	})
	if err != nil {
		return nil, fmt.Errorf("convert to v2: %w", err)
	}

	return v2, nil
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

// looksLikeV2 checks if a flow definition map appears to be in V2 format.
// V2 format has nodes with "action" fields containing typed action definitions.
func looksLikeV2(doc map[string]any) bool {
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return false
	}
	first, ok := nodes[0].(map[string]any)
	if !ok || first == nil {
		return false
	}
	_, hasAction := first["action"]
	return hasAction
}

// normalizeV2Compat normalizes a V2 flow definition for proto compatibility.
// Handles UI-oriented execution defaults and JsonValue wrapping for subflow args.
func normalizeV2Compat(doc map[string]any) {
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

	// Wrap primitive subflow args into expected JsonValue shape.
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
			normalized[k] = jsonValueWrapper(v)
		}
		subflow["args"] = normalized
	}
}

// jsonValueWrapper wraps a value in the expected JsonValue oneof shape if needed.
func jsonValueWrapper(v any) any {
	if v == nil {
		return map[string]any{"null_value": 0}
	}
	switch val := v.(type) {
	case map[string]any:
		// If already wrapped, return as-is.
		if _, hasString := val["string_value"]; hasString {
			return val
		}
		if _, hasBool := val["bool_value"]; hasBool {
			return val
		}
		if _, hasNumber := val["number_value"]; hasNumber {
			return val
		}
		if _, hasNull := val["null_value"]; hasNull {
			return val
		}
		if _, hasObject := val["object_value"]; hasObject {
			return val
		}
		if _, hasList := val["list_value"]; hasList {
			return val
		}
		// Wrap object.
		return map[string]any{"object_value": val}
	case string:
		return map[string]any{"string_value": val}
	case bool:
		return map[string]any{"bool_value": val}
	case float64:
		return map[string]any{"number_value": val}
	case int:
		return map[string]any{"number_value": float64(val)}
	case int32:
		return map[string]any{"number_value": float64(val)}
	case int64:
		return map[string]any{"number_value": float64(val)}
	case []any:
		return map[string]any{"list_value": val}
	default:
		return v
	}
}
