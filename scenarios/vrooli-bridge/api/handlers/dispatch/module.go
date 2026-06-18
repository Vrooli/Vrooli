package dispatch

import (
	"log"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/presence"
	"vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/runs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
)

// Module returns the dispatch domain's contribution to the API: the generated
// Connect-RPC DispatchService handler. The dispatch domain is the safety gate;
// this module is the single place its proto-free seams are bound to the concrete
// registry (scopes), presence (online), runs (durable run creation), audit
// (accountability), and the channel push (typed job delivery). Dispatch owns no
// table, so there is no Schema() to register.
func Module(registrySvc registry.Service, runsSvc runs.Service, auditSink audit.Sink, hub *presence.Hub, scheduler *queue.Scheduler, logger *log.Logger) module.Module {
	svc := dispatch.NewService(
		nodeReaderAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies dispatch.Presence via IsOnline + Dispatchable
		runControllerAdapter{svc: runsSvc},
		auditSinkAdapter{sink: auditSink},
		jobPusherAdapter{scheduler: scheduler}, // bounded-concurrency scheduler on the push path
	)
	path, handler := dispatchconnect.NewDispatchServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "dispatch",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
