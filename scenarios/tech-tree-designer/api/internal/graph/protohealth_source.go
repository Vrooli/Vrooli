package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type ProtoHealthSource struct {
	client ProtoSurfaceClient
}

func NewProtoHealthSource(client ProtoSurfaceClient) *ProtoHealthSource {
	return &ProtoHealthSource{client: client}
}

func (s *ProtoHealthSource) Graph(ctx context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("proto-health graph source is not configured")
	}
	resp, err := s.client.DescribeScenariosProtos(ctx, ProtoSurfaceRequest{
		Scenarios:       req.ScenarioFilter,
		Limit:           req.Limit,
		StabilityFilter: req.StabilityFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("describe proto surfaces: %w", err)
	}
	return graphFromProtoSurfaces(resp), nil
}

func graphFromProtoSurfaces(resp *ProtoSurfaceResponse) *graphv1.TechTreeGraph {
	state := newGraphState()
	if resp == nil {
		return state.graph()
	}
	for _, result := range resp.Results {
		scenario := firstNonEmpty(result.Scenario, result.Surface.Scenario)
		if scenario == "" {
			continue
		}
		stabilityByPath := stabilityByPath(result.Surface)
		stabilities := uniqueStabilities(result.Surface.Files)
		state.addNode(&graphv1.TechNode{
			Scenario:       scenario,
			Kind:           graphv1.NodeKind_NODE_KIND_LIVE,
			DisplayName:    displayName(scenario),
			TransportWorld: result.Surface.TransportWorld,
			Stability:      stabilities,
		})
		if result.Error != "" {
			state.addError(&graphv1.GraphError{
				Source:   SourceProtoHealth,
				Scenario: scenario,
				Message:  result.Error,
			})
			continue
		}
		for _, imp := range result.Surface.CrossScenarioImports {
			from := firstNonEmpty(imp.FromScenario, scenario)
			to := strings.TrimSpace(imp.ToScenario)
			if from == "" || to == "" || from == to {
				continue
			}
			state.addNode(&graphv1.TechNode{
				Scenario:    to,
				Kind:        graphv1.NodeKind_NODE_KIND_LIVE,
				DisplayName: displayName(to),
			})
			state.addEvidence(from, to, &graphv1.GraphEvidence{
				Source:   graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT,
				FromFile: imp.FromFile,
				ToFile:   imp.ToFile,
			}, result.Surface.TransportWorld, firstNonEmpty(stabilityByPath[imp.FromFile], stabilityByPath[imp.ToFile]))
		}
	}
	return state.graph()
}

type graphState struct {
	nodes  map[string]*graphv1.TechNode
	edges  map[string]*graphv1.TechEdge
	errors []*graphv1.GraphError
}

func newGraphState() *graphState {
	return &graphState{
		nodes: map[string]*graphv1.TechNode{},
		edges: map[string]*graphv1.TechEdge{},
	}
}

func (s *graphState) addNode(node *graphv1.TechNode) {
	if node == nil {
		return
	}
	scenario := strings.TrimSpace(node.GetScenario())
	if scenario == "" {
		return
	}
	existing := s.nodes[scenario]
	if existing == nil {
		s.nodes[scenario] = &graphv1.TechNode{
			Scenario:       node.GetScenario(),
			Kind:           node.GetKind(),
			DisplayName:    node.GetDisplayName(),
			TransportWorld: node.GetTransportWorld(),
			Stability:      append([]string(nil), node.GetStability()...),
			Sector:         node.GetSector(),
			Tier:           node.GetTier(),
		}
		return
	}
	if existing.DisplayName == "" {
		existing.DisplayName = node.GetDisplayName()
	}
	if existing.Kind == graphv1.NodeKind_NODE_KIND_UNSPECIFIED {
		existing.Kind = node.GetKind()
	}
	if existing.TransportWorld == "" {
		existing.TransportWorld = node.GetTransportWorld()
	}
	for _, stability := range node.GetStability() {
		if stability != "" && !contains(existing.Stability, stability) {
			existing.Stability = append(existing.Stability, stability)
		}
	}
	sort.Strings(existing.Stability)
}

