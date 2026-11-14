package treasuryadmin

import (
	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/baseusdc"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/nativeusdc"
	"treasury/internal/operatorauth"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	"treasury/internal/receiptfunding"
	"treasury/internal/settlement"
	"treasury/internal/wallet"
	"treasury/internal/walletops"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
)

func Module(db *database.RoutedDB, authorizer operatorauth.Authorizer, clock schedule.Clock, relay approval.Relay, rails *rail.Registry, cardIssuers *card.Registry, credentials instrument.CredentialResolver, signer mandate.Signer, optionalServices ...any) module.Module {
	authorizations := authorization.NewSQLiteRepository(db)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	books := book.NewService(book.NewSQLiteRepository(db), clock)
	budgets := budget.NewService(budget.NewSQLiteRepository(db), clock, authorizations)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, signer)
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorization.NewApprovalController(authorizations, mandates, budgets, clock), relay, clock, evidenceRecorder)
	instruments := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clock, cardIssuers)
	settlements := settlement.NewService(settlement.NewSQLiteRepository(db), authorizations, instruments, rails, nil, clock, budgets)
	services := []any{books, budgets, mandates, approvals, instruments, settlements, authorizations}
	for _, service := range optionalServices {
		if service != nil {
			services = append(services, service)
		}
	}
	path, handler := authorizationconnect.NewTreasuryAdminHandler(NewConnectHandler(authorizer, services...))
	return module.Module{Name: "treasuryadmin", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string {
	return rail.Schema() + "\n" + approval.Schema() + "\n" + instrument.Schema() + "\n" + wallet.Schema() + "\n" + baseusdc.Schema() + "\n" + receiptfunding.Schema() + "\n" + nativeusdc.Schema() + "\n" + walletops.Schema()
}
