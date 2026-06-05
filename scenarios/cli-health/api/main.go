package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cli-health/internal/aisearch"
	"cli-health/internal/clock"
	"cli-health/internal/modules"
	"cli-health/internal/server"

	aisearchpkg "github.com/vrooli/aisearch-go"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	healthH "cli-health/handlers/health"
	reindexH "cli-health/handlers/reindex"
	searchH "cli-health/handlers/search"
	validationH "cli-health/handlers/validation"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
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
	path, err := resolver.Path(
		storage.Options{ScenarioID: "cli-health"},
		storage.ClassData,
		"cli-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve cli-health db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "cli-health"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
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

	// AI search wiring: the shared engine (packages/aisearch-go) provides the
	// embedder + vector store + reconciler + env config + sync loop; cli-health
	// supplies the command discovery source and the search/reindex service.
	// NewDenseEngine assembles the dense-only common case (embedder + store +
	// reranker chain) so the collection name is named exactly once; the reranker
	// chain stays default-off (CLI_HEALTH_RERANK_ENABLED) until an attended A/B.
	searchCfg := aisearchpkg.LoadConfig("CLI_HEALTH")
	engine := aisearchpkg.NewDenseEngine(searchCfg, aisearch.DefaultCollection)
	discovery := aisearch.NewFilesystemDiscoverySource(repoRoot)
	// Index the top-level vrooli CLI alongside scenario CLIs. The
	// records carry Origin="vrooli"; the validation handler rejects
	// "vrooli" because no proto contract exists for it.
	discovery.ExternalCLIs = []aisearch.ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
	aiService := aisearch.NewService(aisearch.Options{
		Embedder:         engine.Embedder,
		VectorStore:      engine.VectorStore,
		Discovery:        discovery,
		Parallelism:      searchCfg.ReconcileParallelism,
		MaxEmbedsPerTick: searchCfg.MaxEmbedsPerTick,
		Floor: aisearchpkg.FloorConfig{
			MaxGap:    searchCfg.RelevanceMaxGap,
			HardFloor: searchCfg.RelevanceHardFloor,
		},
		RerankEnabled:   searchCfg.RerankEnabled,
		Reranker:        engine.Reranker,
		RerankShortlist: searchCfg.RerankShortlist,
	})

	// EnsureCollection is best-effort: if qdrant is unreachable at boot, the
	// scenario still serves text-fallback search and a degraded status.
	if err := aiService.EnsureCollection(context.Background()); err != nil {
		logger.Printf("[cli-health] qdrant collection ensure failed (continuing with degraded search): %v", err)
	}

	// Sync loop drives periodic reconcile against qdrant. Cancelled by the
	// api-core server's shutdown context.
	syncCtx, cancelSync := context.WithCancel(context.Background())
	syncLoop := aisearchpkg.NewSyncLoop("cli-health", aiService.Reconciler(), searchCfg)
	go syncLoop.Start(syncCtx)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "cli-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot, externalCLINames(discovery.ListExternalCLIs())),
		searchH.Module(logger, aiService),
		reindexH.Module(logger, reindexH.ServiceAdapter{Service: aiService}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error {
			cancelSync()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func externalCLINames(clis []aisearch.ExternalCLI) []string {
	out := make([]string, 0, len(clis))
	for _, c := range clis {
		out = append(out, c.Name)
	}
	return out
}
