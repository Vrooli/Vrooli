package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/maintenance"
	"vrooli-memory/internal/modules"
	"vrooli-memory/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/provenance"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"

	facetsH "vrooli-memory/handlers/facets"
	forestH "vrooli-memory/handlers/forest"
	harnessH "vrooli-memory/handlers/harness"
	healthH "vrooli-memory/handlers/health"
	journalH "vrooli-memory/handlers/journal"
	recallH "vrooli-memory/handlers/recall"
	rulesH "vrooli-memory/handlers/rules"
	scopesH "vrooli-memory/handlers/scopes"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("vrooli-memory")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve vrooli-memory storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "vrooli-memory"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "vrooli-memory",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	ledger, err := ledgerclient.New(context.Background())
	if err != nil {
		log.Fatalf("source-ledger client configuration failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "vrooli-memory-api", "1.0.0", maintenance.NewSQLiteStore(db.Primary()), db.Primary()),
		journalH.Module(ledger, log.Default()),
		facetsH.Module(ledger, log.Default()),
		forestH.Module(ledger, log.Default()),
		recallH.Module(ledger, log.Default()),
		harnessH.Module(db, fileRoots, ledger, log.Default(), ledger, schedule.System()),
		rulesH.Module(ledger, log.Default()),
		scopesH.Module(ledger, log.Default()),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(provenance.Middleware(provenance.CLIUtilVerifier{})(rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
