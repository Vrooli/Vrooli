package apply

import (
	"architecture-cartographer/internal/apply"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply/apply_v1connect"
)

// Module returns the apply domain's contribution to the API router.
func Module(svc apply.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := apply_v1connect.NewApplyServiceHandler(h)
	return module.Module{
		Name: "apply",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the apply domain's SQL contribution.
func Schema() string { return apply.Schema() }
