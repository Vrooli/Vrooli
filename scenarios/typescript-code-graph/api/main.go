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

	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/modules"
	intrewrite "typescript-code-graph/internal/rewrite"
	"typescript-code-graph/internal/server"
	"typescript-code-graph/internal/sidecar"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	graphH "typescript-code-graph/handlers/graph"
	healthH "typescript-code-graph/handlers/health"
	validationH "typescript-code-graph/handlers/validation"
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

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "typescript-code-graph"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "typescript-code-graph",
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
	// the same project_path serialize (OT-P0-007). The rewrite domain
	// holds plans in-memory until REQ-P1-002 lands SQLite persistence.
	pathMu := intgraph.NewPathMutex()
	graphSvc := intgraph.NewService(supervisor, pathMu)
	rewriteStore := intrewrite.NewMemoryPlanStore()
	rewriteSvc := intrewrite.NewService(rewriteStore, supervisor, pathMu)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "typescript-code-graph-api", "1.0.0",
			healthH.FuncProvider(func() string { return string(supervisor.Status()) })),
		graphH.Module(graphSvc, rewriteSvc, log.Default()),
		validationH.Module(),
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
