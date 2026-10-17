package analytics

import (
	"architecture-cartographer/internal/analytics"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics/analytics_v1connect"
)

// Module returns the analytics domain's contribution to the API router.
func Module(svc analytics.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := analytics_v1connect.NewAnalyticsServiceHandler(h)
	return module.Module{
		Name: "analytics",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the analytics domain's SQL contribution.
func Schema() string { return analytics.Schema() }
