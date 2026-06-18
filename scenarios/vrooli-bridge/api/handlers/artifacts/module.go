package artifacts

import (
	"log"

	internalartifacts "vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/registry"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
)

// Module returns the artifacts domain's contribution to the API: the generated
// Connect-RPC ArtifactsService handler. The artifacts domain ships non-git
// artifacts to nodes via device-sync-hub directed delivery (OT-P1-003); this
// module is the single place its proto-free seams are bound to the concrete
// registry (revocation) and the device-sync-hub directed-delivery client. It
// owns its own durable distributions table, so it re-exports Schema().
func Module(db internalartifacts.SQLExecutor, clk clock.Clock, registrySvc registry.Service, logger *log.Logger) module.Module {
	svc := internalartifacts.NewService(
		internalartifacts.NewSQLiteRepository(db, clk),
		nodeReaderAdapter{svc: registrySvc},
		deviceSyncDelivery{},
		clk,
	)
	path, handler := artifactsconnect.NewArtifactsServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "artifacts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the artifacts domain's SQL contribution so the modules
// registry collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalartifacts.Schema() }
