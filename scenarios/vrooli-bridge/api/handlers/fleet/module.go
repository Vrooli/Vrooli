package fleet

import (
	"log"

	internalfleet "vrooli-bridge/internal/fleet"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/provision"
	"vrooli-bridge/internal/registry"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet/fleet_v1connect"
)

// Module returns the fleet domain's contribution to the API: the generated
// Connect-RPC FleetService handler. The fleet domain rolls the whole fleet to a
// target revision (OT-P1-001) by delegating per-node provisioning to the
// provision service; this module is the single place its proto-free seams are
// bound to the concrete registry (enumerate + revocation), presence (online +
// protocol compatibility), and provision (dispatch a privileged op). It owns its
// own durable rollout tables, so it re-exports Schema().
func Module(db internalfleet.SQLExecutor, clk schedule.Clock, registrySvc registry.Service, hub *presence.Hub, provisionSvc provision.Service, resolver internalfleet.RevisionResolver, logger *log.Logger) module.Module {
	opts := []internalfleet.Option{}
	if resolver != nil {
		opts = append(opts, internalfleet.WithRevisionResolver(resolver))
	}
	svc := internalfleet.NewService(
		internalfleet.NewSQLiteRepository(db, clk),
		nodeListerAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies fleet.Presence (IsOnline + Dispatchable)
		provisionerAdapter{svc: provisionSvc},
		clk,
		opts...,
	)
	path, handler := fleetconnect.NewFleetServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "fleet",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the fleet domain's SQL contribution so the modules registry
// collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalfleet.Schema() }
