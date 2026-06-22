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

	internalbudgets "performance-health/internal/budgets"
	"performance-health/internal/clock"
	"performance-health/internal/modules"
	"performance-health/internal/server"
	"performance-health/internal/trend"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	analysisH "performance-health/handlers/analysis"
	auditH "performance-health/handlers/audit"
	benchmarkH "performance-health/handlers/benchmark"
	budgetsH "performance-health/handlers/budgets"
	fleetH "performance-health/handlers/fleet"
	healthH "performance-health/handlers/health"
	lighthouseH "performance-health/handlers/lighthouse"
	startupH "performance-health/handlers/startup"
	sweepH "performance-health/handlers/sweep"
	trendH "performance-health/handlers/trend"
	validationH "performance-health/handlers/validation"
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
	scenarioID, err := storage.ScenarioNamespace("performance-health")
	if err != nil {
		return "", fmt.Errorf("resolve performance-health storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"performance-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve performance-health db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "performance-health"}) {
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

	// Additive trend-store migration runs BEFORE EnsureSchemas: it brings an
	// older perf_samples table (the P4 scaffold shipped a narrower one) up to the
	// current column set so the declared CREATE TABLE block matches the on-disk
	// shape and the drift detector passes. It is a no-op on a fresh DB and never
	// drops or rewrites a persisted sample.
	if err := trend.EnsureColumns(context.Background(), db.Primary()); err != nil {
		log.Fatalf("trend store migration failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	// The trend store is the single concrete sample sink/source, constructed
	// once here at the composition root and injected into the producer domains
	// (analysis, benchmark, startup) and the budgets measurement source — so
	// those domains depend on narrow seams, never on the trend domain itself.
	trendStore := trend.NewStore(db.Primary())

	// The capture-sweep gate is a budgets service over the same config store +
	// flow-tagged sample source: it enumerates declared per-flow budgets and
	// checks each flow's latest flow-tagged sample. Shared shape with the budgets
	// handler, constructed here so the sweep can drive CheckFlow off the trend.
	sweepGate := internalbudgets.NewService(
		internalbudgets.NewConfigStore(repoRoot, nil),
		internalbudgets.WithMeasurementSource(internalbudgets.NewSampleMeasurementSource(trendStore)),
	)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "performance-health-api", "1.0.0"),
		analysisH.Module(logger, repoRoot, trendStore),
		auditH.Module(logger, repoRoot),
		benchmarkH.Module(logger, repoRoot, trendStore),
		budgetsH.Module(logger, repoRoot, internalbudgets.NewSampleMeasurementSource(trendStore)),
		fleetH.Module(logger, repoRoot, db.Primary()),
		lighthouseH.Module(logger, repoRoot),
		startupH.Module(logger, db.Primary(), trendStore),
		sweepH.Module(logger, repoRoot, trendStore, sweepGate),
		trendH.Module(logger, db.Primary()),
		validationH.Module(logger, repoRoot, db.Primary()),
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
		// audit/sweep RPCs synchronously restart the target scenario in profile
		// build mode (a full UI rebuild — minutes per CLAUDE.md) and drive a BAS
		// browser capture before responding. The api-core default 30s WriteTimeout
		// severs the connection mid-handler, so the CLI sees `unexpected EOF`. Mirror
		// git-control-tower (same BAS-capture pattern) and give long synchronous
		// captures room. Health/CRUD routes are unaffected (they respond in ms).
		WriteTimeout: 15 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
