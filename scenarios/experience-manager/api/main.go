package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"experience-manager/internal/modules"
	"experience-manager/internal/reconcile"
	"experience-manager/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	capabilitiesH "experience-manager/handlers/capabilities"
	healthH "experience-manager/handlers/health"
	studioH "experience-manager/handlers/studio"
	validationH "experience-manager/handlers/validation"
)

// repoRoot resolves the Vrooli repo root: REPO_ROOT env when set, else walking
// up from the scenario directory this binary serves.
func repoRoot() string {
	if root := strings.TrimSpace(os.Getenv("REPO_ROOT")); root != "" {
		return root
	}
	if dir, err := os.Getwd(); err == nil {
		for d := dir; d != "/"; d = filepath.Dir(d) {
			if _, err := os.Stat(filepath.Join(d, "scenarios")); err == nil {
				if _, err := os.Stat(filepath.Join(d, "packages")); err == nil {
					return d
				}
			}
		}
	}
	return "."
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "experience-manager"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "experience-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := reconcile.EnsureMigrations(context.Background(), db.Primary()); err != nil {
		log.Fatalf("reconcile evidence migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "experience-manager-api", "1.0.0"),
		capabilitiesH.Module(repoRoot()),
		studioH.Module(log.Default(), repoRoot(), studioH.WithDatabase(db)),
		validationH.Module(log.Default(), repoRoot(), nil, validationH.WithDatabase(db)),
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
		// Full-fidelity validation can capture the complete declared state
		// matrix before returning its evidence envelope.
		WriteTimeout: 10 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
