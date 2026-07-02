package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"business-health/internal/clock"
	"business-health/internal/modules"
	"business-health/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
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

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("business-health")
	if err != nil {
		return "", fmt.Errorf("resolve business-health storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"business-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve business-health db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "business-health"}) {
		return
	}

	dsn, err := sqliteDSN()
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
		server.Deps{Clock: clock.System{}, Logger: logger},
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
