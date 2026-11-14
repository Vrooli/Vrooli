// Package config is the landing-config domain's API contribution: the generated
// LandingConfigService Connect handler, which aggregates variant, content,
// pricing, downloads, header, and branding into one public payload (with a
// fail-closed fallback). Business logic lives in internal/landingconfig; it owns
// no tables.
package config

import (
	"landing-page-react-vite-api/internal/landingconfig"
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Module returns the landing-config domain's contribution: the
// LandingConfigService Connect-RPC handler mounted on the shared router.
func Module(svc *landingconfig.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewLandingConfigServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "config",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the aggregator owns no tables.
func Schema() string { return "" }
