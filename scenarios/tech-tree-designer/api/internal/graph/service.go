package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type Service struct {
	source  GraphSource
	planned PlannedGraphSource
}

func NewService(source GraphSource) *Service {
	return &Service{source: source}
}

func NewServiceWithPlanned(source GraphSource, planned PlannedGraphSource) *Service {
	return &Service{source: source, planned: planned}
}

func (s *Service) Describe(ctx context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error) {
	graph, err := s.sourceGraph(ctx, req)
	if graph == nil {
		graph = &graphv1.TechTreeGraph{}
	}
	if err != nil {
		graph.Errors = append(graph.Errors, &graphv1.GraphError{
			Source:  SourceProtoHealth,
			Message: err.Error(),
		})
	}
	if s != nil && s.planned != nil {
		plannedGraph, plannedErr := s.planned.PlannedGraph(ctx)
		if plannedErr != nil {
			graph.Errors = append(graph.Errors, &graphv1.GraphError{
				Source:  "planning",
				Message: plannedErr.Error(),
			})
		} else {
			graph = mergeGraphs(graph, plannedGraph)
		}
	}
	return graph, nil
}

func (s *Service) Neighborhood(ctx context.Context, scenario string, depth int32, filter []string) (*graphv1.TechTreeGraph, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	if depth <= 0 {
		depth = 1
	}
	graph, err := s.Describe(ctx, SourceRequest{ScenarioFilter: filter})
	if err != nil {
		return nil, err
	}
	return inducedSubgraph(graph, neighborhoodSet(graph, scenario, int(depth))), nil
}

func (s *Service) Path(ctx context.Context, from, to string, filter []string) (*graphv1.TechTreeGraph, error) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return nil, fmt.Errorf("from_scenario and to_scenario are required")
	}
	graph, err := s.Describe(ctx, SourceRequest{ScenarioFilter: filter})
	if err != nil {
		return nil, err
	}
	path := shortestPath(graph, from, to)
	if len(path) == 0 {
		return &graphv1.TechTreeGraph{Errors: graph.GetErrors()}, nil
	}
	keep := map[string]struct{}{}
	for _, scenario := range path {
		keep[scenario] = struct{}{}
	}
	return pathSubgraph(graph, path, keep), nil
}

func (s *Service) Ancestors(ctx context.Context, scenario string, filter []string) (*graphv1.TechTreeGraph, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return nil, fmt.Errorf("scenario is required")
	}
	graph, err := s.Describe(ctx, SourceRequest{ScenarioFilter: filter})
	if err != nil {
		return nil, err
	}
	keep := reachableDependencies(graph, scenario)
	if len(keep) == 0 {
		keep[scenario] = struct{}{}
	}
	return inducedSubgraph(graph, keep), nil
}

func (s *Service) Export(ctx context.Context, req SourceRequest, format graphv1.ExportFormat) (*graphv1.ExportTechTreeResponse, error) {
	graph, err := s.Describe(ctx, req)
	if err != nil {
		return nil, err
	}
	switch format {
	case graphv1.ExportFormat_EXPORT_FORMAT_UNSPECIFIED:
		format = graphv1.ExportFormat_EXPORT_FORMAT_TEXT
	case graphv1.ExportFormat_EXPORT_FORMAT_DOT,
		graphv1.ExportFormat_EXPORT_FORMAT_JSON,
		graphv1.ExportFormat_EXPORT_FORMAT_TEXT:
	default:
		return nil, fmt.Errorf("unsupported export format %s", format.String())
	}
	content, mediaType, err := exportGraph(graph, format)
	if err != nil {
		return nil, err
	}
	return &graphv1.ExportTechTreeResponse{
		Format:    format,
		Content:   content,
		MediaType: mediaType,
	}, nil
}

func (s *Service) sourceGraph(ctx context.Context, req SourceRequest) (*graphv1.TechTreeGraph, error) {
	if s == nil || s.source == nil {
		return nil, fmt.Errorf("graph source is not configured")
	}
	return s.source.Graph(ctx, req)
}

