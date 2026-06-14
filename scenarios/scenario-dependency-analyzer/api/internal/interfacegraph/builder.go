package interfacegraph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Builder struct {
	protos  ProtoSurfaceClient
	imports ImportFactsClient
}

type BuildRequest struct {
	Scenarios       []string
	Limit           int32
	RepoRoot        string
	StabilityFilter string
	LanguageFilter  []string
}

func NewBuilder(protos ProtoSurfaceClient, imports ImportFactsClient) *Builder {
	return &Builder{protos: protos, imports: imports}
}

func (b *Builder) Build(ctx context.Context, req BuildRequest) (Graph, error) {
	if b == nil {
		return Graph{}, fmt.Errorf("interface graph builder is nil")
	}
	if b.protos == nil {
		return Graph{}, fmt.Errorf("proto surface client is not configured")
	}
	if b.imports == nil {
		return Graph{}, fmt.Errorf("import facts client is not configured")
	}

	protoResp, err := b.protos.DescribeScenariosProtos(ctx, ProtoSurfaceRequest{
		Scenarios:       req.Scenarios,
		Limit:           req.Limit,
		StabilityFilter: req.StabilityFilter,
	})
	if err != nil {
		return Graph{}, fmt.Errorf("describe proto surfaces: %w", err)
	}
	importResp, err := b.imports.DescribeFleetImports(ctx, ImportFactsRequest{
		Scenarios:      req.Scenarios,
		Limit:          req.Limit,
		RepoRoot:       req.RepoRoot,
		LanguageFilter: req.LanguageFilter,
		UseCache:       true,
	})
	if err != nil {
		return Graph{}, fmt.Errorf("describe fleet imports: %w", err)
	}

	state := newGraphState()
	attributor := NewAttributor(req.Scenarios)
	for _, scenario := range scenariosFromRepoRoot(req.RepoRoot) {
		attributor.AddScenario(scenario)
	}
	surfaceByScenario := map[string]ProtoSurface{}
	for _, result := range protoResp.Results {
		scenario := firstNonEmpty(result.Scenario, result.Surface.Scenario)
		if scenario == "" {
			continue
		}
		attributor.AddScenario(scenario)
		state.addNode(scenario)
		if result.Error != "" {
			state.addError("proto-health", scenario, result.Error)
			continue
		}
		surface := result.Surface
		if surface.TransportWorld == "" {
			surface.TransportWorld = result.TransportWorld
		}
		surfaceByScenario[scenario] = surface
	}
	for _, result := range importResp.Results {
		if result.Scenario != "" {
			attributor.AddScenario(result.Scenario)
			state.addNode(result.Scenario)
		}
		if result.Error != "" {
			state.addError("code-facts", result.Scenario, result.Error)
		}
	}

	for scenario, surface := range surfaceByScenario {
		stabilityByPath := protoStabilityByPath(surface)
		for _, imp := range surface.CrossScenarioImports {
			from := firstNonEmpty(imp.FromScenario, scenario)
			to := imp.ToScenario
			if to == "" || from == "" || from == to {
				continue
			}
			state.addEvidence(from, to, Evidence{
				Source:   EvidenceProtoImport,
				FromFile: imp.FromFile,
				ToFile:   imp.ToFile,
			}, normalizeTransport(surface.TransportWorld), stabilityByPath[imp.ToFile])
		}
	}

	for _, result := range importResp.Results {
		if result.Error != "" {
			continue
		}
		for _, fact := range result.Facts {
			to, ok := attributor.Attribute(fact.ImportPath)
			if !ok || to == "" || to == result.Scenario {
				continue
			}
			state.addEvidence(result.Scenario, to, Evidence{
				Source:     EvidenceGoImport,
				ImportPath: fact.ImportPath,
				Path:       fact.Path,
				Analyzer:   fact.Analyzer,
			}, "", "")
		}
	}

	return state.graph(), nil
}

func scenariosFromRepoRoot(repoRoot string) []string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return nil
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		if _, err := os.Stat(filepath.Join(repoRoot, "scenarios", scenario, ".vrooli", "service.json")); err == nil {
			out = append(out, scenario)
		}
	}
	sort.Strings(out)
	return out
}

type graphState struct {
	nodes  map[string]struct{}
	edges  map[string]*Edge
	errors []Error
}

func newGraphState() *graphState {
	return &graphState{
		nodes: map[string]struct{}{},
		edges: map[string]*Edge{},
	}
}

func (s *graphState) addNode(scenario string) {
	scenario = strings.TrimSpace(scenario)
	if scenario != "" {
		s.nodes[scenario] = struct{}{}
	}
}

func (s *graphState) addError(source, scenario, message string) {
	s.errors = append(s.errors, Error{Source: source, Scenario: scenario, Message: message})
}

func (s *graphState) addEvidence(from, to string, ev Evidence, transportWorld, stability string) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if from == "" || to == "" {
		return
	}
	s.addNode(from)
	s.addNode(to)
	key := from + "\x00" + to
	edge := s.edges[key]
	if edge == nil {
		edge = &Edge{FromScenario: from, ToScenario: to}
		s.edges[key] = edge
	}
	if !hasEvidence(edge.Evidence, ev) {
		edge.Evidence = append(edge.Evidence, ev)
	}
	if edge.TransportWorld == "" && transportWorld != "" {
		edge.TransportWorld = transportWorld
	}
	if stability != "" && !containsString(edge.Stability, stability) {
		edge.Stability = append(edge.Stability, stability)
		sort.Strings(edge.Stability)
	}
}

func (s *graphState) graph() Graph {
	nodes := make([]Node, 0, len(s.nodes))
	for scenario := range s.nodes {
		nodes = append(nodes, Node{Scenario: scenario})
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Scenario < nodes[j].Scenario })

	edges := make([]Edge, 0, len(s.edges))
	for _, edge := range s.edges {
		sort.Slice(edge.Evidence, func(i, j int) bool {
			if edge.Evidence[i].Source != edge.Evidence[j].Source {
				return edge.Evidence[i].Source < edge.Evidence[j].Source
			}
			return edge.Evidence[i].ImportPath < edge.Evidence[j].ImportPath
		})
		edges = append(edges, *edge)
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromScenario != edges[j].FromScenario {
			return edges[i].FromScenario < edges[j].FromScenario
		}
		return edges[i].ToScenario < edges[j].ToScenario
	})

	sort.Slice(s.errors, func(i, j int) bool {
		if s.errors[i].Source != s.errors[j].Source {
			return s.errors[i].Source < s.errors[j].Source
		}
		return s.errors[i].Scenario < s.errors[j].Scenario
	})
	return Graph{Nodes: nodes, Edges: edges, Errors: s.errors}
}

func protoStabilityByPath(surface ProtoSurface) map[string]string {
	out := make(map[string]string, len(surface.Files))
	for _, file := range surface.Files {
		if file.Path != "" && file.Stability != "" {
			out[file.Path] = file.Stability
		}
	}
	return out
}

func normalizeTransport(in string) string {
	in = strings.TrimSpace(strings.ToLower(in))
	in = strings.TrimPrefix(in, "transport_world_")
	return in
}

func hasEvidence(existing []Evidence, next Evidence) bool {
	for _, ev := range existing {
		if ev == next {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
