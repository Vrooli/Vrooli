package app

import (
	"errors"

	"scenario-dependency-analyzer/internal/app/services"
)

var errScenarioNotFound = errors.New("scenario not found")

func newServices(analyzer *Analyzer, workspace *scenarioWorkspace) services.Registry {
	if workspace == nil {
		workspace = newScenarioWorkspace(analyzer.cfg)
	}
	analysis := &analysisService{analyzer: analyzer}
	dependencies := newDependencyService(analyzer.store, analyzer.detector)
	scan := &scanService{analysis: analysis, dependencies: dependencies}
	optimization := &optimizationService{
		analysis:     analysis,
		workspace:    workspace,
		detector:     analyzer.detector,
		dependencies: dependencies,
		store:        analyzer.store,
	}
	scenarios := &scenarioService{workspace: workspace, store: analyzer.store}
	deployment := &deploymentService{workspace: workspace}
	proposal := &proposalService{}
	graph := &graphService{analyzer: analyzer}

	return services.Registry{
		Analysis:     analysis,
		Scan:         scan,
		Graph:        graph,
		Optimization: optimization,
		Scenarios:    scenarios,
		Dependencies: dependencies,
		Deployment:   deployment,
		Proposal:     proposal,
	}
}
