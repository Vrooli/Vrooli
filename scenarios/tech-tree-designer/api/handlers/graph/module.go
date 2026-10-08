package graph

import (
	"tech-tree-designer/internal/module"

	"github.com/gorilla/mux"

	graphdomain "tech-tree-designer/internal/graph"

	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph/graph_v1connect"
)

func Module(service *graphdomain.Service) module.Module {
	h := NewHandler(service)
	return module.Module{
		Name: "graph",
		Mount: func(r *mux.Router) {
			path, handler := graphconnect.NewGraphServiceHandler(h)
			r.PathPrefix(path).Handler(handler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
