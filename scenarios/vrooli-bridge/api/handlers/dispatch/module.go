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

// NewService constructs the dispatch application service with its proto-free
// seams bound to the concrete registry (scopes), presence (online), runs
// (durable run creation), audit (accountability), and the channel push (typed
// job delivery). It is built once in main.go so the SAME instance backs both the
// dispatch handler (the operator DispatchJob verb) and the gate domain's runner
// adapter (which dispatches a validation run per target OS) — every gate run
// flows through the same allowlist + scope + audit gate as any other job, and no
// second dispatch policy can drift from it.
func NewService(registrySvc registry.Service, runsSvc runs.Service, auditSink audit.Sink, hub *presence.Hub, scheduler *queue.Scheduler, grants ...dispatch.CredentialGrantReader) dispatch.Service {
	options := make([]dispatch.Option, 0, 1)
	if len(grants) > 0 && grants[0] != nil {
		options = append(options, dispatch.WithCredentialGrantReader(grants[0]))
	}
	return dispatch.NewService(
		nodeReaderAdapter{svc: registrySvc},
		hub, // *presence.Hub satisfies dispatch.Presence via IsOnline + Dispatchable
		runControllerAdapter{svc: runsSvc},
		auditSinkAdapter{sink: auditSink},
		jobPusherAdapter{scheduler: scheduler}, // bounded-concurrency scheduler on the push path
		options...,
	)
}

// Module returns the dispatch domain's contribution to the API: the generated
// Connect-RPC DispatchService handler. The dispatch domain is the safety gate;
// the service is constructed by NewService (shared with gate). Dispatch owns no
// table, so there is no Schema() to register.
func Module(svc dispatch.Service, logger *log.Logger) module.Module {
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
