package provision

import (
	"log"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/channelsign"
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
// NewService builds the provision domain's application service with its
// proto-free seams bound to the concrete registry / presence / audit services.
// main.go constructs it once and shares the single instance between this
// module's handler and the fleet module's provisioner adapter, so the in-memory
// op waiter/subscriber coordination stays coherent across both call sites.
func NewService(db internalprovision.SQLExecutor, clk clock.Clock, registrySvc registry.Service, hub *presence.Hub, auditSink audit.Sink, signer channelsign.Signer, opts ...internalprovision.Option) internalprovision.Service {
	return internalprovision.NewService(
		internalprovision.NewSQLiteRepository(db, clk),
		nodeReaderAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies provision.Presence via IsOnline
		auditSinkAdapter{sink: auditSink},
		commandPusherAdapter{hub: hub, signer: signer},
		clk,
		opts...,
	)
}

// Module returns the provision domain's contribution to the API given an
// already-built service (NewService).
func Module(svc internalprovision.Service, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
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
