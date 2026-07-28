package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"signal-inbox/internal/categories"
	"signal-inbox/internal/clock"
	"signal-inbox/internal/inference"
	"signal-inbox/internal/modules"
	"signal-inbox/internal/retrieval"
	"signal-inbox/internal/searchregistry"
	"signal-inbox/internal/server"
	"signal-inbox/internal/sources"
	"signal-inbox/internal/triage"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	_ "modernc.org/sqlite"

	categoriesH "signal-inbox/handlers/categories"
	healthH "signal-inbox/handlers/health"
	retrievalH "signal-inbox/handlers/retrieval"
	signalsH "signal-inbox/handlers/signals"
	sourcesH "signal-inbox/handlers/sources"
	triageH "signal-inbox/handlers/triage"
)

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
	scenarioID, err := storage.ScenarioNamespace("signal-inbox")
	if err != nil {
		return "", fmt.Errorf("resolve signal-inbox storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"signal-inbox.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve signal-inbox db path: %w", err)
	}
	return sqliteFileDSN(path)
}

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("signal-inbox")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve signal-inbox storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
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
	if preflight.Run(preflight.Config{ScenarioName: "signal-inbox"}) {
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
	var inferenceClient inference.Client
	if gatewayURL, resolveErr := discovery.ResolveScenarioURLDefault(context.Background(), "ai-gateway"); resolveErr != nil {
		slog.Warn("ai-gateway unavailable; capture will record uncategorized signals", "error", resolveErr)
	} else {
		inferenceClient = inference.NewGatewayClient(routingconnect.NewRoutingServiceClient(http.DefaultClient, gatewayURL))
	}
	categoryService := categories.NewService(categories.NewSQLiteRepository(db), clock.System{}, inferenceClient)
	triageService := triage.NewService(triage.NewSQLiteRepository(db), clock.System{})
	retrievalService := retrieval.NewService(retrieval.NewSQLiteRepository(db), clock.System{}, retrieval.NewQdrantSemanticSearch(inferenceClient))
	if _, err := categoryService.Bootstrap(context.Background()); err != nil {
		log.Fatalf("seed reserved category: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)
	signalsRuntime, err := signalsH.NewRuntimeWithRoutedRoots(db, clock.System{}, fileRoots, categoryService)
	if err != nil {
		log.Fatalf("signals runtime: %v", err)
	}
	sourcesService, err := sources.NewService(sources.NewSQLiteRepository(db), signalsRuntime.Service, clock.System{}, sources.ChromeBookmarksAdapter{})
	if err != nil {
		log.Fatalf("sources runtime: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "signal-inbox-api", "1.0.0"),
		categoriesH.Module(categoryService),
		signalsH.ModuleWithRuntime(signalsRuntime, log.Default()),
		sourcesH.Module(sourcesService),
		retrievalH.Module(retrievalService),
		triageH.Module(triageService),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// Search Hub is an optional federation layer. Register after this scenario's
	// own routes exist, in the background, so an unavailable hub never delays or
	// fails Signal Inbox startup. The lifecycle launches this binary from api/.
	go searchregistry.Register(context.Background(), filepath.Join("..", ".vrooli", "search.json"), log.Default())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
