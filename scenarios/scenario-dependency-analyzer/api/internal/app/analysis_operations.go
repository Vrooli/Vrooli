package app

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/analysisoperations"
)

// Fleet operations reuse the analyzer and graph owner while retaining only one
// detailed response at a time. The durable domain stores bounded summaries.
type fleetOperationRunner struct {
	analyzer *Analyzer
	graph    *graphIngestService
}

func (r fleetOperationRunner) Scenarios(ctx context.Context) ([]string, error) {
	return r.analyzer.ScenarioNames(ctx)
}
func (r fleetOperationRunner) Analyze(ctx context.Context, name string) (analysisoperations.ScenarioOutcome, error) {
	response, err := r.analyzer.AnalyzeScenarioContext(ctx, name)
	if err != nil {
		return analysisoperations.ScenarioOutcome{}, err
	}
	result := analysisoperations.ScenarioOutcome{DetectedResources: len(response.DetectedResources), ScenarioDependencies: len(response.Scenarios)}
	if response.DeploymentReport != nil {
		result.DeploymentDependencies = len(response.DeploymentReport.Dependencies)
	}
	return result, nil
}

func (r fleetOperationRunner) RebuildGraph(ctx context.Context, apply bool) (analysisoperations.GraphOutcome, error) {
	if r.graph == nil || r.graph.ingestor == nil {
		return analysisoperations.GraphOutcome{}, fmt.Errorf("graph ingest unavailable")
	}
	report, err := r.graph.ingestor.RebuildStoredFleet(ctx, r.graph.repoRoot, apply)
	return analysisoperations.GraphOutcome{Applied: apply && err == nil, Edges: report.EdgesPersisted, ScenarioEdges: report.ScenarioEdges, ResourceEdges: report.ResourceEdges, DegradedSources: report.DegradedSources}, err
}
