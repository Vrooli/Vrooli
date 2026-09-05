package planning

import (
	"time"

	"tech-tree-designer/internal/module"
	planningdomain "tech-tree-designer/internal/planning"

	"github.com/gorilla/mux"

	planningconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/planning/planning_v1connect"
)

const timeFormat = time.RFC3339Nano

func Module(service *planningdomain.Service) module.Module {
	h := NewHandler(service)
	return module.Module{
		Name: "planning",
		Mount: func(r *mux.Router) {
			path, handler := planningconnect.NewPlanningServiceHandler(h)
			r.PathPrefix(path).Handler(handler)
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return planningdomain.Schema() }
