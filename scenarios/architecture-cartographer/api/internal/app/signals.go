package app

import (
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/git"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/boundaries"
	"architecture-cartographer/internal/signals/gitcoedit"
	"architecture-cartographer/internal/signals/importcluster"
	"architecture-cartographer/internal/signals/importervoting"
	"architecture-cartographer/internal/signals/pathtoken"
	"architecture-cartographer/internal/signals/symbolglossary"
	"architecture-cartographer/internal/signals/testcoupling"
)

func BoundaryConfig(cfg config.Config) boundaries.Config {
	def := boundaries.DefaultConfig()
	return boundaries.Config{
		GodDomainFanOut:                 cfg.GodDomainFanOut,
		InstabilityWarnBand:             cfg.InstabilityWarnBand,
		StableKernelMaxEfferent:         def.StableKernelMaxEfferent,
		StableKernelMinAfferentFraction: def.StableKernelMinAfferentFraction,
		ExemptArchetypes:                stringSet(cfg.ArchetypeExemptions),
	}
}

func SignalsService(graphSvc graph.Service, domainsSvc domains.Service, cfg config.Config) signals.Service {
	reg := signals.NewRegistry(
		pathtoken.New(),
		importcluster.New(),
		symbolglossary.New(),
		importervoting.New(),
		testcoupling.New(),
		gitcoedit.New(git.NewRealRunner()),
	)
	return signals.NewService(
		reg,
		signals.NewAggregator(reg, nil).WithThresholds(cfg.AutoPlaceMin, cfg.SuggestMin, cfg.TieDelta),
		signals.NewGraphSnapshotProvider(graphSvc),
		domainsSvc,
		signals.WithBoundaryConfig(BoundaryConfig(cfg)),
	)
}

func stringSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, s := range items {
		out[s] = true
	}
	return out
}
