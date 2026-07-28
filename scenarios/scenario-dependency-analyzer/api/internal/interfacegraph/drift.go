package interfacegraph

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	appconfig "scenario-dependency-analyzer/internal/config"
)

type DriftDetector struct {
	builder      *Builder
	scenariosDir string
}

func NewDriftDetector(builder *Builder, scenariosDir string) *DriftDetector {
	return &DriftDetector{builder: builder, scenariosDir: scenariosDir}
}

func (d *DriftDetector) Detect(ctx context.Context, req BuildRequest) (DriftReport, error) {
	if d == nil || d.builder == nil {
		return DriftReport{}, fmt.Errorf("drift detector is not configured")
	}
	graph, err := d.builder.Build(ctx, req)
	if err != nil {
		return DriftReport{}, err
	}

	scenarios := req.Scenarios
	if len(scenarios) == 0 {
		scenarios = graphScenarios(graph)
	}
	sort.Strings(scenarios)
	actual := actualEdgesBySource(graph)

	findings := make([]DriftFinding, 0)
	for _, scenario := range scenarios {
		cfg, err := appconfig.LoadServiceConfig(filepath.Join(d.scenariosDir, scenario))
		if err != nil {
			// A graph node without a loadable service.json is not a real scenario
			// (e.g. a proto-package name that slipped through); skip it rather
			// than failing the whole drift report.
			continue
		}
		declared := map[string]struct{}{}
		for dep := range cfg.Dependencies.Scenarios {
			declared[dep] = struct{}{}
		}
		for dep, edge := range actual[scenario] {
			if _, ok := declared[dep]; ok {
				continue
			}
			findings = append(findings, DriftFinding{
				Scenario:       scenario,
				Dependency:     dep,
				Kind:           DriftUndeclaredUsed,
				Severity:       SeverityWarning,
				Message:        fmt.Sprintf("%s imports %s but does not declare it in service.json", scenario, dep),
				Evidence:       edge.Evidence,
				Declared:       false,
				ActualEvidence: true,
			})
		}
		for dep, spec := range cfg.Dependencies.Scenarios {
			if _, ok := actual[scenario][dep]; ok {
				continue
			}
			// Runtime-only dependencies are intentionally consumed through
			// lifecycle orchestration or an external service/CLI boundary,
			// not an import edge. The schema requires an explicit rationale
			// so this cannot become a blanket drift suppression.
			if spec.RuntimeOnly && spec.RuntimeOnlyRationale != "" {
				continue
			}
			findings = append(findings, DriftFinding{
				Scenario:       scenario,
				Dependency:     dep,
				Kind:           DriftDeclaredWithoutProof,
				Severity:       SeverityInfo,
				Message:        "no import-level evidence; runtime/CLI usage not yet analyzed (follow-up pending)",
				Declared:       true,
				ActualEvidence: false,
			})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Scenario != findings[j].Scenario {
			return findings[i].Scenario < findings[j].Scenario
		}
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity > findings[j].Severity
		}
		return findings[i].Dependency < findings[j].Dependency
	})
	return DriftReport{Graph: graph, Findings: findings, Scenarios: scenarios}, nil
}

func actualEdgesBySource(graph Graph) map[string]map[string]Edge {
	out := map[string]map[string]Edge{}
	for _, edge := range graph.Edges {
		if out[edge.FromScenario] == nil {
			out[edge.FromScenario] = map[string]Edge{}
		}
		out[edge.FromScenario][edge.ToScenario] = edge
	}
	return out
}

func graphScenarios(graph Graph) []string {
	seen := map[string]struct{}{}
	for _, node := range graph.Nodes {
		if node.Scenario != "" {
			seen[node.Scenario] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for scenario := range seen {
		out = append(out, scenario)
	}
	return out
}
