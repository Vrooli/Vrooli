package app

import (
	graphdomain "scenario-dependency-analyzer/internal/graph"
	types "scenario-dependency-analyzer/internal/types"
)

// generateDependencyGraph generates a dependency graph using the active analyzer runtime (or fallback globals).
// For deterministic testing, prefer generateDependencyGraphWithSeams.
func generateDependencyGraph(graphType string) (*types.DependencyGraph, error) {
	if analyzer := analyzerInstance(); analyzer != nil {
		return analyzer.GenerateGraph(graphType)
	}

	depSvc := defaultDependencyService()
	builder := graphdomain.NewBuilder(depSvc.store, depSvc.detector, nil)
	return builder.Generate(graphType)
}
