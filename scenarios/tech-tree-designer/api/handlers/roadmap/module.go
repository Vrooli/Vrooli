package roadmap

import (
	"tech-tree-designer/internal/module"
	roadmapdomain "tech-tree-designer/internal/roadmap"

	"github.com/gorilla/mux"

	roadmapconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/roadmap/roadmap_v1connect"
)

func Module(service *roadmapdomain.Service) module.Module {
	h := NewHandler(service)
	return module.Module{
		Name: "roadmap",
		Mount: func(r *mux.Router) {
			path, handler := roadmapconnect.NewRoadmapServiceHandler(h)
			r.PathPrefix(path).Handler(handler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return roadmapdomain.Schema() }
