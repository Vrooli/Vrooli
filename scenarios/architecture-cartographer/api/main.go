package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"architecture-cartographer/internal/aisearch"
	"architecture-cartographer/internal/app"
	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/config"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/module"
	"architecture-cartographer/internal/modules"
	"architecture-cartographer/internal/observability"
	"architecture-cartographer/internal/server"

	aisearchpkg "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	searchH "architecture-cartographer/handlers/search"
	searchcontrolH "architecture-cartographer/handlers/searchcontrol"
)

// domainMapProviderID is the provider id architecture-cartographer owns in its
// search.json SSOT — the leaf search-hub flips ACTIVE on self-registration.
const domainMapProviderID = "architecture-cartographer.domain-map"

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "architecture-cartographer"}) {
		return
	}

	dsn, err := app.SQLiteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := graph.MigrateSchema(context.Background(), db.Primary()); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	clk := clock.System{}
	logger := log.Default()
	repoRoot, repoErr := repocontract.FindRepoRootFromEnvOrCWD()
	if repoErr != nil {
		log.Printf("cartographer: repo root resolution failed: %v", repoErr)
	}

	// Cartographer-global control surface (tunable levers; no per-scenario
	// config). Misconfigured levers degrade to defaults with a logged
	// diagnostic rather than failing startup.
	cfg, cfgDiags := config.Load(os.Getenv)
	for _, d := range cfgDiags {
		log.Printf("cartographer config: %s: %s", d.Key, d.Message)
	}

	// Domain-map AI search wiring. The corpus is the DERIVED domain map of every
	// scenario (one document per domain). The shared engine (packages/ai-go/search)
	// provides embedding + Qdrant + reconcile + sync loop; LoadConfig supplies only
	// OPERATIONAL wiring (Qdrant address, sync cadence, parallelism, reranker
	// endpoints). The scenario-owned `.vrooli/search.json` is the SSOT for the
	// search tuning factors and is read here at boot. Search is an ADDED capability:
	// every failure below degrades gracefully (text fallback over in-process
	// derivation) rather than killing the API the cartographer's other domains need.
	searchModules := searchModules(logger, clk, cfg, repoRoot)

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		append(app.Modules(db, repoRoot, cfg), searchModules.modules...)...,
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)
	observability.RegisterPprof(rootMux, cfg.PprofEnabled)
	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		WriteTimeout: 5 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			searchModules.cancel()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// searchBundle carries the assembled search/searchcontrol modules plus the
// cancel func that stops the sync loop + self-registration goroutine at shutdown.
type searchBundle struct {
	modules []module.Module
	cancel  context.CancelFunc
}

// searchModules assembles the domain-map search provider: the corpus Source over
// the in-process domain derivation, the tuned engine (from search.json), the
// reconcile sync loop, the token-gated control plane, and the search-hub
// self-registration. Every failure degrades gracefully.
func searchModules(logger *log.Logger, clk clock.Clock, cfg config.Config, repoRoot string) searchBundle {
	searchJSONPath := filepath.Join(repoRoot, "scenarios", "architecture-cartographer", ".vrooli", "search.json")

	searchCfg := aisearchpkg.LoadConfig("ARCHITECTURE_CARTOGRAPHER")
	engineDeps := aisearchpkg.EngineDeps{
		QdrantURL:     searchCfg.QdrantURL,
		QdrantAPIKey:  searchCfg.QdrantAPIKey,
		Collection:    aisearch.DefaultCollection,
		EmbedRole:     searchCfg.EmbedRole,
		RerankerURL:   searchCfg.RerankerURL,
		RerankerModel: searchCfg.RerankerModel,
		RerankRole:    searchCfg.RerankRole,
	}
	policyCtx, cancelPolicy := context.WithTimeout(context.Background(), 5*time.Second)
	resolved, err := aisearchpkg.ResolveEngineDepsEmbedding(policyCtx, engineDeps)
	cancelPolicy()
	if err != nil {
		// Ollama policy unreachable at boot: AI mode stays unavailable until the
		// next reconcile resolves it; search still serves via the text fallback.
		logger.Printf("[architecture-cartographer] embedding policy resolve failed (degraded to text search): %v", err)
	} else {
		engineDeps = resolved
	}

	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	domainsSvc := app.DomainsService(repoRoot, clk, cfg, resolver)
	lister := audit.NewDirScenarioLister(repoRoot)

	tuning := loadSearchTuning(logger, searchJSONPath, domainMapProviderID)
	aiService := aisearch.NewTunedService(tuning, aisearch.TunedOptions{
		Provider:         domainsSvc,
		Lister:           lister,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		EngineDeps:       engineDeps,
	})

	// EnsureCollection is best-effort: if qdrant is unreachable at boot, the
	// scenario still serves text-fallback search and a degraded status.
	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[architecture-cartographer] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	// Sync loop drives periodic reconcile against qdrant. Resolve the reconciler
	// each tick (not bound once) so a live ApplyTuning swap re-points the loop.
	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearchpkg.NewSyncLoopFunc("architecture-cartographer", aiService.Reconciler, searchCfg)
	go syncLoop.Start(syncCtx)

	// The control token gates the query-time override channel AND the shared
	// reindex/config-write plane. search-hub mints it at registration and is its
	// only holder, so the token alone gates the mutating verbs.
	controlToken := searchH.NewTokenHolder()
	overrideGate := &searchH.OverrideGate{Token: controlToken.Get}
	controlGate := &searchcontrolH.Gate{Token: controlToken.Get}

	// Self-register the domain-map provider AND its evaluation corpus with
	// search-hub from the same `.vrooli/search.json` SSOT. This flips the
	// `architecture-cartographer.domain-map` registry leaf from capability_gap to
	// ACTIVE. search-hub is OPTIONAL, so this runs in the background with bounded
	// retry and degrades gracefully.
	go searchregister.Register(syncCtx, searchregister.Config{
		ScenarioID:     "architecture-cartographer",
		SearchFilePath: searchJSONPath,
		Logger:         logger,
		OnControlToken: func(_ string, token string) { controlToken.Set(token) },
		ControlToken:   func(string) string { return controlToken.Get() },
	})

	return searchBundle{
		cancel: cancelSync,
		modules: []module.Module{
			searchH.Module(logger, aiService, overrideGate),
			searchcontrolH.Module(logger, searchcontrolH.Deps{
				Logger:       logger,
				Reindexer:    searchcontrolH.ServiceAdapter{Service: aiService},
				ConfigWriter: searchcontrolH.FileConfigWriter{Path: searchJSONPath},
				CorpusWriter: searchcontrolH.FileCorpusWriter{Path: searchJSONPath},
				Gate:         controlGate,
			}),
		},
	}
}

// loadSearchTuning reads the search tuning for a provider from the scenario-owned
// `.vrooli/search.json` (the SSOT). A missing/malformed file or absent provider
// degrades to package defaults (logged) rather than killing the API — search is
// an added capability, not the cartographer's reason to exist.
func loadSearchTuning(logger *log.Logger, path, providerID string) aisearchpkg.TuningConfig {
	file, err := aisearchpkg.LoadSearchFile(path)
	if err != nil {
		logger.Printf("[architecture-cartographer] load search tuning (%s): %v — using defaults", path, err)
		return aisearchpkg.TuningConfig{}.WithDefaults()
	}
	provider, ok := file.Provider(providerID)
	if !ok {
		logger.Printf("[architecture-cartographer] provider %q not in %s — using defaults", providerID, path)
		return aisearchpkg.TuningConfig{}.WithDefaults()
	}
	return provider.ResolvedTuning()
}
