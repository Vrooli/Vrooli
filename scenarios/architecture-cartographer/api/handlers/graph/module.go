package graph

import (
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

// Module returns the graph domain's contribution to the API router.
func Module(svc graph.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := graph_v1connect.NewGraphServiceHandler(h)
	return module.Module{
		Name: "graph",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the graph domain's SQL contribution.
func Schema() string { return graph.Schema() }
