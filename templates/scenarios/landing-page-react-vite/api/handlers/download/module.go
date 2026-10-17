// Package download is the downloads domain's API contribution: the generated
// DownloadService Connect handler (public entitlement-gated AuthorizeDownload
// plus admin catalog management). Business logic lives in internal/download.
package download

import (
	"landing-page-react-vite-api/internal/module"
	"landing-page-react-vite-api/internal/plan"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internaldownload "landing-page-react-vite-api/internal/download"
)

// Module returns the downloads domain's contribution: the DownloadService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internaldownload.Service, authorizer *internaldownload.Authorizer, planSvc *plan.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewDownloadServiceHandler(NewConnectHandler(Deps{
		Service: svc, Authorizer: authorizer, Plan: planSvc, Logger: logger,
	}))
	return module.Module{
		Name: "download",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the download domain's SQL for the modules registry.
func Schema() string { return internaldownload.Schema() }
