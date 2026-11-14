package agentspend

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"

	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/rail"
	"treasury/internal/rail/card"
	"treasury/internal/settlement"
)

func Module(db *database.RoutedDB, verifier identity.Verifier, clock schedule.Clock, relay approval.Relay, rails *rail.Registry, cardIssuers *card.Registry, credentials instrument.CredentialResolver, optional ...any) module.Module {
	authorizations := authorization.NewSQLiteRepository(db)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, nil)
	budgets := budget.NewService(budget.NewSQLiteRepository(db), clock, authorizations)
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorization.NewApprovalController(authorizations, mandates, budgets, clock), relay, clock, evidenceRecorder)
	authorizationService := authorization.NewService(authorizations, verifier, mandates, budgets, evidenceRecorder, clock, approvals)
	instruments := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clock, cardIssuers)
	settlementService := settlement.NewService(settlement.NewSQLiteRepository(db), authorizations, instruments, rails, verifier, clock, budgets)
	services := []any{authorizationService, settlementService, mandates, budgets, verifier}
	services = append(services, optional...)
	path, handler := authorizationconnect.NewAgentSpendHandler(NewConnectHandler(services...))
	return module.Module{Name: "agentspend", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string {
	return authorization.Schema() + "\n" + settlement.Schema() + "\n" + evidence.Schema() + "\n" + ledger.Schema()
}
