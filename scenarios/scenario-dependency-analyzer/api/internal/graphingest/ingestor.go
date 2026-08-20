package graphingest

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/interfacegraph"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"

	"github.com/vrooli/api-core/metrics"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// InterfaceGraphSource builds the import-evidence graph (proto_import + go_import).
// Satisfied by *interfacegraph.Builder; faked in tests.
type InterfaceGraphSource interface {
	BuildWithStats(ctx context.Context, req interfacegraph.BuildRequest) (interfacegraph.Graph, interfacegraph.BuildStats, error)
}

// DeclaredEdgeSource exposes the analyze-populated dependency rows (declared
// scenarios, vrooli_cli detections, resources). Satisfied by *store.Store.
type DeclaredEdgeSource interface {
	LoadAllDependencies() ([]types.ScenarioDependency, error)
	LoadStoredDependencies(scenario string) (map[string][]types.ScenarioDependency, error)
}

// AnalyzeSource refreshes the analyze-populated rows before they are read.
// Satisfied by *app.Analyzer.
type AnalyzeSource interface {
	AnalyzeScenario(scenario string) (*types.DependencyAnalysisResponse, error)
	AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error)
}

// Persister writes the merged unified edges. Satisfied by *store.Store.
type Persister interface {
	ReplaceGraphEdges(edges []types.UnifiedGraphEdge) error
	UpsertGraphEdgesForScenario(scenario string, edges []types.UnifiedGraphEdge) error
	MarkScenarioEdgesStale(scenario string) error
}

// Clock is the time seam for deterministic tests.
type Clock = TimeSource

type TimeSource interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// Ingestor merges all evidence sources into the unified graph store.
type Ingestor struct {
	graph    InterfaceGraphSource
	declared DeclaredEdgeSource
	analyze  AnalyzeSource
	persist  Persister
	clock    Clock
	env      *commonv1.CaptureEnvironment
}

// Option customizes an Ingestor.
type Option func(*Ingestor)

// WithClock overrides the time seam.
func WithClock(c Clock) Option { return func(i *Ingestor) { i.clock = c } }

// WithEnvironment attaches richer host facts to emitted ExecutionMetrics.
func WithEnvironment(env *commonv1.CaptureEnvironment) Option {
	return func(i *Ingestor) { i.env = env }
}

// NewIngestor constructs an Ingestor with explicit seams.
func NewIngestor(graph InterfaceGraphSource, declared DeclaredEdgeSource, analyze AnalyzeSource, persist Persister, opts ...Option) *Ingestor {
	ing := &Ingestor{
		graph:    graph,
		declared: declared,
		analyze:  analyze,
		persist:  persist,
		clock:    systemClock{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(ing)
		}
	}
	return ing
}

// FleetReport summarizes a full rebuild.
type FleetReport struct {
	ScenariosAnalyzed int                        `json:"scenarios_analyzed"`
	EdgesPersisted    int                        `json:"edges_persisted"`
	ScenarioEdges     int                        `json:"scenario_edges"`
	ResourceEdges     int                        `json:"resource_edges"`
	DegradedSources   []string                   `json:"degraded_sources,omitempty"`
	BuildStats        interfacegraph.BuildStats  `json:"build_stats"`
	Metrics           *commonv1.ExecutionMetrics `json:"-"`
}

// RebuildFleet refreshes analyze rows, builds the fleet interface graph, merges
// every source, and (when apply is true) atomically replaces graph_edges. It
// emits per-stage ExecutionMetrics so the sweeper budgets can be sized from
// measured cost. With apply=false it computes and reports the would-be result
// without persisting (dry-run).
func (ing *Ingestor) RebuildFleet(ctx context.Context, repoRoot string, apply bool) (FleetReport, error) {
	if ing == nil {
		return FleetReport{}, fmt.Errorf("ingestor not initialized")
	}
	report := FleetReport{}
	collector := metrics.Start(metrics.WithEnvironment(ing.env))

	if ing.analyze != nil {
		st := collector.Stage("analyze")
		results, err := ing.analyze.AnalyzeAllScenarios()
		report.ScenariosAnalyzed = len(results)
		st.Gauge("scenarios", float64(len(results)))
		st.End()
		if err != nil {
			collector.Stop()
			return report, fmt.Errorf("analyze fleet: %w", err)
		}
	}

	st := collector.Stage("interfacegraph")
	graph, stats, err := ing.graph.BuildWithStats(ctx, interfacegraph.BuildRequest{RepoRoot: repoRoot})
	st.Gauge("proto_fetch_ms", float64(stats.ProtoFetchMs))
	st.Gauge("import_fetch_ms", float64(stats.ImportFetchMs))
	st.Gauge("assemble_ms", float64(stats.AssembleMs))
	st.End()
	if err != nil {
		collector.Stop()
		return report, fmt.Errorf("build interface graph: %w", err)
	}
	report.BuildStats = stats
	report.DegradedSources = degradedSources(graph)

	stored, err := ing.declared.LoadAllDependencies()
	if err != nil {
		collector.Stop()
		return report, fmt.Errorf("load stored dependencies: %w", err)
	}

	mergeStage := collector.Stage("merge")
	contribs := contributionsFromInterfaceGraph(graph)
	contribs = append(contribs, contributionsFromStoredDeps(stored)...)
	edges := Merge(contribs, ing.clock.Now())
	mergeStage.Gauge("edges", float64(len(edges)))
	mergeStage.End()

	if apply {
		persistStage := collector.Stage("persist")
		if err := ing.persist.ReplaceGraphEdges(edges); err != nil {
			persistStage.End()
			collector.Stop()
			return report, fmt.Errorf("persist graph edges: %w", err)
		}
		persistStage.End()
	}

	report.Metrics = collector.Stop()
	report.EdgesPersisted = len(edges)
	for _, edge := range edges {
		if edge.Kind == KindResource {
			report.ResourceEdges++
		} else {
			report.ScenarioEdges++
		}
	}
	return report, nil
}

