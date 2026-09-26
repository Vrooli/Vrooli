// Package reset is the admin demo-reset domain's API contribution: the
// generated AdminResetService Connect handler, gated by the admin
// session-cookie interceptor (destructive, admin-only) in addition to the
// ENABLE_ADMIN_RESET env gate. Business logic lives in internal/adminreset.
package reset

import (
	"landing-page-react-vite-api/internal/adminreset"
	"landing-page-react-vite-api/internal/module"
	"log"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internaladmin "landing-page-react-vite-api/internal/admin"
)

// Module returns the reset domain's contribution: the AdminResetService Connect
// handler, protected by the admin session-cookie interceptor.
func Module(svc *adminreset.Service, admin *internaladmin.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewAdminResetServiceHandler(
		NewConnectHandler(Deps{Service: svc, Logger: logger}),
		connect.WithInterceptors(admin.Interceptor()),
	)
	return module.Module{
		Name: "reset",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
