package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"business-health/internal/modules"
	"business-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	fleetH "business-health/handlers/fleet"
	healthH "business-health/handlers/health"
	searchH "business-health/handlers/search"
	searchcontrolH "business-health/handlers/searchcontrol"
	validationH "business-health/handlers/validation"
	wizardH "business-health/handlers/wizard"
	"business-health/internal/aisearch"

	aisearchpkg "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

// intentProviderID is the search-hub leaf this scenario registers.
const intentProviderID = "business-health.intent"

// loadSearchTuning reads the tuning SSOT from .vrooli/search.json. A
// missing file or absent provider is a fatal boot error (greenfield — no
// env fallback).
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

// repoRoot resolves the Vrooli repo root: REPO_ROOT env when set, else
// walking up from the scenario directory this binary serves.
func repoRoot() string {
	if root := strings.TrimSpace(os.Getenv("REPO_ROOT")); root != "" {
		return root
	}
	if dir, err := os.Getwd(); err == nil {
		for d := dir; d != "/"; d = filepath.Dir(d) {
			if _, err := os.Stat(filepath.Join(d, "scenarios")); err == nil {
				if _, err := os.Stat(filepath.Join(d, "packages")); err == nil {
					return d
				}
			}
		}
	}
	return "."
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "business-health"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "business-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	root := repoRoot()
	scenarioDir := filepath.Join(root, "scenarios", "business-health")
	engine := validationH.NewEngine(root)
	logger := log.Default()

	// AI search wiring — the scenario-owned .vrooli/search.json is the SSOT
	// for tuning (engine shape, embed recipe, rerank policy); LoadConfig
	// supplies only operational wiring (qdrant address, sync cadence,
	// reranker endpoints). See cli-health for the worked example.
	searchJSONPath := filepath.Join(scenarioDir, ".vrooli", "search.json")
	searchCfg := aisearchpkg.LoadConfig("BUSINESS_HEALTH")
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
	engineDeps, err = aisearchpkg.ResolveEngineDepsEmbedding(policyCtx, engineDeps)
	cancelPolicy()
	if err != nil {
		log.Fatalf("resolve business-health embedding policy: %v", err)
	}
	tuning := loadSearchTuning(searchJSONPath, intentProviderID)
	intentSource := aisearch.NewFleetIntentSource(root)
	aiService := aisearch.NewTunedService(tuning, aisearch.TunedOptions{
		Source:           intentSource,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		EngineDeps:       engineDeps,
	})
	// Best-effort: qdrant down at boot degrades to text-fallback search.
	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[business-health] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}
	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearchpkg.NewSyncLoopFunc("business-health", aiService.Reconciler, searchCfg)
	go syncLoop.Start(syncCtx)
	// Populate a newly selected collection layout immediately at boot; the
	// periodic loop remains the repair path for later drift.
	go func() {
		if _, _, err := syncLoop.RunOnce(syncCtx); err != nil {
			logger.Printf("[business-health] initial search reconcile failed (continuing degraded): %v", err)
		}
	}()

	// The minted control token gates the shared reindex/config-write plane;
	// memory-only, re-acquired on every boot's re-registration.
	controlToken := searchH.NewTokenHolder()
	controlGate := &searchcontrolH.Gate{Token: controlToken.Get}
	go searchregister.Register(syncCtx, searchregister.Config{
		ScenarioID:     "business-health",
		SearchFilePath: searchJSONPath,
		Logger:         logger,
		OnControlToken: func(_ string, token string) { controlToken.Set(token) },
		ControlToken:   func(string) string { return controlToken.Get() },
	})

	// The wizard's capability-dedup hook queries the intent corpus
	// in-process and degrades silently when the backends are down.
	hinter := aisearch.NewWizardHinter(aiService)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "business-health-api", "1.0.0"),
		validationH.Module(logger, root, scenarioDir, engine),
		wizardH.Module(logger, root, filepath.Join(scenarioDir, "data"), engine, hinter),
		fleetH.Module(logger, root, engine),
		searchH.Module(logger, aiService),
		searchcontrolH.Module(logger, searchcontrolH.Deps{
			Logger:       logger,
			Reindexer:    searchcontrolH.ServiceAdapter{Service: aiService},
			ConfigWriter: searchcontrolH.FileConfigWriter{Path: searchJSONPath},
			CorpusWriter: searchcontrolH.FileCorpusWriter{Path: searchJSONPath},
			Gate:         controlGate,
		}),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
