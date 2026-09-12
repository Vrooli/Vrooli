package workflow

import (
	"encoding/json"
	"errors"
	"fmt"

	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// ErrInvalidWorkflowFormat is returned when a workflow is missing required V2 structure.
// All persisted workflows must have nodes with typed ActionDefinition values.
var ErrInvalidWorkflowFormat = errors.New("invalid workflow format: nodes must have 'action' field with typed action definitions")

// BuildFlowDefinitionV2ForWrite is the map-shaped ingress for newly-authored
// workflows. It deliberately accepts only V2 protojson: V1 conversion belongs
// to the explicit external-workflow migration path (ConvertExternalWorkflow),
// never to an ordinary write. No execution code consumes this map
// representation.
func BuildFlowDefinitionV2ForWrite(flow map[string]any, metadata map[string]any, settings map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	if flow == nil {
		return &basworkflows.WorkflowDefinitionV2{}, nil
	}

	merged := make(map[string]any, len(flow)+2)
	for key, value := range flow {
		merged[key] = value
	}
	if metadata != nil && merged["metadata"] == nil {
		merged["metadata"] = metadata
	}
	if settings != nil && merged["settings"] == nil {
		merged["settings"] = settings
	}

	raw, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow definition: %w", err)
	}
	definition := &basworkflows.WorkflowDefinitionV2{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, definition); err != nil {
		return nil, fmt.Errorf("decode workflow definition at ingress: %w", err)
	}
	if err := validateFlowDefinitionV2OnWrite(definition); err != nil {
		return nil, err
	}
	return definition, nil
}

// validateFlowDefinitionV2OnWrite validates the typed workflow before it is
// persisted. The core only receives this representation after ingress.
func validateFlowDefinitionV2OnWrite(def *basworkflows.WorkflowDefinitionV2) error {
	if def == nil {
		return nil
	}

	nodeIDs := make(map[string]struct{}, len(def.Nodes))
	for index, node := range def.Nodes {
		if node == nil {
			return fmt.Errorf("node at index %d is nil", index)
		}
		if node.Id == "" {
			return fmt.Errorf("node at index %d has empty id", index)
		}
		if node.Action == nil {
			return fmt.Errorf("node %q has no action defined", node.Id)
		}
		nodeIDs[node.Id] = struct{}{}
	}

	for index, edge := range def.Edges {
		if edge == nil {
			return fmt.Errorf("edge at index %d is nil", index)
		}
		if edge.Id == "" || edge.Source == "" || edge.Target == "" {
			return fmt.Errorf("edge at index %d has incomplete identifiers", index)
		}
		if _, ok := nodeIDs[edge.Source]; !ok {
			return fmt.Errorf("edge %q references unknown source node %q", edge.Id, edge.Source)
		}
		if _, ok := nodeIDs[edge.Target]; !ok {
			return fmt.Errorf("edge %q references unknown target node %q", edge.Id, edge.Target)
		}
	}
	return nil
}