func (s *graphState) addError(err *graphv1.GraphError) {
	if err != nil {
		s.errors = append(s.errors, err)
	}
}

func (s *graphState) addEvidence(from, to string, ev *graphv1.GraphEvidence, transportWorld, stability string) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return
	}
	key := from + "\x00" + to
	edge := s.edges[key]
	if edge == nil {
		edge = &graphv1.TechEdge{FromScenario: from, ToScenario: to}
		s.edges[key] = edge
	}
	if ev != nil && !hasEvidence(edge.Evidence, ev) {
		edge.Evidence = append(edge.Evidence, ev)
	}
	if edge.TransportWorld == "" && transportWorld != "" {
		edge.TransportWorld = transportWorld
	}
	if stability != "" && !contains(edge.Stability, stability) {
		edge.Stability = append(edge.Stability, stability)
		sort.Strings(edge.Stability)
	}
}

func (s *graphState) graph() *graphv1.TechTreeGraph {
	nodes := make([]*graphv1.TechNode, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].GetScenario() < nodes[j].GetScenario() })

	edges := make([]*graphv1.TechEdge, 0, len(s.edges))
	for _, edge := range s.edges {
		sort.Slice(edge.Evidence, func(i, j int) bool {
			if edge.Evidence[i].GetSource() != edge.Evidence[j].GetSource() {
				return edge.Evidence[i].GetSource() < edge.Evidence[j].GetSource()
			}
			if edge.Evidence[i].GetFromFile() != edge.Evidence[j].GetFromFile() {
				return edge.Evidence[i].GetFromFile() < edge.Evidence[j].GetFromFile()
			}
			return edge.Evidence[i].GetToFile() < edge.Evidence[j].GetToFile()
		})
		edges = append(edges, edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].GetFromScenario() != edges[j].GetFromScenario() {
			return edges[i].GetFromScenario() < edges[j].GetFromScenario()
		}
		return edges[i].GetToScenario() < edges[j].GetToScenario()
	})

	sort.Slice(s.errors, func(i, j int) bool {
		if s.errors[i].GetSource() != s.errors[j].GetSource() {
			return s.errors[i].GetSource() < s.errors[j].GetSource()
		}
		return s.errors[i].GetScenario() < s.errors[j].GetScenario()
	})
	return &graphv1.TechTreeGraph{Nodes: nodes, Edges: edges, Errors: s.errors}
}

func stabilityByPath(surface ProtoSurface) map[string]string {
	out := make(map[string]string, len(surface.Files))
	for _, file := range surface.Files {
		if file.Path != "" && file.Stability != "" {
			out[file.Path] = file.Stability
		}
	}
	return out
}

func uniqueStabilities(files []ProtoFile) []string {
	seen := map[string]struct{}{}
	for _, file := range files {
		stability := strings.TrimSpace(file.Stability)
		if stability != "" {
			seen[stability] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for stability := range seen {
		out = append(out, stability)
	}
	sort.Strings(out)
	return out
}

func normalizeTransportWorld(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "transport_world_")
	return value
}

func displayName(scenario string) string {
	parts := strings.Split(strings.TrimSpace(scenario), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasEvidence(existing []*graphv1.GraphEvidence, next *graphv1.GraphEvidence) bool {
	for _, ev := range existing {
		if ev.GetSource() == next.GetSource() &&
			ev.GetImportPath() == next.GetImportPath() &&
			ev.GetFromFile() == next.GetFromFile() &&
			ev.GetToFile() == next.GetToFile() &&
			ev.GetPath() == next.GetPath() &&
			ev.GetAnalyzer() == next.GetAnalyzer() {
			return true
		}
	}
	return false
}
