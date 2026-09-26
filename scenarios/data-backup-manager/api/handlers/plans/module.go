package plans

import (
	"log"

	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans/plans_v1connect"

	internalplans "data-backup-manager/internal/plans"
)

// Module returns the plans domain's contribution to the API: the generated
// PlansService Connect-RPC handler. The fully-wired service (including the
// coverage guard) is built in the composition root (main.go) and passed in —
// mirroring discovery/runs/restores — because the guard composes a cross-domain
// seam (discovery) the leaf handler must not know about.
func Module(svc internalplans.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := plansconnect.NewPlansServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "plans",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the plans domain SQL so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalplans.Schema() }
