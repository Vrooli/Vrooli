package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"security-health/internal/dependencies"
	"security-health/internal/dependencies/aisearch"
	"security-health/internal/modules"
	"security-health/internal/server"
	"security-health/internal/validationcache"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	dependenciesH "security-health/handlers/dependencies"
	healthH "security-health/handlers/health"
	reindexH "security-health/handlers/reindex"
	validationH "security-health/handlers/validation"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "security-health"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "security-health",
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
	depStore := dependencies.NewStore(db)
	depDeps := dependencies.Deps{
		RepoRoot: repoRoot,
		Store:    depStore,
		Clock:    schedule.System(),
	}
	// The semantic index is the optional AI-ranking overlay (Ollama embeddings +
	// Qdrant). NewFromConfig returns nil when disabled; only attach a non-nil
	// index so the service's nil-check (TEXT-only) stays correct (avoid the
	// typed-nil interface trap). When attached, search ranks MODE_AI by vector
	// similarity and degrades to TEXT if the backends are down.
	aiCfg := aisearch.LoadConfigFromEnv()
	resolveCtx, resolveCancel := context.WithTimeout(context.Background(), 10*time.Second)
	resolvedAICfg, err := aisearch.ResolveConfigEmbedding(resolveCtx, aiCfg)
	resolveCancel()
	if err != nil {
		logger.Printf("[security-health] semantic index policy resolution failed (search on TEXT): %v", err)
	} else if idx := aisearch.NewFromConfig(resolvedAICfg); idx != nil {
		depDeps.Index = idx
	}
	depService := dependencies.NewService(depDeps)
	// Create the Qdrant collection up front (idempotent, best-effort) so the
	// first reconcile can populate it without a cold-start miss.
	if err := depService.EnsureIndex(context.Background()); err != nil {
		logger.Printf("[security-health] semantic index unavailable at startup (search on TEXT): %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: logger},
		healthH.Module(db, "security-health-api", "1.0.0"),
		validationH.Module(validationH.ModuleDeps{
			Logger: logger, RepoRoot: repoRoot,
			EvidenceStore: validationcache.New(db), OSVReportCache: depStore,
		}),
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
		// ValidateScenario runs gosec/govulncheck/osv-scanner/gitleaks
		// synchronously inside the RPC, which routinely exceeds the api-core
		// default 30s WriteTimeout (a warm test-genie scan alone is ~31s; cold
		// vuln-DB caches push it higher). Without this override the server kills
		// the connection mid-response ("unexpected EOF"), silently breaking the
		// test-genie → security-health producer path. Match the CLI-side ceiling.
		WriteTimeout: 5 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			reconcileCancel()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// EnvReconcileInterval overrides the base reconcile cadence (Go duration, e.g.
// "5m", "10m"). Falls back to defaultReconcileInterval when unset/invalid.
const EnvReconcileInterval = "SECURITY_HEALTH_RECONCILE_INTERVAL"

// defaultReconcileInterval is the periodic fleet-reconcile cadence. The actual
// wait is this plus a per-tick jitter so a fleet of self-monitoring scenarios
// doesn't burst on an aligned boundary.
const defaultReconcileInterval = 5 * time.Minute

// runReconcileLoop drives a periodic fleet reconcile. It runs once shortly
// after boot (so a freshly-started scenario has an index) and then every
// reconcileInterval (+jitter) until the context is cancelled. Per-scenario
// osv-scanner results are content-cached (see internal/dependencies), so a
// steady-state reconcile re-scans only changed scenarios (and re-scans
// everything at most once per day to pick up newly-published vulnerabilities).
// Reconcile failures are logged and retried on the next tick — a transient
// scanner hiccup must never crash the server.
func runReconcileLoop(ctx context.Context, svc *dependencies.Service, logger *log.Logger) {
	interval := loadReconcileInterval(logger)
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
			timer.Reset(interval + reconcileJitter(interval))
		}
	}
}

// loadReconcileInterval reads the cadence from the environment, falling back to
// the default for an empty/invalid/non-positive value.
func loadReconcileInterval(logger *log.Logger) time.Duration {
	raw := strings.TrimSpace(os.Getenv(EnvReconcileInterval))
	if raw == "" {
		return defaultReconcileInterval
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		logger.Printf("[security-health] invalid %s=%q, using default %s", EnvReconcileInterval, raw, defaultReconcileInterval)
		return defaultReconcileInterval
	}
	return d
}

// reconcileJitter returns a random offset in [0, interval/4) so reconcile ticks
// spread out instead of firing on an aligned boundary across the fleet.
func reconcileJitter(interval time.Duration) time.Duration {
	span := interval / 4
	if span <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(span)))
}
