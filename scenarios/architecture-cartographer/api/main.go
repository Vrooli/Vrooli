package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/clock"
	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	mislocatedresolver "architecture-cartographer/internal/conflicts/resolvers/mislocatedfile"
	"architecture-cartographer/internal/git"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/graph/gocodegraph"
	"architecture-cartographer/internal/graph/tscodegraph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/modules"
	"architecture-cartographer/internal/server"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/gitcoedit"
	"architecture-cartographer/internal/signals/importcluster"
	"architecture-cartographer/internal/signals/importervoting"
	"architecture-cartographer/internal/signals/pathtoken"
	"architecture-cartographer/internal/signals/symbolglossary"
	"architecture-cartographer/internal/signals/testcoupling"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	analyticsH "architecture-cartographer/handlers/analytics"
	applyH "architecture-cartographer/handlers/apply"
	conflictsH "architecture-cartographer/handlers/conflicts"
	graphH "architecture-cartographer/handlers/graph"
	healthH "architecture-cartographer/handlers/health"
	manifestH "architecture-cartographer/handlers/manifest"
	signalsH "architecture-cartographer/handlers/signals"
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
		storage.Options{ScenarioID: "architecture-cartographer"},
		storage.ClassData,
		"architecture-cartographer.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve architecture-cartographer db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "architecture-cartographer"}) {
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

	// Wire per-domain services. Production wires the sqlite repository
	// in every domain that persists; signals is stateless. Adapter
	// stubs return IntegrationError until go-code-graph and
	// typescript-code-graph ship.
	clk := clock.System{}
	primary := db.Primary()

	// Register only the code-graph adapters whose discovery URL is
	// configured. Operators wire each scenario via env (GO_CODE_GRAPH_URL,
	// TYPESCRIPT_CODE_GRAPH_URL); unconfigured adapters are intentionally
	// omitted so a missing target language returns a clean "no adapter
	// registered" error rather than per-call scenario_unreachable noise.
	adapters := make([]graph.CodeGraphAdapter, 0, 2)
	if u := strings.TrimSpace(os.Getenv("GO_CODE_GRAPH_URL")); u != "" {
		adapters = append(adapters, gocodegraph.New(u))
	}
	if u := strings.TrimSpace(os.Getenv("TYPESCRIPT_CODE_GRAPH_URL")); u != "" {
		adapters = append(adapters, tscodegraph.New(u))
	}
	graphSvc := graph.NewService(
		graph.NewSQLiteRepository(primary, clk), clk,
		adapters...,
	)
	manifestSvc := manifest.NewService(manifest.NewSQLiteRepository(primary, clk))
	analyticsSvc := analytics.NewService(analytics.NewSQLiteRepository(primary, clk))

	signalsReg := signals.NewRegistry(
		pathtoken.New(),
		importcluster.New(),
		symbolglossary.New(),
		importervoting.New(),
		testcoupling.New(),
		gitcoedit.New(git.NewRealRunner()),
	)
	signalsSvc := signals.NewService(
		signalsReg, signals.NewAggregator(signalsReg, nil),
		signals.NewGraphSnapshotProvider(graphSvc),
		manifestSvc,
	)

	conflictsRepo := conflicts.NewSQLiteRepository(primary, clk)
	conflictsSvc := conflicts.NewServiceWithAnalytics(
		conflictsRepo,
		conflicts.NewRegistry(cycle.New(), mislocatedfile.New()),
		conflicts.NewResolverRegistry(mislocatedresolver.New()),
		conflicts.NewAnalyticsAdapter(analyticsSvc),
	)

	applySvc := apply.NewService(
		apply.NewSQLiteRepository(primary, clk),
		conflictsSvc,
		apply.NewRecipeRegistry(),
	)

	srv := server.New(
		server.Deps{Clock: clk, Logger: log.Default()},
		healthH.Module(db, "architecture-cartographer-api", "1.0.0"),
		analyticsH.Module(analyticsSvc),
		applyH.Module(applySvc),
		conflictsH.Module(conflictsH.Deps{Conflicts: conflictsSvc, Graph: graphSvc, Manifest: manifestSvc}),
		graphH.Module(graphSvc),
		manifestH.Module(manifestSvc),
		signalsH.Module(signalsSvc),
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
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