// ScenarioResult summarizes a single-scenario incremental ingest.
type ScenarioResult struct {
	Scenario       string
	EdgesPersisted int
	Degraded       bool
	BuildStats     interfacegraph.BuildStats
}

// IngestScenario re-ingests a single scenario's outbound edges. On a source
// outage it retains the scenario's last-good edges (marking them stale) rather
// than dropping them, so importance never flattens mid-outage. With apply=false
// it computes the would-be edges without persisting (dry-run); degradation
// marking only occurs on an applied run.
func (ing *Ingestor) IngestScenario(ctx context.Context, repoRoot, scenario string, apply bool) (ScenarioResult, error) {
	if ing == nil {
		return ScenarioResult{}, fmt.Errorf("ingestor not initialized")
	}
	result := ScenarioResult{Scenario: scenario}

	if ing.analyze != nil {
		// analyze failure is non-fatal: declared/resource contributions simply
		// fall back to whatever the store already holds.
		_, _ = ing.analyze.AnalyzeScenario(scenario)
	}

	graph, stats, err := ing.graph.BuildWithStats(ctx, interfacegraph.BuildRequest{
		RepoRoot:  repoRoot,
		Scenarios: []string{scenario},
	})
	result.BuildStats = stats
	if err != nil {
		// Upstream source down — keep last-good, flag stale.
		result.Degraded = true
		if apply {
			if markErr := ing.persist.MarkScenarioEdgesStale(scenario); markErr != nil {
				return result, fmt.Errorf("mark stale %s: %w", scenario, markErr)
			}
		}
		return result, fmt.Errorf("build interface graph for %s: %w", scenario, err)
	}

	buckets, err := ing.declared.LoadStoredDependencies(scenario)
	if err != nil {
		return result, fmt.Errorf("load stored dependencies for %s: %w", scenario, err)
	}
	var stored []types.ScenarioDependency
	for _, deps := range buckets {
		stored = append(stored, deps...)
	}

	contribs := contributionsFromInterfaceGraph(graph)
	contribs = append(contribs, contributionsFromStoredDeps(stored)...)
	// Only edges originating from this scenario belong to its upsert window.
	edges := filterFrom(Merge(contribs, ing.clock.Now()), scenario)
	if apply {
		if err := ing.persist.UpsertGraphEdgesForScenario(scenario, edges); err != nil {
			return result, fmt.Errorf("persist graph edges for %s: %w", scenario, err)
		}
	}
	result.EdgesPersisted = len(edges)
	return result, nil
}

func filterFrom(edges []types.UnifiedGraphEdge, scenario string) []types.UnifiedGraphEdge {
	out := edges[:0]
	for _, edge := range edges {
		if edge.From == scenario {
			out = append(out, edge)
		}
	}
	return out
}

func contributionsFromInterfaceGraph(graph interfacegraph.Graph) []Contribution {
	var contribs []Contribution
	for _, edge := range graph.Edges {
		for _, ev := range edge.Evidence {
			contribs = append(contribs, Contribution{
				From:   edge.FromScenario,
				To:     edge.ToScenario,
				Kind:   KindScenario,
				Source: ev.Source,
				Evidence: types.UnifiedEdgeEvidence{
					Source:     ev.Source,
					ImportPath: ev.ImportPath,
					FromFile:   ev.FromFile,
					ToFile:     ev.ToFile,
					Path:       ev.Path,
					Analyzer:   ev.Analyzer,
				},
			})
		}
	}
	return contribs
}

func contributionsFromStoredDeps(deps []types.ScenarioDependency) []Contribution {
	var contribs []Contribution
	for _, dep := range deps {
		switch dep.DependencyType {
		case "resource":
			contribs = append(contribs, Contribution{
				From:     dep.ScenarioName,
				To:       dep.DependencyName,
				Kind:     KindResource,
				Source:   SourceResource,
				Required: dep.Required,
				Evidence: types.UnifiedEdgeEvidence{Source: SourceResource, Detail: dep.AccessMethod},
			})
		case "scenario":
			source := SourceDeclared
			if dep.AccessMethod == "vrooli_cli" {
				source = SourceVrooliCLI
			}
			contribs = append(contribs, Contribution{
				From:     dep.ScenarioName,
				To:       dep.DependencyName,
				Kind:     KindScenario,
				Source:   source,
				Required: dep.Required,
				Evidence: types.UnifiedEdgeEvidence{Source: source, Path: dep.AccessMethod, Detail: dep.Purpose},
			})
		default:
			// shared_workflow / cli_tool are not part of the unified graph union.
		}
	}
	return contribs
}

func degradedSources(graph interfacegraph.Graph) []string {
	seen := map[string]struct{}{}
	for _, e := range graph.Errors {
		if e.Source != "" {
			seen[e.Source] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for source := range seen {
		out = append(out, source)
	}
	sort.Strings(out)
	return out
}
