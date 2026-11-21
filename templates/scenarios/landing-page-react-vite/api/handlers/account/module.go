// Package account is the account domain's API contribution: the generated
// AccountService Connect handler (caller-scoped subscription, credits, and
// entitlements). Caller identity is read from the X-User-Email header. Business
// logic lives in internal/account.
package account

import (
	"landing-page-react-vite-api/internal/account"
	"landing-page-react-vite-api/internal/module"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Module returns the account domain's contribution: the AccountService
// Connect-RPC handler mounted on the shared router.
func Module(svc *account.Service, logger *log.Logger) module.Module {
	path, handler := landingconnect.NewAccountServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "account",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
