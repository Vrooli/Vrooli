package app

import (
	"log"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/gocodegraph"
	"architecture-cartographer/internal/graph/scenariopath"
	"architecture-cartographer/internal/graph/tscodegraph"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/discovery"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
)

func GraphService(primary graph.SQLExecutor, clk schedule.Clock, repoRoot string, resolver *discovery.Resolver) graph.Service {
	if repoRoot == "" {
		log.Printf("cartographer: repo root unavailable; code-graph adapters cannot locate project dirs")
	}
	tsProjects := scenariopath.NewResolver(repoRoot, TSProjectCandidates())
	goProjects := scenariopath.NewResolver(repoRoot, GoProjectCandidates())
	return graph.NewServiceWithFingerprinter(
		graph.NewSQLiteRepository(primary, clk),
		clk,
		graph.NewFileSystemFingerprinterWithOptions(repoRoot, "go-profile=structural"),
		gocodegraph.New(gocodegraph.Config{
			URLResolver: resolver,
			ProjectPath: goProjects.Resolve,
			Profile:     graphv1.ExtractionProfile_EXTRACTION_PROFILE_STRUCTURAL,
		}),
		tscodegraph.New(tscodegraph.Config{URLResolver: resolver, ProjectPath: tsProjects.Resolve}),
	)
}
