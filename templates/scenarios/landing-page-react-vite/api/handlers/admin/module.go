// Package admin is the admin-auth domain's API contribution: the generated
// AdminAuthService Connect handler (login/logout/session). Login mints a signed
// session cookie; Session/Logout read and clear it. The session-cookie
// interceptor that gates other admin-only services lives in internal/admin.
// Business logic lives in internal/admin.
package admin

import (
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internaladmin "landing-page-react-vite-api/internal/admin"
)

// Module returns the admin-auth domain's contribution: the AdminAuthService
// Connect-RPC handler mounted on the shared router.
func Module(svc *internaladmin.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewAdminAuthServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "admin",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the admin domain's SQL (admin_users) for the modules registry.
func Schema() string { return internaladmin.Schema() }
