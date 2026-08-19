package treasuryadmin

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
)

func Module(db *database.RoutedDB, authorizer operatorauth.Authorizer, clock schedule.Clock, relay approval.Relay, rails *rail.Registry, credentials instrument.CredentialResolver) module.Module {
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorization.NewSQLiteRepository(db), relay, clock)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, nil)
	instruments := instrument.NewService(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clock)
	path, handler := authorizationconnect.NewTreasuryAdminHandler(NewConnectHandler(authorizer, approvals, instruments))
	return module.Module{Name: "treasuryadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return rail.Schema() + "\n" + approval.Schema() + "\n" + instrument.Schema() }
