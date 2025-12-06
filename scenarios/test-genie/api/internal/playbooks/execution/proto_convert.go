// Package execution provides BAS API client and proto conversion utilities.
package execution

import (
	"fmt"

	"test-genie/internal/playbooks/types"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// WorkflowToProto converts a map[string]any workflow definition to a proto WorkflowDefinition.
//
// DEPRECATED: This function is no longer used in the main execution flow because BAS
// uses standard JSON decoding (not protojson) and expects node types as plain strings
// like "navigate" rather than proto enum names like "STEP_TYPE_NAVIGATE". The main
// ExecuteWorkflow function now sends plain JSON instead.
//
// This function is kept for backwards compatibility and for cases where proto
// serialization might be needed (e.g., future WebSocket streaming of workflow definitions).
func WorkflowToProto(definition map[string]any) (*basv1.WorkflowDefinition, error) {
	if definition == nil {
		return nil, fmt.Errorf("workflow definition is nil")
	}

	proto := &basv1.WorkflowDefinition{}

	// Convert nodes
	if nodesRaw, ok := definition["nodes"]; ok {
		nodes, err := convertNodes(nodesRaw)
		if err != nil {
			return nil, fmt.Errorf("nodes: %w", err)
		}
		proto.Nodes = nodes
	}

	// Convert edges
	if edgesRaw, ok := definition["edges"]; ok {
		edges, err := convertEdges(edgesRaw)
		if err != nil {
			return nil, fmt.Errorf("edges: %w", err)
		}
		proto.Edges = edges
	}

	// Convert metadata
	if metadataRaw, ok := definition["metadata"].(map[string]any); ok {
		metadata, err := toValueMap(metadataRaw)
		if err != nil {
			return nil, fmt.Errorf("metadata: %w", err)
		}
		proto.Metadata = metadata
	}

	// Convert settings
	if settingsRaw, ok := definition["settings"].(map[string]any); ok {
		settings, err := toValueMap(settingsRaw)
		if err != nil {
			return nil, fmt.Errorf("settings: %w", err)
		}
		proto.Settings = settings
	}

	return proto, nil
}

// convertNodes converts a raw nodes array to proto WorkflowNode slice.
func convertNodes(nodesRaw any) ([]*basv1.WorkflowNode, error) {
	nodesSlice, ok := nodesRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", nodesRaw)
	}

	nodes := make([]*basv1.WorkflowNode, 0, len(nodesSlice))
	for i, nodeRaw := range nodesSlice {
		nodeMap, ok := nodeRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("[%d]: expected object, got %T", i, nodeRaw)
		}

		node := &basv1.WorkflowNode{
			Id:   getString(nodeMap, "id"),
			Type: types.StringToStepType(getString(nodeMap, "type")),
		}

		// Convert node data to Struct
		if dataRaw, ok := nodeMap["data"].(map[string]any); ok {
			data, err := structpb.NewStruct(dataRaw)
			if err != nil {
				return nil, fmt.Errorf("[%d].data: %w", i, err)
			}
			node.Data = data
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// convertEdges converts a raw edges array to proto WorkflowEdge slice.
func convertEdges(edgesRaw any) ([]*basv1.WorkflowEdge, error) {
	edgesSlice, ok := edgesRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", edgesRaw)
	}

	edges := make([]*basv1.WorkflowEdge, 0, len(edgesSlice))
	for i, edgeRaw := range edgesSlice {
		edgeMap, ok := edgeRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("[%d]: expected object, got %T", i, edgeRaw)
		}

		edge := &basv1.WorkflowEdge{
			Id:     getString(edgeMap, "id"),
			Source: getString(edgeMap, "source"),
			Target: getString(edgeMap, "target"),
			Type:   getString(edgeMap, "type"),
		}

		// Convert edge data to Struct if present
		if dataRaw, ok := edgeMap["data"].(map[string]any); ok {
			data, err := structpb.NewStruct(dataRaw)
			if err != nil {
				return nil, fmt.Errorf("[%d].data: %w", i, err)
			}
			edge.Data = data
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// toValueMap converts map[string]any to map[string]*structpb.Value.
func toValueMap(source map[string]any) (map[string]*structpb.Value, error) {
	if len(source) == 0 {
		return nil, nil
	}

	result := make(map[string]*structpb.Value, len(source))
	for key, val := range source {
		pbVal, err := structpb.NewValue(val)
		if err != nil {
			return nil, fmt.Errorf("[%s]: %w", key, err)
		}
		result[key] = pbVal
	}
	return result, nil
}

// getString safely extracts a string from a map.
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// BuildAdhocRequest creates an ExecuteAdhocRequest proto from workflow definition and name.
//
// DEPRECATED: This function is no longer used in the main execution flow. See WorkflowToProto
// for the reason. The ExecuteWorkflow function now builds plain JSON requests directly.
func BuildAdhocRequest(definition map[string]any, name string) (*basv1.ExecuteAdhocRequest, error) {
	flowDef, err := WorkflowToProto(definition)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow definition: %w", err)
	}

	return &basv1.ExecuteAdhocRequest{
		FlowDefinition:    flowDef,
		Parameters:        nil, // Empty parameters
		WaitForCompletion: false,
		Metadata: &basv1.ExecutionMetadata{
			Name: name,
		},
	}, nil
}
