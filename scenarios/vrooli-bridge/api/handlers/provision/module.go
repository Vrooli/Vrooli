package provision

import (
	"log"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	"vrooli-bridge/internal/presence"
	internalprovision "vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/registry"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	provisionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision/provision_v1connect"
)

// Module returns the provision domain's contribution to the API: the generated
// Connect-RPC ProvisionService handler. The provision domain is the PRIVILEGED
// tier (OT-P0-006); this module is the single place its proto-free seams are
// bound to the concrete registry (node revocation), presence (online + the
// channel push), and audit (accountability). It owns its own durable op tables,
// so it re-exports Schema().
func Module(db internalprovision.SQLExecutor, clk clock.Clock, registrySvc registry.Service, hub *presence.Hub, auditSink audit.Sink, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
	svc := internalprovision.NewService(
		internalprovision.NewSQLiteRepository(db, clk),
		nodeReaderAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies provision.Presence via IsOnline
		auditSinkAdapter{sink: auditSink},
		commandPusherAdapter{hub: hub},
		clk,
	)
	path, handler := provisionconnect.NewProvisionServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Verifier: verifier,
		Logger:   logger,
	}))
	return module.Module{
		Name: "provision",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the provision domain's SQL contribution so the modules
// registry collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalprovision.Schema() }
