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
	"treasury/internal/mandate"
	"treasury/internal/module"
	"treasury/internal/rail"
	"treasury/internal/settlement"
)

func Module(db *database.RoutedDB, verifier identity.Verifier, clock schedule.Clock, relay approval.Relay, rails *rail.Registry, credentials instrument.CredentialResolver) module.Module {
	authorizations := authorization.NewSQLiteRepository(db)
	approvals := approval.NewService(approval.NewSQLiteRepository(db), authorizations, relay, clock)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(db), clock, nil)
	budgets := budget.NewService(budget.NewSQLiteRepository(db.Primary()), clock)
	evidenceRecorder := evidence.NewRecorder(evidence.NewSQLiteRecorder(db))
	authorizationService := authorization.NewService(authorizations, verifier, mandates, budgets, evidenceRecorder, clock, approvals)
	instruments := instrument.NewService(instrument.NewSQLiteRepository(db), mandates, rails, credentials, clock)
	settlementService := settlement.NewService(settlement.NewSQLiteRepository(db), authorizations, instruments, rails, verifier, clock)
	path, handler := authorizationconnect.NewAgentSpendHandler(NewConnectHandler(authorizationService, settlementService))
	return module.Module{Name: "agentspend", Mount: func(router *mux.Router) {
		connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

func Schema() string { return authorization.Schema() + "\n" + settlement.Schema() }
