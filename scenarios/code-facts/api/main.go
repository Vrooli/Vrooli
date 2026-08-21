package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code-facts/internal/catalog"
	"code-facts/internal/modules"
	"code-facts/internal/registration"
	"code-facts/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	factsH "code-facts/handlers/facts"
	healthH "code-facts/handlers/health"
)

func cacheMaxBytesFromEnv() (int64, error) {
	raw := strings.TrimSpace(os.Getenv("CODE_FACTS_CACHE_MAX_BYTES"))
	if raw == "" {
		return factsH.DefaultCacheMaxBytes(), nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse CODE_FACTS_CACHE_MAX_BYTES: %w", err)
	}
	if value < 0 {
		return 0, fmt.Errorf("CODE_FACTS_CACHE_MAX_BYTES must be >= 0")
	}
	return value, nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "code-facts"}) {
		return
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)

	db, err := database.Open(context.Background(), database.Config{
		Driver:   database.DriverSQLite,
		Scenario: "code-facts",
		// WAL-backed indexed reads must not queue behind one long FTS request or
		// maintenance batch; writes remain transactionally serialized by SQLite.
		MaxOpenConns: 8,
		MaxIdleConns: 4,
	})
	if err != nil {
		logger.Fatalf("Database connection failed: %v", err)
	}

	ctx := context.Background()
	if err := factsH.MigrateSchema(ctx, db.Primary()); err != nil {
		logger.Fatalf("schema migration failed: %v", err)
	}
	if err := catalog.Migrate(ctx, db.Primary()); err != nil {
		logger.Fatalf("catalog schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		logger.Fatalf("schema initialization failed: %v", err)
	}
	cacheMaxBytes, err := cacheMaxBytesFromEnv()
	if err != nil {
		logger.Fatalf("cache configuration failed: %v", err)
	}
	searchTokens := registration.NewTokenStore()
	admission := factsH.NewAdmission()
	go func() {
		time.Sleep(10 * time.Second)
		sweep, err := factsH.SweepCache(context.Background(), db.Primary(), cacheMaxBytes)
		if err != nil {
			logger.Printf("code-facts cache startup sweep failed: %v", err)
			return
		}
		logger.Printf("code-facts cache startup sweep: stale_rows=%d evicted_rows=%d reclaimed_bytes=%d remaining_bytes=%d max_bytes=%d",
			sweep.StaleRows, sweep.EvictedRows, sweep.ReclaimedByte, sweep.RemainingByte, cacheMaxBytes)
	}()

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "code-facts-api", "1.0.0", func(ctx context.Context) (map[string]any, error) {
			return factsH.OperationalMetrics(ctx, db.Primary(), cacheMaxBytes, admission)
		}),
		factsH.Module(db.Primary(), logger, cacheMaxBytes, admission, os.Getenv("CODE_FACTS_INDEX_CONTROL_TOKEN"), searchTokens.Matches),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())
	go func() {
		// The lifecycle registry publishes this scenario only after health turns
		// green. Delay endpoint-probed self-registration until that publication
		// window; registering immediately creates a deterministic startup race.
		timer := time.NewTimer(20 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			registration.Register(ctx, searchDescriptorPath(), logger, searchTokens)
		}
	}()

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		logger.Fatalf("Server error: %v", err)
	}
}

func searchDescriptorPath() string {
	if path := strings.TrimSpace(os.Getenv("CODE_FACTS_SEARCH_FILE")); path != "" {
		return path
	}
	for _, path := range []string{filepath.Join("..", ".vrooli", "search.json"), filepath.Join(".vrooli", "search.json")} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return filepath.Join("..", ".vrooli", "search.json")
}
