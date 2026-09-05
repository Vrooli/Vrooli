package graph

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	intgraph "typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/module"
	intrewrite "typescript-code-graph/internal/rewrite"

	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// Module returns the graph domain's contribution to the API: the
// Connect-mounted TypeScriptCodeGraphService plus the static
// EndpointDescriptors the codegen reads.
//
// All three RPCs (Extract, RewritePlan, RewriteApply) share one
// Connect mount because the proto declaration is one service. Extract
// flows to graph.Service; the two rewrite RPCs flow to
// rewrite.Service via the handlers/rewrite delegation helpers.
func Module(graphSvc *intgraph.Service, rewriteSvc *intrewrite.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := graph_v1connect.NewTypeScriptCodeGraphServiceHandler(NewConnectHandler(Deps{
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
