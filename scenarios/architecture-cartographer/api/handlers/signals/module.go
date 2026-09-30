package signals

import (
	"architecture-cartographer/internal/module"
	"architecture-cartographer/internal/signals"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals/signals_v1connect"
)

// Module returns the signals domain's contribution to the API router.
func Module(svc signals.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := signals_v1connect.NewSignalsServiceHandler(h)
	return module.Module{
		Name: "signals",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — signals is stateless (verdicts are not persisted
// in v0.1; analytics records placement outcomes).
func Schema() string { return "" }
