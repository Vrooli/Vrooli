package audits

import (
	"log"

	internalaudits "data-backup-manager/internal/audits"
	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	auditsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits/audits_v1connect"
)

// Module returns the audits domain's contribution to the API. The audit service
// is constructed in main.go (it needs cross-domain lookup adapters for
// targets/destinations plus the engine and source registry), so this module
// takes the already-built service. The schema is static and re-exported via
// Schema().
func Module(svc internalaudits.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := auditsconnect.NewAuditsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "audits",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the audits domain SQL for the modules registry.
func Schema() string { return internalaudits.Schema() }
