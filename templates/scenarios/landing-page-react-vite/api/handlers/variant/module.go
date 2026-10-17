// Package variant is the variant domain's API contribution: the generated
// VariantService Connect handler plus its endpoint descriptors and schema
// re-export. Business logic lives in internal/variant.
package variant

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalvariant "landing-page-react-vite-api/internal/variant"
)

// Module returns the variant domain's contribution: the VariantService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internalvariant.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewVariantServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "variant",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the variant domain's SQL for the modules registry.
func Schema() string { return internalvariant.Schema() }
