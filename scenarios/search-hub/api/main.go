package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"search-hub/internal/clock"
	"search-hub/internal/eval"
	"search-hub/internal/modules"
	"search-hub/internal/server"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	apihealth "github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite"

	evalH "search-hub/handlers/eval"
	healthH "search-hub/handlers/health"
	metricsH "search-hub/handlers/metrics"
	registryH "search-hub/handlers/registry"
	routingH "search-hub/handlers/routing"
	validationH "search-hub/handlers/validation"
)

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

type serviceResource struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// requiredResourceChecks derives readiness dependencies from the scenario
// manifest. Search Hub observes resource state but does not own remediation;
// adding a required resource therefore changes one declarative file, not the
// health handler. Resource-specific health paths are tied to resource type,
// never to a provider scenario id.
func requiredResourceChecks(repoRoot string) []apihealth.Checker {
	raw, err := os.ReadFile(filepath.Join(repoRoot, "scenarios", "search-hub", ".vrooli", "service.json"))
	if err != nil {
		log.Printf("health: cannot read service resource manifest: %v", err)
		return nil
	}
	var manifest struct {
		Dependencies struct {
			Resources map[string]serviceResource `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		log.Printf("health: cannot parse service resource manifest: %v", err)
		return nil
	}
	checks := make([]apihealth.Checker, 0)
	for name, resource := range manifest.Dependencies.Resources {
		if !resource.Required {
			continue
		}
		path := "/healthz"
		if resource.Type == "ollama" {
			path = "/api/tags"
		}
		envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_URL"
		fallback := "http://127.0.0.1:6333"
		if resource.Type == "ollama" {
			fallback = "http://127.0.0.1:11434"
		}
		checks = append(checks, apihealth.HTTP(name, envOrDefault(envKey, fallback)+path))
	}
	return checks
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
		storage.Options{ScenarioID: "search-hub"},
		storage.ClassData,
		"search-hub.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve search-hub db path: %w", err)
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
	if preflight.Run(preflight.Config{ScenarioName: "search-hub"}) {
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
	repoRoot, err := repocontract.ResolveRepoRoot()
	if err != nil {
		log.Fatalf("repo root resolution failed: %v", err)
	}

	if err := metricsH.Migrate(context.Background(), db); err != nil {
		log.Fatalf("metrics schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Register the SHRINKING set of starter eval suites search-hub still ships
	// (idempotent upsert by suite_id) for the one provider that has not yet
	// adopted corpus self-registration (swarm-manager.records). Adopted providers
	// (cli-health, knowledge-observatory, ui-health) self-register both their
	// descriptor AND their tests corpus from their own .vrooli/search.json at
	// boot, so they need no seed here; a suite only references a provider_id and
	// the runner resolves it at run time. When swarm-manager adopts, this call and
	// the eval/seeds mechanism are deleted. See internal/eval/seeds.go.
	if err := eval.RegisterSeeds(context.Background(), eval.NewSQLiteStore(db, clock.System{})); err != nil {
		log.Fatalf("eval seed registration failed: %v", err)
	}

	// The metrics domain owns the query_telemetry store. Its Recorder bridge is
	// injected into the routing module so each federated query records telemetry
	// (Phase 7), while the routing handler stays free of any metrics-store import.
	telemetryRecorder := metricsH.Recorder(db, clock.System{}, log.Default())
	router := routingH.NewRouter(db, clock.System{}, log.Default(), telemetryRecorder)
	routingH.StartRecoveryProbes(router, log.Default())
	resourceChecks := requiredResourceChecks(repoRoot)
	resourceChecks = append(resourceChecks, apihealth.Func("federation", func(ctx context.Context) error {
		share, breached, err := router.CircuitOpenQuorum(ctx)
		if err != nil {
			return err
		}
		if breached {
			return fmt.Errorf("circuit-open share %.1f%% meets federation quorum %.1f%%", share*100, routingH.CircuitOpenQuorumThreshold*100)
		}
		return nil
	}))

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "search-hub-api", "1.0.0", resourceChecks...),
		metricsH.Module(db, clock.System{}, log.Default()),
		registryH.Module(db, clock.System{}, log.Default()),
		routingH.ModuleWithRouter(router, log.Default()),
		evalH.Module(db, clock.System{}, log.Default()),
		validationH.Module(log.Default(), repoRoot, db, clock.System{}),
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
		// EvalService.RunSuite is a synchronous, long-running RPC: it fans out one
		// HTTP call per golden case to the target provider, and a provider whose
		// active reranker leg is an LLM (seconds per query) can push a full suite
		// well past the 30s default. Give the server enough headroom to finish a
		// suite run rather than resetting the connection mid-fan-out (which would
		// discard the immutable run record the harness exists to produce).
		WriteTimeout: 15 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
