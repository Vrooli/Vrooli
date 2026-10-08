package conflicts

import (
	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts/conflicts_v1connect"
)

// Module returns the conflicts domain's contribution to the API router.
func Module(d Deps) module.Module {
	h := NewHandler(d)
	pattern, connectHandler := conflicts_v1connect.NewConflictsServiceHandler(h)
	return module.Module{
		Name: "conflicts",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the conflicts domain's SQL contribution.
func Schema() string { return conflicts.Schema() }
