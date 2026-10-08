package runs

import (
	"log"

	"data-backup-manager/internal/module"
	internalruns "data-backup-manager/internal/runs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs/runs_v1connect"
)

// Module returns the runs domain's contribution to the API. Unlike the
// reference domains, the run service is constructed in main.go (it needs
// cross-domain lookup adapters for plans/targets/destinations plus the engine
// and source registry), so this module takes the already-built service. The
// verified lookup (proven-restorable rollup) is likewise composed in main.go
// over the restores service. The schema is static and re-exported via Schema().
func Module(svc internalruns.Service, verified VerifiedLookup, logger *log.Logger) module.Module {
	connectPath, connectHandler := runsconnect.NewRunsServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Verified: verified,
		Logger:   logger,
	}))
	return module.Module{
		Name: "runs",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the runs domain SQL for the modules registry.
func Schema() string { return internalruns.Schema() }
