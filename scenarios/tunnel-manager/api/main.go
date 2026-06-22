package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tunnel-manager/internal/clock"
	internalexposure "tunnel-manager/internal/exposure"
	"tunnel-manager/internal/modules"
	internalprobes "tunnel-manager/internal/probes"
	internalrecovery "tunnel-manager/internal/recovery"
	internalroutes "tunnel-manager/internal/routes"
	"tunnel-manager/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	auditH "tunnel-manager/handlers/audit"
	configH "tunnel-manager/handlers/config"
	exposureH "tunnel-manager/handlers/exposure"
	healthH "tunnel-manager/handlers/health"
	probesH "tunnel-manager/handlers/probes"
	recoveryH "tunnel-manager/handlers/recovery"
	routesH "tunnel-manager/handlers/routes"
	tunnelH "tunnel-manager/handlers/tunnel"
)

type lifecycleScheduler interface {
	Run(context.Context)
}

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
	scenarioID, err := storage.ScenarioNamespace("tunnel-manager")
	if err != nil {
		return "", fmt.Errorf("resolve tunnel-manager storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"tunnel-manager.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve tunnel-manager db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "tunnel-manager"}) {
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

	clk := clock.System{}
	logger := log.Default()
	routesSvc := internalroutes.NewService(internalroutes.NewSQLiteRepository(db, clk))
	configSvc := configH.NewProductionService(db, clk, routesSvc)
	exposureSvc := exposureH.NewProductionService(db, clk, routesSvc, configSvc)
	probesSvc := probesH.NewProductionService(routesSvc, db, clk)
	recoverySvc := recoveryH.NewProductionService(db, clk)

	var schedulerStops []func(context.Context)
	if exposureSchedulerEnabledFromEnv() {
		scheduler, err := internalexposure.NewScheduler(internalexposure.SchedulerConfig{
			Service:  exposureSvc,
			Interval: exposureSchedulerIntervalFromEnv(),
			Logger:   logger,
		})
		if err != nil {
			log.Fatalf("exposure scheduler initialization failed: %v", err)
		}
		schedulerStops = append(schedulerStops, startScheduler(scheduler))
	}

	if probeSchedulerEnabledFromEnv() {
		scheduler, err := internalprobes.NewScheduler(internalprobes.SchedulerConfig{
			Service:  probesSvc,
			Interval: probeSchedulerIntervalFromEnv(),
			Logger:   logger,
		})
		if err != nil {
			log.Fatalf("probe scheduler initialization failed: %v", err)
		}
		schedulerStops = append(schedulerStops, startScheduler(scheduler))
	}

	if recoverySchedulerEnabledFromEnv() {
		scheduler, err := internalrecovery.NewScheduler(internalrecovery.SchedulerConfig{
			Service:  recoverySvc,
			Interval: recoverySchedulerIntervalFromEnv(),
			Logger:   logger,
		})
		if err != nil {
			log.Fatalf("recovery scheduler initialization failed: %v", err)
		}
		schedulerStops = append(schedulerStops, startScheduler(scheduler))
	}

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "tunnel-manager-api", "1.0.0"),
		routesH.Module(db, clk, logger),
		auditH.ModuleWithRoutes(routesSvc, logger),
		configH.ModuleWithService(configSvc, logger),
		exposureH.ModuleWithService(exposureSvc, logger),
		probesH.ModuleWithService(probesSvc, logger),
		recoveryH.ModuleWithService(recoverySvc, logger),
		tunnelH.Module(db, clk, logger),
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
			for i := len(schedulerStops) - 1; i >= 0; i-- {
				schedulerStops[i](ctx)
			}
			return errors.Join(ctx.Err(), db.Close())
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func exposureSchedulerEnabledFromEnv() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_EXPOSURE_SCHEDULER_DISABLED")))
	return raw != "1" && raw != "true" && raw != "yes"
}

func exposureSchedulerIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL"))
	if raw == "" {
		return internalexposure.DefaultReconcileInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("invalid TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL=%q; using %s", raw, internalexposure.DefaultReconcileInterval)
		return internalexposure.DefaultReconcileInterval
	}
	return d
}

func probeSchedulerEnabledFromEnv() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_PROBE_SCHEDULER_DISABLED")))
	return raw != "1" && raw != "true" && raw != "yes"
}

func probeSchedulerIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_PROBE_INTERVAL"))
	if raw == "" {
		return internalprobes.DefaultProbeInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("invalid TUNNEL_MANAGER_PROBE_INTERVAL=%q; using %s", raw, internalprobes.DefaultProbeInterval)
		return internalprobes.DefaultProbeInterval
	}
	return d
}

func recoverySchedulerEnabledFromEnv() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_RECOVERY_SCHEDULER_DISABLED")))
	return raw != "1" && raw != "true" && raw != "yes"
}

func recoverySchedulerIntervalFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("TUNNEL_MANAGER_RECOVERY_EVALUATE_INTERVAL"))
	if raw == "" {
		return internalrecovery.DefaultEvaluationInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("invalid TUNNEL_MANAGER_RECOVERY_EVALUATE_INTERVAL=%q; using %s", raw, internalrecovery.DefaultEvaluationInterval)
		return internalrecovery.DefaultEvaluationInterval
	}
	return d
}

func startScheduler(s lifecycleScheduler) func(context.Context) {
	schedulerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.Run(schedulerCtx)
	}()
	return func(ctx context.Context) {
		cancel()
		waitForScheduler(ctx, done)
	}
}

func waitForScheduler(ctx context.Context, done <-chan struct{}) {
	select {
	case <-done:
	case <-ctx.Done():
	}
}
