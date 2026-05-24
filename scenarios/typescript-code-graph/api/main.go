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

	"typescript-code-graph/internal/clock"
	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/modules"
	intrewrite "typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/server"
	"typescript-code-graph/internal/sidecar"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	graphH "typescript-code-graph/handlers/graph"
	healthH "typescript-code-graph/handlers/health"
)

// resolveSidecarDistPath returns the absolute path to the bundled Node
// sidecar entrypoint. Resolution order:
//
//  1. TS_CODE_GRAPH_SIDECAR_DIST env override (operator escape hatch).
//  2. <scenario_root>/sidecar/dist/index.js, where scenario_root is
//     two levels up from the api binary's working directory (the
//     conventional layout under scenarios/typescript-code-graph/).
func resolveSidecarDistPath() (string, error) {
	if p := strings.TrimSpace(os.Getenv("TS_CODE_GRAPH_SIDECAR_DIST")); p != "" {
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", fmt.Errorf("resolve TS_CODE_GRAPH_SIDECAR_DIST: %w", err)
		}
		return abs, nil
	}
	// CWD is typically the api/ directory when run via `go run .`, or
	// the scenario root when run via the lifecycle. Try both.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	candidates := []string{
		filepath.Join(cwd, "sidecar", "dist", "index.js"),
		filepath.Join(cwd, "..", "sidecar", "dist", "index.js"),
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
	}
	// Fall back to the first candidate so the supervisor's own
	// "dist path not found" error fires with a useful message.
	return filepath.Abs(candidates[0])
}

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
		storage.Options{ScenarioID: "typescript-code-graph"},
		storage.ClassData,
		"typescript-code-graph.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve typescript-code-graph db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "typescript-code-graph"}) {
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

	// Sidecar supervisor: must come up before the HTTP listener so
	// callers never see a Connect handler responding "ready" while the
	// sidecar is still mid-spawn. If the supervisor cannot start at
	// all, the API is useless — fail loud.
	distPath, err := resolveSidecarDistPath()
	if err != nil {
		log.Fatalf("sidecar dist path resolution failed: %v", err)
	}
	supCtx, supCancel := context.WithCancel(context.Background())
	defer supCancel()
	supervisor := sidecar.NewSupervisor(sidecar.Config{
		DistPath:          distPath,
		NodeBin:           "node",
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  5 * time.Second,
		StderrSink:        os.Stderr,
	})
	if err := supervisor.Start(supCtx); err != nil {
		log.Fatalf("sidecar supervisor failed to start: %v", err)
	}

	// graph + rewrite share one PathMutex so Extract and Apply against
	// the same scenario_path serialize (OT-P0-007). The rewrite domain
	// holds plans in-memory until REQ-P1-002 lands SQLite persistence.
	pathMu := intgraph.NewPathMutex()
	graphSvc := intgraph.NewService(supervisor, pathMu)
	rewriteStore := intrewrite.NewMemoryPlanStore()
	rewriteSvc := intrewrite.NewService(rewriteStore, supervisor, pathMu)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "typescript-code-graph-api", "1.0.0",
			healthH.FuncProvider(func() string { return string(supervisor.Status()) })),
		graphH.Module(graphSvc, rewriteSvc, log.Default()),
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
			// 5s graceful sidecar shutdown then DB close. We deliberately
			// honor the caller's ctx so a forceful shutdown still kills
			// the child rather than hanging the API.
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			if err := supervisor.Shutdown(shutdownCtx); err != nil {
				log.Printf("sidecar shutdown: %v", err)
			}
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

