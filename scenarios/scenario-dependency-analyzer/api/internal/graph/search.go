package graph

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/aisearch"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"
	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/store"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
)

// ConnectionSearcher is the read seam the SearchInterfaceGraph handler consumes:
// the multi-corpus aisearch service, queried against the .scenarios corpus. Held
// as an interface so the handler does not own the search service's lifecycle.
type ConnectionSearcher interface {
	SearchCorpus(ctx context.Context, corpus aisearch.CorpusID, query string, limit int) ([]aisearch.CorpusResult, error)
}

// SearchInterfaceGraph is the federated AI-search leaf over the connection graph.
// It is Connection-only (what depends on what); purpose/anatomy queries belong to
// architecture-cartographer's domain-map leaf.
func (h *connectHandler) SearchInterfaceGraph(ctx context.Context, req *connect.Request[graphv1.SearchInterfaceGraphRequest]) (*connect.Response[graphv1.SearchInterfaceGraphResponse], error) {
	if h == nil || h.searcher == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("interface graph search is not configured"))
	}
	msg := req.Msg
	if msg == nil {
		msg = &graphv1.SearchInterfaceGraphRequest{}
	}
	results, err := h.searcher.SearchCorpus(ctx, aisearch.CorpusScenarios, msg.GetQuery(), int(msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &graphv1.SearchInterfaceGraphResponse{
		Results: make([]*graphv1.ScenarioConnectionHit, 0, len(results)),
	}
	for _, r := range results {
		dependsOn := payloadStrings(r.Payload, "depends_on")
		usedBy := payloadStrings(r.Payload, "used_by")
		out.Results = append(out.Results, &graphv1.ScenarioConnectionHit{
			Scenario:       payloadString(r.Payload, "scenario", r.SourceID),
			DependsOn:      dependsOn,
			UsedBy:         usedBy,
			Summary:        connectionSummary(dependsOn, usedBy),
			RelevanceScore: r.Score,
		})
	}
	return connect.NewResponse(out), nil
}

// connectionSummary renders the federated snippet for a hit.
func connectionSummary(dependsOn, usedBy []string) string {
	depends := "(none)"
	if len(dependsOn) > 0 {
		depends = strings.Join(dependsOn, ", ")
	}
	used := "(none)"
	if len(usedBy) > 0 {
		used = strings.Join(usedBy, ", ")
	}
	return "Depends on: " + depends + ". Used by: " + used + "."
}

func payloadString(payload map[string]any, key, fallback string) string {
	if payload != nil {
		if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return fallback
}

// payloadStrings coerces a payload field to []string, tolerating both the text
// fallback's native []string and Qdrant's round-tripped []interface{}.
func payloadStrings(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	switch v := payload[key].(type) {
	case []string:
		return append([]string(nil), v...)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// ConnectionsProvider builds the .scenarios corpus's connection records from the
// live interface graph, reusing the store-backed cache. It implements
// aisearch.ScenarioGraphProvider.
type ConnectionsProvider struct {
	scenariosDir func() string
	store        *store.Store
	opts         ConnectOptions
}

// NewConnectionsProvider builds the connection provider. graphStore may be nil
// (the provider then always builds fresh).
func NewConnectionsProvider(scenariosDir func() string, graphStore *store.Store, opts ConnectOptions) *ConnectionsProvider {
	return &ConnectionsProvider{scenariosDir: scenariosDir, store: graphStore, opts: opts.withDefaults()}
}

// ScenarioConnections builds the whole-fleet interface graph and inverts its
// edges into per-scenario {depends_on, used_by} records.
func (p *ConnectionsProvider) ScenarioConnections(ctx context.Context) ([]aisearch.ScenarioConnection, error) {
	repoRoot := ""
	if p.scenariosDir != nil {
		repoRoot = filepath.Dir(strings.TrimSpace(p.scenariosDir()))
	}
	graph, _, err := loadInterfaceGraph(ctx, p.store, p.opts, interfacegraph.BuildRequest{RepoRoot: repoRoot})
	if err != nil {
		return nil, err
	}
	return invertGraph(graph), nil
}

// invertGraph turns the interface graph's nodes + directed edges into one
// connection record per scenario: depends_on = forward edges (from==s -> to),
// used_by = reverse edges (to==s -> from). Lists are sorted + deduplicated.
func invertGraph(graph interfacegraph.Graph) []aisearch.ScenarioConnection {
	dependsOn := map[string]map[string]struct{}{}
	usedBy := map[string]map[string]struct{}{}
	ensure := func(m map[string]map[string]struct{}, k string) map[string]struct{} {
		if m[k] == nil {
			m[k] = map[string]struct{}{}
		}
		return m[k]
	}
	scenarios := map[string]struct{}{}
	for _, n := range graph.Nodes {
		if s := strings.TrimSpace(n.Scenario); s != "" {
			scenarios[s] = struct{}{}
			ensure(dependsOn, s)
			ensure(usedBy, s)
		}
	}
	for _, e := range graph.Edges {
		from := strings.TrimSpace(e.FromScenario)
		to := strings.TrimSpace(e.ToScenario)
		if from == "" || to == "" || from == to {
			continue
		}
		scenarios[from] = struct{}{}
		scenarios[to] = struct{}{}
		ensure(dependsOn, from)[to] = struct{}{}
		ensure(usedBy, to)[from] = struct{}{}
		ensure(dependsOn, to)
		ensure(usedBy, from)
	}
	names := make([]string, 0, len(scenarios))
	for s := range scenarios {
		names = append(names, s)
	}
	sort.Strings(names)
	out := make([]aisearch.ScenarioConnection, 0, len(names))
	for _, s := range names {
		out = append(out, aisearch.ScenarioConnection{
			Scenario:  s,
			DependsOn: sortedKeys(dependsOn[s]),
			UsedBy:    sortedKeys(usedBy[s]),
		})
	}
	return out
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
