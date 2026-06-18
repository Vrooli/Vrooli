package app

import (
	"log"

	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/gocodegraph"
	"architecture-cartographer/internal/graph/scenariopath"
	"architecture-cartographer/internal/graph/tscodegraph"

	"github.com/vrooli/api-core/discovery"
)

func GraphService(primary graph.SQLExecutor, clk clock.Clock, repoRoot string, resolver *discovery.Resolver) graph.Service {
	if repoRoot == "" {
		log.Printf("cartographer: repo root unavailable; code-graph adapters cannot locate project dirs")
	}
	tsProjects := scenariopath.NewResolver(repoRoot, TSProjectCandidates())
	goProjects := scenariopath.NewResolver(repoRoot, GoProjectCandidates())
	return graph.NewService(
		graph.NewSQLiteRepository(primary, clk),
		clk,
		gocodegraph.New(gocodegraph.Config{URLResolver: resolver, ProjectPath: goProjects.Resolve}),
		tscodegraph.New(tscodegraph.Config{URLResolver: resolver, ProjectPath: tsProjects.Resolve}),
	)
}
