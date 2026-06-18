package gate

import (
	"log"

	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/dispatch"
	internalgate "vrooli-bridge/internal/gate"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	gateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate/gate_v1connect"
)

// Module returns the gate domain's contribution to the API: the generated
// Connect-RPC GateService handler. The gate domain runs cross-OS deployment
// gates (OT-P1-002) by selecting one eligible node per target OS and delegating
// each validation run to the SHARED dispatch service (allowlist + scopes +
// audit) and the runs service (durable lifecycle); this module is the single
// place its proto-free seams are bound to the concrete registry (enumerate +
// OS/revocation), presence (online + protocol compatibility), and dispatch +
// runs. It owns its own durable gate tables, so it re-exports Schema().
func Module(db internalgate.SQLExecutor, clk clock.Clock, registrySvc registry.Service, hub *presence.Hub, dispatchSvc dispatch.Service, runsSvc runs.Service, logger *log.Logger) module.Module {
	svc := internalgate.NewService(
		internalgate.NewSQLiteRepository(db, clk),
		nodeListerAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies gate.Presence (IsOnline + Dispatchable)
		runnerAdapter{dispatchSvc: dispatchSvc, runsSvc: runsSvc},
		clk,
	)
	path, handler := gateconnect.NewGateServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "gate",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the gate domain's SQL contribution so the modules registry
// collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalgate.Schema() }
