package queue

import (
	"log"

	"vrooli-bridge/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	queueconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue/queue_v1connect"
)

// Module returns the queue domain's contribution to the API: the generated
// Connect-RPC QueueService handler (the read-only control-plane view over the
// per-node job scheduler, OT-P1-004). The scheduler itself is constructed in
// main.go (it is shared with dispatch's Submit and the runs terminal hook); this
// module only exposes its live snapshot. The queue owns no tables (the durable
// source of truth for a job is its Run), so there is no Schema().
func Module(scheduler Snapshotter, logger *log.Logger) module.Module {
	path, handler := queueconnect.NewQueueServiceHandler(NewConnectHandler(Deps{
		Scheduler: scheduler,
		Logger:    logger,
	}))
	return module.Module{
		Name: "queue",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
