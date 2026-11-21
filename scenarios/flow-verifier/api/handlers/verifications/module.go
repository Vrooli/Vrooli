// Package verifications wires the Connect-RPC VerificationsService for
// kicking off verify-pipeline runs. StartVerification runs synchronously
// and returns the per-flow runs.Run rows it persisted via the runs
// domain. GetVerification resolves to the same runs row by id.
package verifications

import (
	"database/sql"
	"log"

	"flow-verifier/internal/module"
	"flow-verifier/internal/runs"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	verificationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications/verifications_v1connect"
)

// Module returns the verifications domain's Connect-RPC contribution.
func Module(db *sql.DB, clk schedule.Clock) module.Module {
	svc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc, log.Default())
}

// ModuleWithService is the test-friendly variant.
func ModuleWithService(svc *runs.Service, logger *log.Logger) module.Module {
	path, handler := verificationsconnect.NewVerificationsServiceHandler(NewConnectHandler(Deps{Runs: svc, Logger: logger}))
	return module.Module{
		Name: "verifications",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — verifications dispatches runs but persistence lives
// in the runs domain.
func Schema() string { return "" }
