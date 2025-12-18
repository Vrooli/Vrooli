package workflow

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// ErrInvalidWorkflowFormat is returned when a workflow is missing required V2 structure.
// All workflows must use V2 format where nodes have an "action" field with typed action definitions.
// Legacy V1 format (nodes with type+data instead of action field) is rejected.
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

	// Parse V2 format.
	normalizeV2Compat(merged)
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
// Returns false for empty workflows or workflows missing the action field.
func isV2Format(doc map[string]any) bool {
	nodes, ok := doc["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		// Empty workflow is technically valid V2, but we check first node for non-empty
		return true
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
			normalized[k] = typeconv.WrapJsonValue(v)
		}
		subflow["args"] = normalized
	}
}
