// Package metrics is the metrics domain's API contribution: the generated
// MetricsService Connect handler plus its endpoint descriptors and schema
// re-export. Business logic lives in internal/metrics.
package metrics

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalmetrics "landing-page-react-vite-api/internal/metrics"
)

// Module returns the metrics domain's contribution: the MetricsService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internalmetrics.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewMetricsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "metrics",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the metrics domain's SQL for the modules registry.
func Schema() string { return internalmetrics.Schema() }
