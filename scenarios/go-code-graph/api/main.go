package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-code-graph/internal/modules"
	"go-code-graph/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	graphH "go-code-graph/handlers/graph"
	healthH "go-code-graph/handlers/health"
	rewriteH "go-code-graph/handlers/rewrite"

	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "go-code-graph"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "go-code-graph",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Wire the production seams once. graph and rewrite share a single
	// per-path mutex so concurrent Extract / Apply calls for the same
	// module_path serialize against each other (OT-P0-006).
	pathMutex := intgraph.NewPathMutex()
	loader := intgraph.NewPackagesLoader()
	graphCache := configuredExtractionCache()
	graphSvc := intgraph.NewServiceWithCacheAndEnvironment(
		loader,
		pathMutex,
		configuredMaxConcurrentExtracts(),
		graphCache,
		extractionEnvironmentFingerprint(),
	)

	executor := intrewrite.NewFSExecutor()
	planStore := intrewrite.NewSQLiteStore(db.Primary(), schedule.System())
	rewriteSvc := intrewrite.NewServiceWithLog(planStore, executor, pathMutex, planStore)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "go-code-graph-api", "1.0.0"),
		graphH.Module(graphSvc, rewriteSvc, log.Default()),
		rewriteH.Module(rewriteSvc, log.Default()),
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

func configuredExtractionCache() intgraph.ExtractionCache {
	dir := strings.TrimSpace(os.Getenv("GO_CODE_GRAPH_CACHE_DIR"))
	if dir == "" {
		dataDir := strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR"))
		if dataDir == "" {
			log.Printf("SCENARIO_DATA_DIR is unset; extraction cache disabled")
			return nil
		}
		dir = filepath.Join(dataDir, "extraction-cache")
	}
	cache, err := intgraph.NewFileExtractionCacheWithLimit(dir, configuredExtractionCacheMaxBytes())
	if err != nil {
		log.Printf("extraction cache disabled: %v", err)
		return nil
	}
	return cache
}

func configuredExtractionCacheMaxBytes() int64 {
	const (
		envName = "GO_CODE_GRAPH_CACHE_MAX_BYTES"
	)
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return 512 << 20
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		log.Printf("invalid %s=%q; using default %d", envName, raw, 512<<20)
		return 512 << 20
	}
	return value
}

func extractionEnvironmentFingerprint() string {
	values := []string{
		"GOFLAGS=" + os.Getenv("GOFLAGS"),
		"GOOS=" + os.Getenv("GOOS"),
		"GOARCH=" + os.Getenv("GOARCH"),
		"CGO_ENABLED=" + os.Getenv("CGO_ENABLED"),
		"GOEXPERIMENT=" + os.Getenv("GOEXPERIMENT"),
		"GOAMD64=" + os.Getenv("GOAMD64"),
		"GOARM=" + os.Getenv("GOARM"),
	}
	return strings.Join(values, "\x00")
}

func configuredMaxConcurrentExtracts() int {
	const envName = "GO_CODE_GRAPH_MAX_CONCURRENT_EXTRACTS"
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return intgraph.DefaultMaxConcurrentExtracts
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		log.Printf("invalid %s=%q; using default %d", envName, raw, intgraph.DefaultMaxConcurrentExtracts)
		return intgraph.DefaultMaxConcurrentExtracts
	}
	return value
}
