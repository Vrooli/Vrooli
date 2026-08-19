package treasuryadmin

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	"treasury/internal/settlement"
)

func Module(db *database.RoutedDB, authorizer operatorauth.Authorizer, clock schedule.Clock, relay approval.Relay, rails *rail.Registry, cardIssuers *card.Registry, credentials instrument.CredentialResolver, signer mandate.Signer) module.Module {
	authorizations := authorization.NewSQLiteRepository(db)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorizations, relay, clock, evidenceRecorder)
	books := book.NewService(book.NewSQLiteRepository(db), clock)
	budgets := budget.NewService(budget.NewSQLiteRepository(db), clock, authorizations)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, signer)
	instruments := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clock, cardIssuers)
	settlements := settlement.NewService(settlement.NewSQLiteRepository(db), authorizations, instruments, rails, nil, clock, budgets)
	path, handler := authorizationconnect.NewTreasuryAdminHandler(NewConnectHandler(authorizer, books, budgets, mandates, approvals, instruments, settlements, authorizations))
	return module.Module{Name: "treasuryadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string {
	return rail.Schema() + "\n" + approval.Schema() + "\n" + instrument.Schema()
}
