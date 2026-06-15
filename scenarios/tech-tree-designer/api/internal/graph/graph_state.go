package graph

import (
	"sort"
	"strings"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

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

func hasEvidence(values []*graphv1.GraphEvidence, needle *graphv1.GraphEvidence) bool {
	for _, value := range values {
		if value.GetSource() == needle.GetSource() &&
			value.GetImportPath() == needle.GetImportPath() &&
			value.GetFromFile() == needle.GetFromFile() &&
			value.GetToFile() == needle.GetToFile() &&
			value.GetPath() == needle.GetPath() &&
			value.GetAnalyzer() == needle.GetAnalyzer() {
			return true
		}
	}
	return false
}
