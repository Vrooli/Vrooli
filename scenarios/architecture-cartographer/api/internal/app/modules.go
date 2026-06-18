package app

import (
	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/campaign"
	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/module"
	"architecture-cartographer/internal/suppressions"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"

	analyticsH "architecture-cartographer/handlers/analytics"
	applyH "architecture-cartographer/handlers/apply"
	auditH "architecture-cartographer/handlers/audit"
	campaignH "architecture-cartographer/handlers/campaign"
	conflictsH "architecture-cartographer/handlers/conflicts"
	domainsH "architecture-cartographer/handlers/domains"
	graphH "architecture-cartographer/handlers/graph"
	healthH "architecture-cartographer/handlers/health"
	signalsH "architecture-cartographer/handlers/signals"
)

func Modules(db *database.RoutedDB, repoRoot string, cfg config.Config) []module.Module {
	clk := clock.System{}
	primary := db.Primary()
	resolver := discovery.NewResolver(discovery.ResolverConfig{})

	graphSvc := GraphService(primary, clk, repoRoot, resolver)
	analyticsSvc := analytics.NewService(analytics.NewSQLiteRepository(primary, clk))
	scenarioLocator := domains.NewRepoScenarioLocator(repoRoot)
	domainsSvc := DomainsService(repoRoot, clk, cfg, resolver)
	signalsSvc := SignalsService(graphSvc, domainsSvc, cfg)
	conflictsSvc := ConflictsService(primary, clk, cfg, BoundaryConfig(cfg), analyticsSvc)

	suppressionProvider := suppressions.NewProvider(scenarioLocator, suppressions.NewFileScanner(), clk)
	auditSvc := audit.NewService(
		graphSvc,
		domainsSvc,
		conflictsSvc,
		conflictsH.NewSignalsVerdictAdapter(signalsSvc),
		suppressionProvider,
		audit.NewDirScenarioLister(repoRoot),
		clk,
	)
	applySvc := apply.NewService(
		apply.NewSQLiteRepository(primary, clk),
		conflictsSvc,
		apply.NewRecipeRegistry(),
		apply.WithSuppressionWriter(suppressions.NewFileWriter(), scenarioLocator),
	)
	campaignSvc := campaign.NewService(campaign.NewSQLiteRepository(primary, clk))

	return []module.Module{
		healthH.Module(db, "architecture-cartographer-api", "1.0.0"),
		analyticsH.Module(analyticsSvc),
		applyH.Module(applySvc),
		auditH.Module(auditSvc),
		conflictsH.Module(conflictsH.Deps{Conflicts: conflictsSvc, Graph: graphSvc, Domains: domainsSvc, Signals: signalsSvc, Suppressions: suppressionProvider}),
		domainsH.Module(domainsSvc),
		graphH.Module(graphSvc),
		campaignH.Module(campaignSvc),
		signalsH.Module(signalsSvc),
	}
}
