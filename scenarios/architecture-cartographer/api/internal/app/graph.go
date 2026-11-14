package app

import (
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/codefacts"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/discovery"
)

func GraphService(primary graph.SQLExecutor, clk schedule.Clock, repoRoot string, resolver *discovery.Resolver) graph.Service {
	return graph.NewService(
		graph.NewSQLiteRepository(primary, clk),
		clk,
		codefacts.New(resolver, repoRoot, graph.LanguageGo, nil),
		codefacts.New(resolver, repoRoot, graph.LanguageTypeScript, nil),
	)
}
