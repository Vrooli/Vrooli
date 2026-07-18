package graph

import (
	"fmt"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"google.golang.org/protobuf/proto"
)

func encodeGraphResponse(resp GraphResponse) (*apipb.GraphResponse, error) {
	nodes := make([]*domainpb.GraphNode, 0, len(resp.Nodes))
	for _, node := range resp.Nodes {
		protoNode, err := encodeGraphNode(node)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, protoNode)
	}

	edges := make([]*domainpb.GraphEdge, 0, len(resp.Edges))
	for _, edge := range resp.Edges {
		edges = append(edges, &domainpb.GraphEdge{
			Id:     edge.ID,
			Source: edge.Source,
			Target: edge.Target,
			Type:   edge.Type,
		})
	}

	meta := &domainpb.GraphMeta{
		Lens:        string(resp.Meta.Lens),
		NodeCount:   int32(resp.Meta.NodeCount),
		EdgeCount:   int32(resp.Meta.EdgeCount),
		GeneratedAt: resp.Meta.GeneratedAt,
	}
	if resp.Meta.AgentManagerAvailable != nil {
		meta.AgentManagerAvailable = proto.Bool(*resp.Meta.AgentManagerAvailable)
	}
	if resp.Meta.Hint != "" {
		meta.Hint = proto.String(resp.Meta.Hint)
	}

	return &apipb.GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Meta:  meta,
	}, nil
}

func encodeGraphNode(node Node) (*domainpb.GraphNode, error) {
	data, err := encodeGraphNodeData(node.Data)
	if err != nil {
		return nil, fmt.Errorf("encode node %q: %w", node.ID, err)
	}

	return &domainpb.GraphNode{
		Id:   node.ID,
		Type: node.Type,
		Data: data,
		Position: &domainpb.GraphPosition{
			X: node.Position.X,
			Y: node.Position.Y,
		},
	}, nil
}

func encodeGraphNodeData(data any) (*domainpb.GraphNodeData, error) {
	switch value := data.(type) {
	case GraphBacklogNodeData:
		return encodeGraphNodeData(&value)
	case *GraphBacklogNodeData:
		return encodeBacklogNodeData(value)
	case GraphInitiativeNodeData:
		return encodeGraphNodeData(&value)
	case *GraphInitiativeNodeData:
		return encodeInitiativeNodeData(value)
	case GraphCaptureNodeData:
		return encodeGraphNodeData(&value)
	case *GraphCaptureNodeData:
		return encodeCaptureNodeData(value)
	case GraphScenarioNodeData:
		return encodeGraphNodeData(&value)
	case *GraphScenarioNodeData:
		return encodeScenarioNodeData(value)
	case GraphExecutionNodeData:
		return encodeGraphNodeData(&value)
	case *GraphExecutionNodeData:
		return encodeExecutionNodeData(value)
	case GraphAgentActivityNodeData:
		return encodeGraphNodeData(&value)
	case *GraphAgentActivityNodeData:
		return encodeAgentActivityNodeData(value)
	case GraphRunNodeData:
		return encodeGraphNodeData(&value)
	case *GraphRunNodeData:
		return encodeRunNodeData(value)
	default:
		return nil, fmt.Errorf("unsupported node data type %T", data)
	}
}

func encodeBacklogNodeData(value *GraphBacklogNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing backlog node data")
	}
	msg := &domainpb.GraphBacklogNodeData{
		Kind:                 value.Kind,
		Name:                 value.Name,
		Title:                value.Title,
		Status:               value.Status,
		Priority:             value.Priority,
		ActiveExecutionCount: value.ActiveExecutionCount,
	}
	if value.ActiveExecutionStatus != "" {
		msg.ActiveExecutionStatus = proto.String(value.ActiveExecutionStatus)
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Backlog{
			Backlog: msg,
		},
	}, nil
}

func encodeInitiativeNodeData(value *GraphInitiativeNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing initiative node data")
	}
	initiative := &domainpb.GraphInitiativeNodeData{
		Name:   value.Name,
		Title:  value.Title,
		Status: value.Status,
		Rollup: &domainpb.GraphInitiativeRollup{
			Total:      value.Rollup.Total,
			Completed:  value.Rollup.Completed,
			InProgress: value.Rollup.InProgress,
			Failed:     value.Rollup.Failed,
			Pending:    value.Rollup.Pending,
		},
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Initiative{
			Initiative: initiative,
		},
	}, nil
}

func encodeCaptureNodeData(value *GraphCaptureNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing capture node data")
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Capture{
			Capture: &domainpb.GraphCaptureNodeData{
				Id:     value.ID,
				Text:   value.Text,
				Status: value.Status,
			},
		},
	}, nil
}

func encodeScenarioNodeData(value *GraphScenarioNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing scenario node data")
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Scenario{
			Scenario: &domainpb.GraphScenarioNodeData{
				Name:   value.Name,
				Status: value.Status,
			},
		},
	}, nil
}

func encodeExecutionNodeData(value *GraphExecutionNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing execution node data")
	}
	msg := &domainpb.GraphExecutionNodeData{
		ExecutionId: value.ExecutionID,
		BacklogKind: value.BacklogKind,
		BacklogName: value.BacklogName,
		Status:      value.Status,
		Mode:        value.Mode,
	}
	if value.RunID != "" {
		msg.RunId = proto.String(value.RunID)
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Execution{
			Execution: msg,
		},
	}, nil
}

func encodeAgentActivityNodeData(value *GraphAgentActivityNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing activity node data")
	}
	msg := &domainpb.GraphAgentActivityNodeData{
		ActivityId:      value.ActivityID,
		OwnerType:       value.OwnerType,
		OwnerName:       value.OwnerName,
		Purpose:         value.Purpose,
		InteractionType: value.InteractionType,
		Status:          value.Status,
		RequestedAt:     value.RequestedAt,
	}
	if value.OwnerKind != "" {
		msg.OwnerKind = proto.String(value.OwnerKind)
	}
	if value.OwnerTitle != "" {
		msg.OwnerTitle = proto.String(value.OwnerTitle)
	}
	if value.ExecutionID != "" {
		msg.ExecutionId = proto.String(value.ExecutionID)
	}
	if value.RunID != "" {
		msg.RunId = proto.String(value.RunID)
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Activity{
			Activity: msg,
		},
	}, nil
}

func encodeRunNodeData(value *GraphRunNodeData) (*domainpb.GraphNodeData, error) {
	if value == nil {
		return nil, fmt.Errorf("missing run node data")
	}
	msg := &domainpb.GraphRunNodeData{
		RunId:  value.RunID,
		Status: value.Status,
	}
	if value.TaskID != "" {
		msg.TaskId = proto.String(value.TaskID)
	}
	return &domainpb.GraphNodeData{
		Value: &domainpb.GraphNodeData_Run{
			Run: msg,
		},
	}, nil
}
