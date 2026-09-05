// Package bundles is the bundle-admin domain's API contribution: the generated
// BundleAdminService Connect handler (list catalog + update price display
// metadata). Business logic lives in internal/plan.
package bundles

import (
	"landing-page-react-vite-api/internal/module"
	"landing-page-react-vite-api/internal/plan"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Module returns the bundle-admin domain's contribution: the BundleAdminService
// Connect-RPC handler mounted on the shared router.
func Module(svc *plan.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewBundleAdminServiceHandler(NewConnectHandler(Deps{Plan: svc, Logger: logger}))
	return module.Module{
		Name: "bundles",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
