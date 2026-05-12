package main

import (
	"context"
	"log"

	"flow-verifier/internal/clock"
	localdb "flow-verifier/internal/database"
	"flow-verifier/internal/modules"
	"flow-verifier/internal/server"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	flowsH "flow-verifier/handlers/flows"
	healthH "flow-verifier/handlers/health"
	runsH "flow-verifier/handlers/runs"
	verificationsH "flow-verifier/handlers/verifications"
)

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

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.Default()},
		healthH.Module(db, "flow-verifier-api", "1.0.0"),
		flowsH.Module(),
		verificationsH.Module(db, clock.System{}),
		runsH.Module(db, clock.System{}),
	)

	if err := apiserver.Run(apiserver.Config{
		Handler: srv.Handler(),
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
