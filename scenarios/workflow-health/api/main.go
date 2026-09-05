package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"workflow-health/internal/modules"
	"workflow-health/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	repocontract "github.com/vrooli/repo-contract-go"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	healthH "workflow-health/handlers/health"
	validationH "workflow-health/handlers/validation"
	workflowsH "workflow-health/handlers/workflows"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "workflow-health"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "workflow-health",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := validationH.RecoverInterrupted(context.Background(), db); err != nil {
		log.Printf("recover interrupted durable validation runs: %v", err)
	}

	repoRoot, repoErr := repocontract.ResolveRepoRoot()
	if repoErr != nil {
		log.Printf("repo root resolution failed; workflow validation will report targets as unresolvable: %v", repoErr)
	}
	syncCtx, cancelSearchRegistration := context.WithCancel(context.Background())
	if repoRoot != "" {
		searchJSONPath := filepath.Join(repoRoot, "scenarios", "workflow-health", ".vrooli", "search.json")
		go searchregister.Register(syncCtx, searchregister.Config{
			ScenarioID:     "workflow-health",
			SearchFilePath: searchJSONPath,
			Logger:         log.Default(),
		})
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "workflow-health-api", "1.0.0"),
		validationH.Module(log.Default(), repoRoot, db),
		workflowsH.Module(log.Default()),
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

	// WaitValidationRun is the sole intentionally blocking provider endpoint.
	// Its 15-minute Workflow phase budget receives one minute of bounded
	// transport headroom; Start/Get/Abort return promptly and do not rely on it.
	if err := apiserver.Run(apiserver.Config{
		Handler:      handler,
		WriteTimeout: 16 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			cancelSearchRegistration()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
