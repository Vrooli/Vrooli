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

	"code-facts/internal/modules"
	"code-facts/internal/registration"
	"code-facts/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	factsH "code-facts/handlers/facts"
	healthH "code-facts/handlers/health"
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
	scenarioID, err := storage.ScenarioNamespace("code-facts")
	if err != nil {
		return "", fmt.Errorf("resolve code-facts storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"code-facts.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve code-facts db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

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

	ctx := context.Background()
	if err := factsH.MigrateSchema(ctx, db.Primary()); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	cacheMaxBytes, err := cacheMaxBytesFromEnv()
	if err != nil {
		log.Fatalf("cache configuration failed: %v", err)
	}
	go func() {
		time.Sleep(10 * time.Second)
		sweep, err := factsH.SweepCache(context.Background(), db.Primary(), cacheMaxBytes)
		if err != nil {
			log.Printf("code-facts cache startup sweep failed: %v", err)
			return
		}
		log.Printf("code-facts cache startup sweep: stale_rows=%d evicted_rows=%d reclaimed_bytes=%d remaining_bytes=%d max_bytes=%d",
			sweep.StaleRows, sweep.EvictedRows, sweep.ReclaimedByte, sweep.RemainingByte, cacheMaxBytes)
	}()

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "code-facts-api", "1.0.0", func(ctx context.Context) (map[string]any, error) {
			return factsH.CacheMetrics(ctx, db.Primary(), cacheMaxBytes)
		}),
		factsH.Module(db.Primary(), log.Default(), cacheMaxBytes),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())
	go registration.Register(ctx, searchDescriptorPath(), log.Default())

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
