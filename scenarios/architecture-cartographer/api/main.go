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
	"architecture-cartographer/internal/graph/scenariopath"
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
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	analyticsH "architecture-cartographer/handlers/analytics"
	applyH "architecture-cartographer/handlers/apply"
	conflictsH "architecture-cartographer/handlers/conflicts"
	graphH "architecture-cartographer/handlers/graph"
	healthH "architecture-cartographer/handlers/health"
	manifestH "architecture-cartographer/handlers/manifest"
	signalsH "architecture-cartographer/handlers/signals"
)

// tsProjectCandidates returns the ordered probe list for locating a
// scenario's TypeScript project directory (a dir containing
// tsconfig.json). Defaults to ui/ then the scenario root; override the
// subdir list with CARTOGRAPHER_TS_PROJECT_DIRS (comma-separated,
// scenario-relative; "." means the scenario root).
func tsProjectCandidates() []scenariopath.Candidate {
	return projectCandidates("CARTOGRAPHER_TS_PROJECT_DIRS", []string{"ui", "."}, "tsconfig.json")
}

// goProjectCandidates returns the ordered probe list for locating a
// scenario's Go project directory (a dir containing go.mod). Defaults to
// api/ then cli/ then the scenario root; override with
// CARTOGRAPHER_GO_PROJECT_DIRS.
func goProjectCandidates() []scenariopath.Candidate {
	return projectCandidates("CARTOGRAPHER_GO_PROJECT_DIRS", []string{"api", "cli", "."}, "go.mod")
}

// projectCandidates builds the candidate probe list for a language,
// honoring an optional comma-separated env override of the subdir order.
func projectCandidates(envKey string, defaultSubdirs []string, marker string) []scenariopath.Candidate {
	subdirs := defaultSubdirs
	if raw := strings.TrimSpace(os.Getenv(envKey)); raw != "" {
		parsed := make([]string, 0, len(defaultSubdirs))
		for _, p := range strings.Split(raw, ",") {
			if p = strings.TrimSpace(p); p != "" {
				parsed = append(parsed, p)
			}
		}
		if len(parsed) > 0 {
			subdirs = parsed
		}
	}
	out := make([]scenariopath.Candidate, 0, len(subdirs))
	for _, d := range subdirs {
		out = append(out, scenariopath.Candidate{Subdir: d, Marker: marker})
	}
	return out
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

	// Code-graph adapters reach their sibling producer scenarios over
	// Connect, resolving the (dynamic) sibling API port per call via
	// api-core discovery (`vrooli scenario port <slug>`). Both adapters
	// are always registered; for a given target scenario each adapter
	// probes for its own language's project directory (ui/ → TypeScript,
	// api//cli/ → Go) and contributes nothing when absent, and the graph
	// service skips any producer that is not running. A missing language
	// or stopped producer therefore degrades gracefully instead of
	// erroring the whole extract.
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	repoRoot, repoErr := repocontract.FindRepoRootFromEnvOrCWD()
	if repoErr != nil {
		log.Printf("cartographer: repo root resolution failed; code-graph adapters cannot locate project dirs: %v", repoErr)
	}
	tsProjects := scenariopath.NewResolver(repoRoot, tsProjectCandidates())
	goProjects := scenariopath.NewResolver(repoRoot, goProjectCandidates())
	adapters := []graph.CodeGraphAdapter{
		gocodegraph.New(gocodegraph.Config{URLResolver: resolver, ProjectPath: goProjects.Resolve}),
		tscodegraph.New(tscodegraph.Config{URLResolver: resolver, ProjectPath: tsProjects.Resolve}),
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
