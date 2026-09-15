package runs

import (
	"log"

	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	internalruns "vrooli-bridge/internal/runs"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

// Module returns the runs domain's contribution to the API: the generated
// Connect-RPC RunsService handler. The runs.Service is constructed in main.go
// (shared with the dispatch handler, which creates runs) and passed in, so the
// in-memory block-once waiter / live-event-subscriber coordination is one
// coherent instance across every call site.
func Module(svc internalruns.Service, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
	path, handler := runsconnect.NewRunsServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Verifier: verifier,
		Logger:   logger,
	}))
	return module.Module{
		Name: "runs",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the runs domain's SQL contribution so the modules registry
// collects endpoint descriptors and schema from one handler package.
func Schema() string { return internalruns.Schema() }
