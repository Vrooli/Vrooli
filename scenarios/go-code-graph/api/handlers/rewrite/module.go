package rewrite

import (
	"log"

	"github.com/gorilla/mux"

	"go-code-graph/internal/module"
	intrewrite "go-code-graph/internal/rewrite"
)

// Module returns the rewrite domain's static contribution to the API:
// the EndpointDescriptors for RewritePlan / RewriteApply, plus a
// no-op Mount.
//
// The actual Connect handler that satisfies the GoCodeGraphService
// interface is mounted by handlers/graph because the proto declares
// one service with all three RPCs. The graph Module constructor takes
// both *intgraph.Service AND *intrewrite.Service, and routes each RPC
// to the right domain service. See handlers/graph/module.go for the
// wiring.
//
// We keep this Module() constructor so main.go's registration shape
// (`server.New(..., graphH.Module(...), rewriteH.Module(...))`) is
// uniform across domains. The `svc` parameter is unused at the
// handler layer — it's accepted so callers don't have to remember
// the asymmetry — but main.go must still pass the same *Service it
// hands to graphH.Module so they share state.
func Module(_ *intrewrite.Service, _ *log.Logger) module.Module {
	return module.Module{
		Name:      "rewrite",
		Mount:     func(_ *mux.Router) {},
		Endpoints: Endpoints,
	}
}
