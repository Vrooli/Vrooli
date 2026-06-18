package app

import (
	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/convergencedrift"
	"architecture-cartographer/internal/conflicts/detectors/couplingsmell"
	"architecture-cartographer/internal/conflicts/detectors/crossscenario"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/domainsparsewarning"
	"architecture-cartographer/internal/conflicts/detectors/filecohesion"
	"architecture-cartographer/internal/conflicts/detectors/glossarydrift"
	"architecture-cartographer/internal/conflicts/detectors/layering"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	"architecture-cartographer/internal/conflicts/detectors/naming"
	"architecture-cartographer/internal/conflicts/detectors/surfacecoherence"
	mislocatedresolver "architecture-cartographer/internal/conflicts/resolvers/mislocatedfile"
	"architecture-cartographer/internal/signals/boundaries"
)

func ConflictsService(
	primary conflicts.SQLExecutor,
	clk clock.Clock,
	cfg config.Config,
	boundaryCfg boundaries.Config,
	analyticsSvc analytics.Service,
) conflicts.Service {
	return conflicts.NewServiceWithAnalytics(
		conflicts.NewSQLiteRepository(primary, clk),
		conflicts.NewRegistryWithProfiles(
			conflicts.DefaultSurfaceProfiles(),
			convergencedrift.New(),
			couplingsmell.NewWithConfig(boundaryCfg),
			crossscenario.New(),
			cycle.New(),
			domainsparsewarning.New(),
			filecohesion.NewWithConfig(filecohesion.Config{
				MaxLines:   cfg.FileCohesionMaxLines,
				MaxSymbols: cfg.FileCohesionMaxSymbols,
			}),
			glossarydrift.New(),
			layering.NewWithStrict(cfg.LayeringStrict),
			mislocatedfile.New(),
			naming.NewWithBannedVocabulary(cfg.BannedVocabulary),
			surfacecoherence.New(),
		),
		conflicts.NewResolverRegistry(mislocatedresolver.New()),
		conflicts.NewAnalyticsAdapter(analyticsSvc),
	)
}