func neighborhoodSet(graph *graphv1.TechTreeGraph, root string, depth int) map[string]struct{} {
	keep := map[string]struct{}{root: {}}
	frontier := []string{root}
	for i := 0; i < depth && len(frontier) > 0; i++ {
		nextSet := map[string]struct{}{}
		for _, edge := range graph.GetEdges() {
			for _, scenario := range frontier {
				if edge.GetFromScenario() == scenario {
					if _, seen := keep[edge.GetToScenario()]; !seen {
						nextSet[edge.GetToScenario()] = struct{}{}
					}
				}
				if edge.GetToScenario() == scenario {
					if _, seen := keep[edge.GetFromScenario()]; !seen {
						nextSet[edge.GetFromScenario()] = struct{}{}
					}
				}
			}
		}
		frontier = frontier[:0]
		for scenario := range nextSet {
			keep[scenario] = struct{}{}
			frontier = append(frontier, scenario)
		}
		sort.Strings(frontier)
	}
	return keep
}

func shortestPath(graph *graphv1.TechTreeGraph, from, to string) []string {
	if from == to {
		return []string{from}
	}
	adj := outgoingAdjacency(graph)
	queue := []string{from}
	prev := map[string]string{from: ""}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if _, seen := prev[next]; seen {
				continue
			}
			prev[next] = cur
			if next == to {
				return reconstructPath(prev, from, to)
			}
			queue = append(queue, next)
		}
	}
	return nil
}

