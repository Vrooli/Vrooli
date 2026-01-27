package workflows

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/workflow/validator"
	"google.golang.org/protobuf/encoding/protojson"
)

// BuildWorkflowFromSteps creates a workflow definition from parsed step specs.
func BuildWorkflowFromSteps(steps []*StepSpec) (map[string]any, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	nodes := make([]map[string]any, len(steps))
	edges := make([]map[string]any, 0, len(steps)-1)

	for i, step := range steps {
		nodeID := fmt.Sprintf("step-%d", i+1)
		node, err := buildNode(nodeID, step)
		if err != nil {
			return nil, fmt.Errorf("step %d (%s): %w", i+1, step.Type, err)
		}
		nodes[i] = node

		// Create edge from previous node
		if i > 0 {
			edges = append(edges, map[string]any{
				"id":     fmt.Sprintf("edge-%d", i),
				"source": fmt.Sprintf("step-%d", i),
				"target": nodeID,
			})
		}
	}

	return map[string]any{
		"metadata": map[string]any{
			"name":        "inline-workflow",
			"description": "Generated from CLI --step flags",
		},
		"nodes": nodes,
		"edges": edges,
	}, nil
}

// buildNode creates a node definition from a step spec using the API's compiler.
// The output is proto-compatible JSON format that the API expects.
func buildNode(id string, step *StepSpec) (map[string]any, error) {
	// Build params map from step spec
	params := make(map[string]any)

	// Set positional argument using the step definition's MapsTo field
	if step.Positional != "" {
		key := getPositionalMappingKey(step.Type)
		params[key] = step.Positional
	}

	// Apply all key-value pairs
	for k, v := range step.KVPairs {
		params[k] = parseValue(v)
	}

	// Use the API's compiler to build a proper ActionDefinition proto
	action, err := compiler.BuildActionDefinition(step.Type, params)
	if err != nil {
		return nil, err
	}

	// Serialize to JSON using protojson (handles proto field names correctly)
	jsonBytes, err := protojson.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal action: %w", err)
	}

	// Parse into a map for inclusion in the workflow
	var actionMap map[string]any
	if err := json.Unmarshal(jsonBytes, &actionMap); err != nil {
		return nil, fmt.Errorf("failed to parse action JSON: %w", err)
	}

	return map[string]any{
		"id":     id,
		"action": actionMap,
	}, nil
}

// getPositionalMappingKey returns the parameter key that a positional argument should map to
// for a given step type. This uses the step definition's MapsTo field from the validator package.
func getPositionalMappingKey(stepType string) string {
	def, ok := validator.GetStepDefinition(stepType)
	if ok && def.Positional != nil {
		return def.Positional.MapsTo
	}
	return "selector" // Default fallback for most actions
}
