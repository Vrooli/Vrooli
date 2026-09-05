// Package branding is the branding domain's API contribution: the generated
// BrandingService Connect handler plus its endpoint descriptors and schema
// re-export. Business logic lives in internal/branding.
package branding

import (
	"errors"
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalbranding "landing-page-react-vite-api/internal/branding"
)

var errFieldRequired = errors.New("field name required")

// Module returns the branding domain's contribution: the BrandingService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internalbranding.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewBrandingServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "branding",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the branding domain's SQL for the modules registry.
func Schema() string { return internalbranding.Schema() }
