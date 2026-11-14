package restores

import (
	"log"

	"data-backup-manager/internal/module"
	internalrestores "data-backup-manager/internal/restores"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	restoresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores/restores_v1connect"
)

// Module returns the restores domain's contribution to the API. The restore
// service is constructed in main.go (it needs cross-domain lookup adapters for
// targets/destinations plus the engine and source registry), so this module
// takes the already-built service. The schema is static and re-exported via
// Schema().
func Module(svc internalrestores.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := restoresconnect.NewRestoresServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "restores",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the restores domain SQL for the modules registry.
func Schema() string { return internalrestores.Schema() }
