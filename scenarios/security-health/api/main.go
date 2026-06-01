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

	"security-health/internal/clock"
	"security-health/internal/dependencies"
	"security-health/internal/dependencies/aisearch"
	"security-health/internal/modules"
	"security-health/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	dependenciesH "security-health/handlers/dependencies"
	healthH "security-health/handlers/health"
	reindexH "security-health/handlers/reindex"
	validationH "security-health/handlers/validation"
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
		storage.Options{ScenarioID: "security-health"},
		storage.ClassData,
		"security-health.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve security-health db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "security-health"}) {
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

	logger := log.Default()
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("resolve repo root: %v", err)
	}

	// The fleet Dependency & Vulnerability Intelligence service is shared by the
	// dependencies (search/status) module, the reindex (async job) module, and
	// the background reconcile loop, so all three see one corpus + job registry.
	depDeps := dependencies.Deps{
		RepoRoot: repoRoot,
		Store:    dependencies.NewStore(db),
		Clock:    clock.System{},
	}
	// The semantic index is the optional AI-ranking overlay (Ollama embeddings +
	// Qdrant). NewFromConfig returns nil when disabled; only attach a non-nil
	// index so the service's nil-check (TEXT-only) stays correct (avoid the
	// typed-nil interface trap). When attached, search ranks MODE_AI by vector
	// similarity and degrades to TEXT if the backends are down.
	if idx := aisearch.NewFromConfig(aisearch.LoadConfigFromEnv()); idx != nil {
		depDeps.Index = idx
	}
	depService := dependencies.NewService(depDeps)
	// Create the Qdrant collection up front (idempotent, best-effort) so the
	// first reconcile can populate it without a cold-start miss.
	if err := depService.EnsureIndex(context.Background()); err != nil {
		logger.Printf("[security-health] semantic index unavailable at startup (search on TEXT): %v", err)
	}

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, "security-health-api", "1.0.0"),
		validationH.Module(logger, repoRoot),
		dependenciesH.Module(logger, depService),
		reindexH.Module(logger, depService),
	)

	// Background reconcile loop: refresh the fleet SBOM corpus every 5 minutes
	// so the Dependency feature never blocks on a live scan. Cancelled on
	// shutdown via reconcileCancel.
	reconcileCtx, reconcileCancel := context.WithCancel(context.Background())
	go runReconcileLoop(reconcileCtx, depService, logger)

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
			reconcileCancel()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// runReconcileLoop drives a periodic fleet reconcile. It runs once shortly
// after boot (so a freshly-started scenario has an index) and then every
// reconcileInterval until the context is cancelled. Reconcile failures are
// logged and retried on the next tick — a transient scanner/network hiccup
// must never crash the server.
func runReconcileLoop(ctx context.Context, svc *dependencies.Service, logger *log.Logger) {
	const reconcileInterval = 5 * time.Minute
	// Small initial delay so boot isn't competing with the first reconcile's
	// fleet walk + osv-scanner calls.
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := svc.RunReconcileOnce(ctx); err != nil && ctx.Err() == nil {
				logger.Printf("[security-health] dependency reconcile failed (will retry): %v", err)
			}
			timer.Reset(reconcileInterval)
		}
	}
}
