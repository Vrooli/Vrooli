package main

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"ui-health/internal/aisearch"
	"ui-health/internal/modules"
	"ui-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	aisearchpkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	healthH "ui-health/handlers/health"
	reindexH "ui-health/handlers/reindex"
	searchH "ui-health/handlers/search"
	validationH "ui-health/handlers/validation"
	visualhealthH "ui-health/handlers/visualhealth"
)

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "ui-health"}) {
		return
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "ui-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	// AI search wiring: the scenario-owned `.vrooli/search.json` is the SSOT for
	// the surface-search tuning factors (engine shape, embed recipe, rerank policy,
	// floor band) — read here at boot. The shared engine (packages/ai-go/search)
	// provides the embedder + vector store + reconciler + sync loop; LoadConfig
	// supplies only the OPERATIONAL wiring (Qdrant address, sync cadence,
	// parallelism). The surfaces corpus is dense single-chunk; NewSearchService
	// picks the engine shape from the tuning DATA.
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "ui-health", ".vrooli", "search.json")
	searchCfg := aisearchpkg.LoadConfig("UI_HEALTH")
	engineDeps := aisearchpkg.EngineDeps{
		QdrantURL:    searchCfg.QdrantURL,
		QdrantAPIKey: searchCfg.QdrantAPIKey,
		Collection:   aisearch.DefaultCollection,
		EmbedRole:    searchCfg.EmbedRole,
	}
	policyCtx, cancelPolicy := context.WithTimeout(context.Background(), 5*time.Second)
	engineDeps, err = aisearchpkg.ResolveEngineDepsEmbedding(policyCtx, engineDeps)
	cancelPolicy()
	if err != nil {
		log.Fatalf("resolve ui-health embedding policy: %v", err)
	}
	tuning := loadSearchTuning(searchJSONPath, surfacesProviderID)
	discovery := aisearch.NewFilesystemDiscoverySource(repoRoot)
	aiService := aisearch.NewSearchService(tuning, aisearch.Options{
		Discovery:        discovery,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		Threshold:        aisearch.DefaultSearchThreshold,
		EngineDeps:       engineDeps,
	})

	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[ui-health] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearchpkg.NewSyncLoopFunc("ui-health", aiService.Reconciler, searchCfg)
	go syncLoop.Start(syncCtx)

	// Self-register the surfaces provider with search-hub from the same SSOT: the
	// descriptor goes to the registry and the tests block is mirrored into the
	// eval store. search-hub is an OPTIONAL dependency, so this runs in the
	// background with bounded retry and degrades gracefully.
	go searchregister.Register(syncCtx, searchregister.Config{
		ScenarioID:     "ui-health",
		SearchFilePath: searchJSONPath,
		Logger:         logger,
	})

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "ui-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot),
		visualhealthH.Module(logger),
		searchH.Module(logger, aiService),
		reindexH.Module(logger, reindexH.ServiceAdapter{Service: aiService}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		// Execution-enabled validation collects BAS runtime evidence and can take
		// longer than the platform's 30-second default. The HTTP write deadline
		// spans handler execution, so keep the response channel alive for the
		// bounded provider phase instead of returning a misleading EOF to clients.
		WriteTimeout: 15 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// surfacesProviderID is the provider id ui-health owns in its search.json SSOT.
const surfacesProviderID = "ui-health.surfaces"

// loadSearchTuning reads the resolved tuning for the surfaces provider from the
// scenario-owned search.json SSOT.
func loadSearchTuning(path, providerID string) aisearchpkg.TuningConfig {
	file, err := aisearchpkg.LoadSearchFile(path)
	if err != nil {
		log.Fatalf("load search tuning: %v", err)
	}
	provider, ok := file.Provider(providerID)
	if !ok {
		log.Fatalf("load search tuning: provider %q not found in %s", providerID, path)
	}
	return provider.ResolvedTuning()
}
