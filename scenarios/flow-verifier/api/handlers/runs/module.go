// Package runs wires the Connect-RPC RunsService for verification run
// history. Reads from the SQLite-backed runs domain (internal/runs);
// writes happen through the verifications handler, which records via
// the same runs.Service.
package runs

import (
	"database/sql"
	"log"

	"flow-verifier/internal/clock"
	"flow-verifier/internal/module"
	"flow-verifier/internal/runs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs/runs_v1connect"
)

// Module returns the runs domain's Connect-RPC contribution.
func Module(db *sql.DB, clk clock.Clock) module.Module {
	svc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc, log.Default())
}

// ModuleWithService is the test-friendly variant.
func ModuleWithService(svc *runs.Service, logger *log.Logger) module.Module {
	path, handler := runsconnect.NewRunsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "runs",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports runs.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler.
func Schema() string { return runs.Schema() }
