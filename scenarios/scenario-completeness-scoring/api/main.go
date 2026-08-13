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

	"scenario-completeness-scoring/internal/modules"
	"scenario-completeness-scoring/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	healthH "scenario-completeness-scoring/handlers/health"
	measuresH "scenario-completeness-scoring/handlers/measures"
	scoringH "scenario-completeness-scoring/handlers/scoring"
	internalscoring "scenario-completeness-scoring/internal/scoring"
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
	scenarioID, err := storage.ScenarioNamespace("scenario-completeness-scoring")
	if err != nil {
		return "", fmt.Errorf("resolve scenario-completeness-scoring storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"scenario-completeness-scoring.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve scenario-completeness-scoring db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "scenario-completeness-scoring"}) {
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

	// Guarded, idempotent column-evolution migrations run BEFORE EnsureSchemas'
	// drift check, which would otherwise fail on a pre-existing score_snapshots
	// table missing the recency columns (CREATE TABLE IF NOT EXISTS cannot add a
	// column to a data-bearing table). On a fresh database this is a no-op and
	// EnsureSchemas creates the table complete.
	if err := internalscoring.Migrate(context.Background(), db.Primary()); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	scorer, err := internalscoring.New()
	if err != nil {
		log.Fatalf("scoring service init failed: %v", err)
	}
	sweepScorer, err := internalscoring.New(internalscoring.WithImportance(nil))
	if err != nil {
		log.Fatalf("sweep scoring service init failed: %v", err)
	}
	snapshots := internalscoring.NewSQLiteSnapshotRepository(db)
	sweepCtx, cancelSweep := context.WithCancel(context.Background())
	sweepInterval := scoreSweepIntervalFromEnv()
	sweeper, err := internalscoring.NewSweeper(internalscoring.SweeperConfig{
		ScenariosRoot: scorer.ScenariosRoot(),
		Repository:    snapshots,
		Scorer:        sweepScorer,
		Logger:        log.Default(),
		Concurrency:   scoreSweepConcurrencyFromEnv(),
		Interval:      sweepInterval,
		InitialJitter: scoreSweepStartJitterFromEnv(sweepInterval),
	})
	if err != nil {
		log.Fatalf("score sweeper init failed: %v", err)
	}
	sweepDone := make(chan struct{})
	if !scoreSweepDisabledFromEnv() {
		go func() {
			defer close(sweepDone)
			sweeper.RunLoop(sweepCtx)
		}()
	} else {
		close(sweepDone)
	}

	// Importance enrichment runs on a separate, slower cadence than the fast
	// score sweep so it never adds the ~1s/scenario centrality fetch to the
	// digest-gated hot path. It always re-scores (AlwaysScore) so importance is
	// refreshed even for stable scenarios whose tree never changes, and upserts
	// importance onto the existing (scenario, digest) snapshot rows.
	importanceInterval := importanceRefreshIntervalFromEnv()
	importanceSweeper, err := internalscoring.NewSweeper(internalscoring.SweeperConfig{
		ScenariosRoot: scorer.ScenariosRoot(),
		Repository:    snapshots,
		Scorer:        scorer,
		Logger:        log.Default(),
		Concurrency:   importanceRefreshConcurrencyFromEnv(),
		Interval:      importanceInterval,
		InitialJitter: scoreSweepStartJitterFromEnv(importanceInterval),
		AlwaysScore:   true,
	})
	if err != nil {
		log.Fatalf("importance refresh sweeper init failed: %v", err)
	}
	importanceDone := make(chan struct{})
	if !importanceRefreshDisabledFromEnv() {
		go func() {
			defer close(importanceDone)
			importanceSweeper.RunLoop(sweepCtx)
		}()
	} else {
		close(importanceDone)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "scenario-completeness-scoring-api", "1.0.0"),
		scoringH.Module(scorer, snapshots, log.Default()),
		measuresH.Module(snapshots, time.Now),
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
			cancelSweep()
			select {
			case <-sweepDone:
			case <-ctx.Done():
			}
			select {
			case <-importanceDone:
			case <-ctx.Done():
			}
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func scoreSweepIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("SCS_SCORE_SWEEP_INTERVAL"))
	if raw == "" {
		return time.Hour
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return time.Hour
	}
	return d
}

func scoreSweepConcurrencyFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("SCS_SCORE_SWEEP_CONCURRENCY"))
	if raw == "" {
		return 4
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 4
	}
	if n > 64 {
		return 64
	}
	return n
}

func scoreSweepStartJitterFromEnv(interval time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv("SCS_SCORE_SWEEP_START_JITTER"))
	if raw != "" {
		d, err := time.ParseDuration(raw)
		if err == nil && d >= 0 {
			return d
		}
	}
	if interval <= 0 {
		interval = time.Hour
	}
	maxJitter := interval / 10
	if maxJitter > time.Minute {
		maxJitter = time.Minute
	}
	if maxJitter <= 0 {
		return 0
	}
	seed := time.Now().UnixNano() + int64(os.Getpid())
	if seed < 0 {
		seed = -seed
	}
	return time.Duration(seed % int64(maxJitter+1))
}

func scoreSweepDisabledFromEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SCS_SCORE_SWEEP_DISABLED"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func importanceRefreshIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("SCS_IMPORTANCE_REFRESH_INTERVAL"))
	if raw == "" {
		return 6 * time.Hour
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 6 * time.Hour
	}
	return d
}

func importanceRefreshConcurrencyFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("SCS_IMPORTANCE_REFRESH_CONCURRENCY"))
	if raw == "" {
		return 2
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 2
	}
	if n > 16 {
		return 16
	}
	return n
}

func importanceRefreshDisabledFromEnv() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SCS_IMPORTANCE_REFRESH_DISABLED"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
