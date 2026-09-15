package app

import (
	"fmt"

	graphdomain "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/graph"
	types "github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/types"
)

type graphService struct {
	analyzer *Analyzer
}

func (g *graphService) GenerateGraph(graphType string) (*types.DependencyGraph, error) {
	if g == nil || g.analyzer == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	return g.analyzer.GenerateGraph(graphType)
}

func (g *graphService) GraphCentrality(coreSeeds []string, scenario string) (*types.GraphCentralityReport, error) {
	if g == nil || g.analyzer == nil {
		return nil, fmt.Errorf("analyzer not initialized")
	}
	graph, err := g.analyzer.GenerateGraph("combined")
	if err != nil {
		return nil, err
	}
	return graphdomain.CalculateCentrality(graph, coreSeeds, scenario), nil
}