func reconstructPath(prev map[string]string, from, to string) []string {
	var path []string
	for cur := to; cur != ""; cur = prev[cur] {
		path = append(path, cur)
		if cur == from {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	if len(path) == 0 || path[0] != from {
		return nil
	}
	return path
}

func reachableDependencies(graph *graphv1.TechTreeGraph, root string) map[string]struct{} {
	keep := map[string]struct{}{root: {}}
	adj := outgoingAdjacency(graph)
	queue := append([]string(nil), adj[root]...)
	for _, scenario := range queue {
		keep[scenario] = struct{}{}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if _, seen := keep[next]; seen {
				continue
			}
			keep[next] = struct{}{}
			queue = append(queue, next)
		}
	}
	return keep
}

func outgoingAdjacency(graph *graphv1.TechTreeGraph) map[string][]string {
	adj := map[string][]string{}
	for _, edge := range graph.GetEdges() {
		from := edge.GetFromScenario()
		to := edge.GetToScenario()
		if from == "" || to == "" {
			continue
		}
		if !contains(adj[from], to) {
			adj[from] = append(adj[from], to)
		}
	}
	for from := range adj {
		sort.Strings(adj[from])
	}
	return adj
}

func inducedSubgraph(graph *graphv1.TechTreeGraph, keep map[string]struct{}) *graphv1.TechTreeGraph {
	out := &graphv1.TechTreeGraph{Errors: append([]*graphv1.GraphError(nil), graph.GetErrors()...)}
	for _, node := range graph.GetNodes() {
		if _, ok := keep[node.GetScenario()]; ok {
			out.Nodes = append(out.Nodes, node)
		}
	}
	for _, edge := range graph.GetEdges() {
		_, fromOK := keep[edge.GetFromScenario()]
		_, toOK := keep[edge.GetToScenario()]
		if fromOK && toOK {
			out.Edges = append(out.Edges, edge)
		}
	}
	return out
}

func pathSubgraph(graph *graphv1.TechTreeGraph, path []string, keep map[string]struct{}) *graphv1.TechTreeGraph {
	out := &graphv1.TechTreeGraph{Errors: append([]*graphv1.GraphError(nil), graph.GetErrors()...)}
	for _, node := range graph.GetNodes() {
		if _, ok := keep[node.GetScenario()]; ok {
			out.Nodes = append(out.Nodes, node)
		}
	}
	pathEdges := map[string]struct{}{}
	for i := 0; i+1 < len(path); i++ {
		pathEdges[path[i]+"\x00"+path[i+1]] = struct{}{}
	}
	for _, edge := range graph.GetEdges() {
		if _, ok := pathEdges[edge.GetFromScenario()+"\x00"+edge.GetToScenario()]; ok {
			out.Edges = append(out.Edges, edge)
		}
	}
	return out
}

func exportGraph(graph *graphv1.TechTreeGraph, format graphv1.ExportFormat) (string, string, error) {
	switch format {
	case graphv1.ExportFormat_EXPORT_FORMAT_DOT:
		return exportDOT(graph), "text/vnd.graphviz", nil
	case graphv1.ExportFormat_EXPORT_FORMAT_JSON:
		body, err := protojson.MarshalOptions{UseProtoNames: true, Multiline: true, Indent: "  "}.Marshal(graph)
		if err != nil {
			return "", "", fmt.Errorf("marshal graph json: %w", err)
		}
		return string(body) + "\n", "application/json", nil
	case graphv1.ExportFormat_EXPORT_FORMAT_TEXT:
		return exportText(graph), "text/plain", nil
	default:
		return "", "", fmt.Errorf("unsupported export format %s", format.String())
	}
}

func mergeGraphs(graphs ...*graphv1.TechTreeGraph) *graphv1.TechTreeGraph {
	state := newGraphState()
	for _, graph := range graphs {
		if graph == nil {
			continue
		}
		for _, node := range graph.GetNodes() {
			state.addNode(node)
		}
		for _, edge := range graph.GetEdges() {
			state.addNode(&graphv1.TechNode{
				Scenario:    edge.GetFromScenario(),
				DisplayName: displayName(edge.GetFromScenario()),
			})
			state.addNode(&graphv1.TechNode{
				Scenario:    edge.GetToScenario(),
				DisplayName: displayName(edge.GetToScenario()),
			})
			for _, ev := range edge.GetEvidence() {
				state.addEvidence(edge.GetFromScenario(), edge.GetToScenario(), ev, edge.GetTransportWorld(), "")
			}
			for _, stability := range edge.GetStability() {
				state.addEvidence(edge.GetFromScenario(), edge.GetToScenario(), nil, edge.GetTransportWorld(), stability)
			}
		}
		for _, graphErr := range graph.GetErrors() {
			state.addError(graphErr)
		}
	}
	return state.graph()
}

func exportDOT(graph *graphv1.TechTreeGraph) string {
	var b strings.Builder
	b.WriteString("digraph tech_tree {\n")
	for _, node := range graph.GetNodes() {
		fmt.Fprintf(&b, "  %s [label=%s];\n", jsonQuote(node.GetScenario()), jsonQuote(firstNonEmpty(node.GetDisplayName(), node.GetScenario())))
	}
	for _, edge := range graph.GetEdges() {
		fmt.Fprintf(&b, "  %s -> %s;\n", jsonQuote(edge.GetFromScenario()), jsonQuote(edge.GetToScenario()))
	}
	b.WriteString("}\n")
	return b.String()
}

func exportText(graph *graphv1.TechTreeGraph) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tech Tree Graph: %d node(s), %d edge(s)\n", len(graph.GetNodes()), len(graph.GetEdges()))
	for _, node := range graph.GetNodes() {
		fmt.Fprintf(&b, "- %s [%s %s]\n", node.GetScenario(), node.GetTransportWorld(), strings.Join(node.GetStability(), ","))
	}
	for _, edge := range graph.GetEdges() {
		fmt.Fprintf(&b, "%s -> %s\n", edge.GetFromScenario(), edge.GetToScenario())
	}
	for _, graphErr := range graph.GetErrors() {
		fmt.Fprintf(&b, "! %s %s: %s\n", graphErr.GetSource(), graphErr.GetScenario(), graphErr.GetMessage())
	}
	return b.String()
}

func jsonQuote(value string) string {
	body, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(body)
}
