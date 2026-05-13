package main

import (
	"context"
	"log"
	"os"

	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/flows"
	"flow-verifier/internal/modules"
	"flow-verifier/internal/runs"
	"flow-verifier/internal/scenarios"
	"flow-verifier/internal/server"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	artifactsH "flow-verifier/handlers/artifacts"
	flowsH "flow-verifier/handlers/flows"
	healthH "flow-verifier/handlers/health"
	runsH "flow-verifier/handlers/runs"
	scenariosH "flow-verifier/handlers/scenarios"
	settingsH "flow-verifier/handlers/settings"
	verificationsH "flow-verifier/handlers/verifications"
)

// flowsListerAdapter bridges flows.List into the FlowLister contract
// the scenarios package depends on. Kept inline (vs. a dedicated
// package) because it's a one-method shim with no logic to test on
// its own — every caller is right here in main.go.
type flowsListerAdapter struct{}

func (flowsListerAdapter) List(root string) ([]flows.Summary, error) {
	return flows.List(root, "", "")
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "flow-verifier"}) {
		return
	}

	dsn, err := localdb.DefaultDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Scenarios discovery anchors on the Vrooli repo root. CWD-walk
	// finds it in development; FLOW_VERIFIER_VROOLI_ROOT pins it in
	// production deploys or test harnesses. Fatal-on-missing — the
	// product can't render anything useful without a root.
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cwd: %v", err)
	}
	vrooliRoot, err := scenarios.ResolveVrooliRoot(cwd, os.Getenv)
	if err != nil {
		log.Fatalf("scenarios root: %v", err)
	}
	scenariosSvc := scenarios.NewService(vrooliRoot, flowsListerAdapter{})
	log.Printf("scenarios: Vrooli root resolved to %s", vrooliRoot)

	// Artifacts service is constructed once and shared with the
	// scenarios handler so the streaming GenerateScenarioArtifacts
	// RPC and the per-flow artifacts RPCs see the same generator.
	runsSvc := runs.NewService(runs.NewSQLiteRepository(db, clock.System{}))
	artifactsSvc := artifactsH.NewService(runsSvc)

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "flow-verifier-api", "1.0.0"),
		flowsH.Module(scenariosSvc),
		scenariosH.Module(scenariosSvc, artifactsSvc),
		verificationsH.Module(db, clock.System{}),
		artifactsH.ModuleWithDeps(artifactsSvc, scenariosSvc, log.Default()),
		runsH.Module(db, clock.System{}),
		settingsH.Module(db, clock.System{}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
