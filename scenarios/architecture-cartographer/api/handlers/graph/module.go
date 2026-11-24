package graph

import (
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"github.com/vrooli/api-core/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph/graph_v1connect"
)

// Module returns the graph domain's contribution to the API router.
func Module(svc graph.Service, domainsSvc ...domains.Service) module.Module {
	h := NewHandler(svc, domainsSvc...)
	return buildModule(h)
}

func ModuleWithLocator(svc graph.Service, domainsSvc domains.Service, locator domains.ScenarioLocator) module.Module {
	return buildModule(NewHandlerWithLocator(svc, domainsSvc, locator))
}

func buildModule(h *Handler) module.Module {
	pattern, connectHandler := graph_v1connect.NewGraphServiceHandler(h)
	return module.Module{
		Name: "graph",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
			r.Handle("/api/v1/slices/search", h.SearchSlicesHTTP()).Methods("POST")
			r.Handle("/api/v1/doc-code-drift/search", h.SearchDocCodeDriftHTTP()).Methods("POST")
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the graph domain's SQL contribution.
func Schema() string { return graph.Schema() }
