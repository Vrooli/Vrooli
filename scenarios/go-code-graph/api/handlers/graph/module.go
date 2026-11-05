package graph

import (
	"log"

	"go-code-graph/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	intgraph "go-code-graph/internal/graph"
	intrewrite "go-code-graph/internal/rewrite"

	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// Module returns the graph domain's contribution to the API: the
// Connect-mounted GoCodeGraphService plus the static EndpointDescriptors
// the codegen reads.
//
// All three GoCodeGraphService RPCs (Extract, RewritePlan, RewriteApply)
// share one Connect mount because the proto declaration is one service.
// The connectHandler holds references to BOTH the graph and rewrite
// domain services and routes each RPC to the right one. handlers/rewrite
// contributes the rewrite EndpointDescriptors separately (so the
// registry walk maps cleanly to per-domain ownership) but does not own
// a Connect mount of its own.
func Module(graphSvc *intgraph.Service, rewriteSvc *intrewrite.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := graph_v1connect.NewGoCodeGraphServiceHandler(NewConnectHandler(Deps{
		GraphService:   graphSvc,
		RewriteService: rewriteSvc,
		Logger:         logger,
	}))
	return module.Module{
		Name: "graph",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — graph is stateless; the registry collects this
// re-export so a future stateful turn (e.g. caching extracted graphs)
// is a one-line schema swap rather than a structural change.
func Schema() string { return "" }
