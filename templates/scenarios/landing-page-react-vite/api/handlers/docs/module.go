// Package docs is the docs domain's API contribution: the generated DocsService
// Connect handler plus its endpoint descriptors. Filesystem logic lives in
// internal/docs; the domain owns no database tables.
package docs

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internaldocs "landing-page-react-vite-api/internal/docs"
)

// Module returns the docs domain's contribution: the DocsService Connect-RPC
// handler mounted on the shared router.
func Module(svc *internaldocs.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewDocsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "docs",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — docs is filesystem-backed and owns no tables.
func Schema() string { return "" }
