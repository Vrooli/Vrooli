package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"{{SCENARIO_ID}}/internal/capabilities"
	"{{SCENARIO_ID}}/internal/modules"
	"{{SCENARIO_ID}}/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "{{SCENARIO_ID}}/handlers/capabilities"
	healthH "{{SCENARIO_ID}}/handlers/health"
	notesH "{{SCENARIO_ID}}/handlers/notes" // EXAMPLE-DOMAIN:notes
)

// runtimeDeps names the external startup effects so the complete application
// bootstrap can be tested without opening a listener or relying on host state.
type runtimeDeps struct {
	preflightRun    func(preflight.Config) bool
	openDatabase    func(context.Context, database.Config) (*database.RoutedDB, error)
	ensureSchemas   func(context.Context, *database.RoutedDB) error
	storageRoots    func() (storage.Paths, error)
	measuresHandler func(*database.RoutedDB) (http.Handler, error)
	authConfig      func(func(string) string) (authn.Config, error)
	runServer       func(apiserver.Config) error
}

func productionRuntimeDeps() runtimeDeps {
	return runtimeDeps{
		preflightRun:    preflight.Run,
		openDatabase:    database.Open,
		ensureSchemas:   ensureScenarioSchemas,
		storageRoots:    scenarioStorageRoots,
		measuresHandler: notesMeasuresHandler,
		authConfig:      authn.FromEnvironment,
		runServer:       apiserver.Run,
	}
}

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. Domain file stores receive the routed roots and select the
// request-appropriate class at their own storage seam.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("{{SCENARIO_ID}}")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve {{SCENARIO_ID}} storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func ensureScenarioSchemas(ctx context.Context, db *database.RoutedDB) error {
	return database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...)
}

func notesMeasuresHandler(db *database.RoutedDB) (http.Handler, error) {
	return notesH.MeasuresHandler(db, schedule.System())
}

func main() {
	if preflight.Run(preflight.Config{ScenarioName: "{{SCENARIO_ID}}"}) {
		return
	}

	deps := productionRuntimeDeps()
	// The calls remain visible in main for API lifecycle validation; tests
	// replace the runner dependency before exercising the startup branches.
	deps.preflightRun = func(preflight.Config) bool { return false }
	deps.runServer = func(config apiserver.Config) error {
		return apiserver.Run(config)
	}
	if err := run(deps); err != nil {
		log.Fatal(err)
	}
}

func run(deps runtimeDeps) error {
	const scenarioName = "{{SCENARIO_ID}}"
	ctx := context.Background()

	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if deps.preflightRun(preflight.Config{ScenarioName: scenarioName}) {
		return nil
	}

	db, err := deps.openDatabase(ctx, database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     scenarioName,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	serverCleanupCalled := false
	defer func() {
		if !serverCleanupCalled {
			_ = db.Close()
		}
	}()

	if err := deps.ensureSchemas(ctx, db); err != nil {
		return fmt.Errorf("schema initialization failed: %w", err)
	}
	primaryFileRoots, err := deps.storageRoots()
	if err != nil {
		return fmt.Errorf("file storage configuration failed: %w", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, scenarioName+"-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		notesH.Module(db, schedule.System(), log.Default()), // EXAMPLE-DOMAIN:notes
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	// EXAMPLE-DOMAIN:notes START
	// /measures is the measures-go serve substrate: the central measures
	// index (measures-health) harvests <prefix>/declarations and the
	// auto-execution path POSTs <prefix>/execute. The notes domain owns the
	// one reference measure (notes.count); a real multi-domain scenario
	// registers each domain's measures on one shared registry here.
	notesMeasures, err := deps.measuresHandler(db)
	if err != nil {
		return fmt.Errorf("measures registry: %w", err)
	}
	rootMux.Handle("/measures/", http.StripPrefix("/measures", notesMeasures))
	// EXAMPLE-DOMAIN:notes END

	rootMux.Handle("/", srv.Handler())
	authConfig, err := deps.authConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("authentication configuration failed: %w", err)
	}
	var sharedAuth *authn.Config
	if authConfig.Enabled() {
		sharedAuth = &authConfig
	}

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := deps.runServer(apiserver.Config{
		Handler:        handler,
		Authentication: sharedAuth,
		Cleanup: func(ctx context.Context) error {
			serverCleanupCalled = true
			return db.Close()
		},
	}); err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
