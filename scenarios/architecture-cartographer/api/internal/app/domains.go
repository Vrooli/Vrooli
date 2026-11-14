package app

import (
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/domains"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/discovery"
)

func DomainsService(repoRoot string, clk schedule.Clock, cfg config.Config, resolver *discovery.Resolver) domains.Service {
	surfaceProvider := domains.NewCodeFactsSurfaceProvider(resolver, nil, nil)
	return domains.NewService(
		domains.NewRepoScenarioLocator(repoRoot),
		clk,
		domains.ExtractorsForWithSurfaceProvider(cfg.LadderOrder, cfg.ExtraNonDomainFolders, surfaceProvider)...,
	)
}
