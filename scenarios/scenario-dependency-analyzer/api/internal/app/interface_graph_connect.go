package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/interfacegraph"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph/graph_v1connect"
)

type interfaceGraphConnectHandler struct {
	handler *handler
}

func registerInterfaceGraphConnectRoutes(router *gin.Engine, h *handler) {
	connectPath, connectHandler := graphconnect.NewInterfaceGraphServiceHandler(&interfaceGraphConnectHandler{handler: h})
	router.Any(connectPath+"*path", gin.WrapH(connectHandler))
}

func (h *interfaceGraphConnectHandler) DescribeInterfaceGraph(ctx context.Context, req *connect.Request[graphv1.DescribeInterfaceGraphRequest]) (*connect.Response[graphv1.DescribeInterfaceGraphResponse], error) {
	if h == nil || h.handler == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("interface graph handler is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &graphv1.DescribeInterfaceGraphRequest{}
	}
	graph, err := h.handler.describeInterfaceGraph(ctx, interfacegraph.BuildRequest{
		Scenarios:       msg.GetScenarios(),
		Limit:           msg.GetLimit(),
		RepoRoot:        filepath.Dir(h.handler.scenariosDir()),
		StabilityFilter: msg.GetStabilityFilter(),
		LanguageFilter:  msg.GetLanguageFilter(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("describe interface graph: %w", err))
	}
	return connect.NewResponse(&graphv1.DescribeInterfaceGraphResponse{
		Graph: interfaceGraphToProto(graph),
	}), nil
}

func (h *handler) describeInterfaceGraph(ctx context.Context, req interfacegraph.BuildRequest) (interfacegraph.Graph, error) {
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	return builder.Build(ctx, req)
}

func interfaceGraphToProto(in interfacegraph.Graph) *graphv1.InterfaceGraph {
	out := &graphv1.InterfaceGraph{
		Nodes:  make([]*graphv1.GraphNode, 0, len(in.Nodes)),
		Edges:  make([]*graphv1.GraphEdge, 0, len(in.Edges)),
		Errors: make([]*graphv1.GraphError, 0, len(in.Errors)),
	}
	for _, node := range in.Nodes {
		out.Nodes = append(out.Nodes, &graphv1.GraphNode{Scenario: node.Scenario})
	}
	for _, edge := range in.Edges {
		item := &graphv1.GraphEdge{
			FromScenario:   edge.FromScenario,
			ToScenario:     edge.ToScenario,
			TransportWorld: edge.TransportWorld,
			Stability:      append([]string(nil), edge.Stability...),
			Evidence:       make([]*graphv1.GraphEvidence, 0, len(edge.Evidence)),
		}
		for _, evidence := range edge.Evidence {
			item.Evidence = append(item.Evidence, evidenceToProto(evidence))
		}
		out.Edges = append(out.Edges, item)
	}
	for _, graphErr := range in.Errors {
		out.Errors = append(out.Errors, &graphv1.GraphError{
			Source:   graphErr.Source,
			Scenario: graphErr.Scenario,
			Message:  graphErr.Message,
		})
	}
	return out
}

func evidenceToProto(in interfacegraph.Evidence) *graphv1.GraphEvidence {
	return &graphv1.GraphEvidence{
		Source:     evidenceSourceToProto(in.Source),
		ImportPath: in.ImportPath,
		FromFile:   in.FromFile,
		ToFile:     in.ToFile,
		Path:       in.Path,
		Analyzer:   in.Analyzer,
	}
}

func evidenceSourceToProto(source string) graphv1.EvidenceSource {
	switch source {
	case interfacegraph.EvidenceProtoImport:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT
	case interfacegraph.EvidenceGoImport:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT
	default:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_UNSPECIFIED
	}
}

var _ graphconnect.InterfaceGraphServiceHandler = (*interfaceGraphConnectHandler)(nil)
