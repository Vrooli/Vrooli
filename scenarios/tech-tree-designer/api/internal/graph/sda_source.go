package graph

import (
	"context"
	"fmt"

	sdagraphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type SDASource struct {
	client SDAInterfaceGraphClient
}

func NewSDASource(client SDAInterfaceGraphClient) *SDASource {
	return &SDASource{client: client}
}

func (s *SDASource) Graph(ctx context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("scenario-dependency-analyzer graph source is not configured")
	}
	resp, err := s.client.DescribeInterfaceGraph(ctx, SDAInterfaceGraphRequest{
		Scenarios:       req.ScenarioFilter,
		Limit:           req.Limit,
		StabilityFilter: req.StabilityFilter,
		MaxScenarioHops: req.MaxScenarioHops,
	})
	if err != nil {
		return nil, fmt.Errorf("describe SDA interface graph: %w", err)
	}
	return graphFromSDA(resp), nil
}

func graphFromSDA(resp *SDAInterfaceGraphResponse) *graphv1.TechTreeGraph {
	state := newGraphState()
	if resp == nil || resp.Graph == nil {
		return state.graph()
	}
	for _, node := range resp.Graph.GetNodes() {
		scenario := node.GetScenario()
		state.addNode(&graphv1.TechNode{
			Scenario:    scenario,
			Kind:        graphv1.NodeKind_NODE_KIND_LIVE,
			DisplayName: displayName(scenario),
		})
	}
	for _, edge := range resp.Graph.GetEdges() {
		state.addNode(&graphv1.TechNode{
			Scenario:    edge.GetFromScenario(),
			Kind:        graphv1.NodeKind_NODE_KIND_LIVE,
			DisplayName: displayName(edge.GetFromScenario()),
		})
		state.addNode(&graphv1.TechNode{
			Scenario:    edge.GetToScenario(),
			Kind:        graphv1.NodeKind_NODE_KIND_LIVE,
			DisplayName: displayName(edge.GetToScenario()),
		})
		for _, evidence := range edge.GetEvidence() {
			state.addEvidence(edge.GetFromScenario(), edge.GetToScenario(), evidenceFromSDA(evidence), normalizeTransportWorld(edge.GetTransportWorld()), "")
		}
		for _, stability := range edge.GetStability() {
			state.addEvidence(edge.GetFromScenario(), edge.GetToScenario(), nil, normalizeTransportWorld(edge.GetTransportWorld()), stability)
		}
	}
	for _, graphErr := range resp.Graph.GetErrors() {
		state.addError(&graphv1.GraphError{
			Source:   firstNonEmpty(graphErr.GetSource(), SourceSDA),
			Scenario: graphErr.GetScenario(),
			Message:  graphErr.GetMessage(),
		})
	}
	return state.graph()
}

func evidenceFromSDA(in *sdagraphv1.GraphEvidence) *graphv1.GraphEvidence {
	if in == nil {
		return nil
	}
	return &graphv1.GraphEvidence{
		Source:     evidenceSourceFromSDA(in.GetSource()),
		ImportPath: in.GetImportPath(),
		FromFile:   in.GetFromFile(),
		ToFile:     in.GetToFile(),
		Path:       in.GetPath(),
		Analyzer:   in.GetAnalyzer(),
	}
}

func evidenceSourceFromSDA(source sdagraphv1.EvidenceSource) graphv1.EvidenceSource {
	switch source {
	case sdagraphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT
	case sdagraphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT
	default:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_UNSPECIFIED
	}
}
