package accounts

import (
	"persona/internal/accounts"
	"persona/internal/handoffs"
	"persona/internal/journal"
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/accounts/accounts_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	p := personas.NewService(personas.NewSQLiteRepository(db, clock))
	h := handoffs.NewService(handoffs.NewSQLiteRepository(db, clock), p, journal.NewService(journal.NewSQLiteRepository(db, clock)), clock)
	j := journal.NewService(journal.NewSQLiteRepository(db, clock))
	return ModuleWithService(accounts.NewService(accounts.NewSQLiteRepository(db, clock), p, h, j, clock))
}
func ModuleWithService(service accounts.Service) module.Module {
	path, handler := accountsconnect.NewAccountsServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "accounts", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return accounts.Schema() }
