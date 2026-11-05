package ledgeradmin

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/ledger/ledger_v1connect"
	"treasury/internal/ledger"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
)

func Module(db *database.RoutedDB, authorizer operatorauth.Authorizer) module.Module {
	path, handler := ledgerconnect.NewLedgerServiceHandler(NewHandler(authorizer, ledger.NewSQLiteRepository(db)))
	return module.Module{Name: "ledgeradmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}
