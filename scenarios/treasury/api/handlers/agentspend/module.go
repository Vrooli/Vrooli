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
	"treasury/internal/walletops"
)

type HandlerOverrides struct {
	WalletOps      *walletops.Service
	Authorization  *authorization.Service
	Settlements    *settlement.Service
	ResearchBudget ResearchBudgetPublisher
}

func Module(db *database.RoutedDB, verifier identity.Verifier, clk schedule.Clock, relay approval.Relay, rails *rail.Registry, cardIssuers *card.Registry, credentials instrument.CredentialResolver, overrides HandlerOverrides) module.Module {
	authorizations := authorization.NewSQLiteRepository(db)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clk, nil)
	budgets := budget.NewService(budget.NewSQLiteRepository(db), clk, authorizations)
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorization.NewApprovalController(authorizations, mandates, budgets, clk), relay, clk, evidenceRecorder)
	authorizationService := authorization.NewService(authorizations, verifier, mandates, budgets, evidenceRecorder, clk, approvals)
	instruments := instrument.NewServiceWithCardIssuers(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clk, cardIssuers)
	settlementService := settlement.NewService(settlement.NewSQLiteRepository(db), authorizations, instruments, rails, verifier, clk, budgets)
	settlementService.SetOperatorMandates(mandates)
	if overrides.Authorization != nil {
		authorizationService = overrides.Authorization
	}
	if overrides.Settlements != nil {
		settlementService = overrides.Settlements
	}
	path, handler := authorizationconnect.NewAgentSpendHandler(NewConnectHandler(AgentSpendDependencies{
		Authorization: authorizationService, Settlements: settlementService,
		Mandates: mandates, MandateReader: mandates, Verifier: verifier,
		Headroom: budgets, WalletOps: overrides.WalletOps, Clock: clk,
		ResearchBudget: overrides.ResearchBudget,
	}))
	return module.Module{Name: "agentspend", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string {
	return authorization.Schema() + "\n" + settlement.Schema() + "\n" + evidence.Schema() + "\n" + ledger.Schema()
}
