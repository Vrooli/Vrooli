// Package content is the content domain's API contribution: the generated
// ContentService Connect handler plus its endpoint descriptors and schema
// re-export. Business logic lives in internal/content.
package content

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalcontent "landing-page-react-vite-api/internal/content"
)

// Module returns the content domain's contribution: the ContentService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internalcontent.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewContentServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "content",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the content domain's SQL for the modules registry.
func Schema() string { return internalcontent.Schema() }
