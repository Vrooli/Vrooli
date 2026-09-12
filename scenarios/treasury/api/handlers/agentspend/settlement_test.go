package agentspend

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/authorization"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/ledger"
	"treasury/internal/mandate"
	"treasury/internal/rail"
	"treasury/internal/settlement"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	authorizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization/authorization_v1connect"
	settlementv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/settlement"
)

// [REQ:TRS-P0-011] The generated AgentSpend client executes the server-owned
// rail result exactly once and rejects the deprecated caller-asserted outcome.
func TestReportOutcomeUsesDurableServerOwnedSettlement(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, handle,
		database.SchemaProviderFunc(book.Schema), database.SchemaProviderFunc(budget.Schema), database.SchemaProviderFunc(mandate.Schema),
		database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(evidence.Schema), database.SchemaProviderFunc(instrument.Schema), database.SchemaProviderFunc(settlement.Schema), database.SchemaProviderFunc(ledger.Schema),
	))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	clock := schedule.NewFake(now)
	_, err := book.NewService(book.NewSQLiteRepository(handle), clock).Create(ctx, book.CreateInput{ID: "book-1", Name: "Operator", BeneficiaryIdentity: "operator:1"})
	require.NoError(t, err)
	_, err = budget.NewService(budget.NewSQLiteRepository(handle), clock).Create(ctx, budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 5_000, PeriodicCapMinor: 5_000, PerTransactionCapMinor: 500, Period: time.Hour, AllowedCounterparties: []string{"api.example"}})
	require.NoError(t, err)
	signer, err := mandate.NewHMACSigner([]byte("handler-settlement-signing-key"))
	require.NoError(t, err)
	mandates := mandate.NewService(mandate.NewSQLiteRepository(handle), clock, signer)
	_, err = mandates.Issue(ctx, mandate.IssueInput{ID: "mandate-1", IdempotencyKey: "mandate-key", BookID: "book-1", BudgetID: "budget-1", Authorizer: "operator:1", CapMinor: 500, Currency: "USD", AllowedCounterparties: []string{"api.example"}, ExpiresAt: now.Add(time.Hour)})
	require.NoError(t, err)
	adapter := &handlerRail{}
	registry, err := rail.NewRegistry(adapter)
	require.NoError(t, err)
	instruments := instrument.NewService(instrument.NewSQLiteRepository(handle), mandates, registry, handlerCredentialResolver{}, clock)
	_, err = instruments.Register(ctx, instrument.RegisterInput{ID: "instrument-1", MandateID: "mandate-1", Rail: adapter.Name(), CredentialReference: "treasury/handler-test", Counterparty: "api.example"})
	require.NoError(t, err)
	authorizations := authorization.NewSQLiteRepository(handle)
	_, err = authorizations.Create(ctx, authorization.Record{ID: "auth-1", IdempotencyKey: "auth-key", BookID: "book-1", MandateID: "mandate-1", BudgetID: "budget-1", RequestingAgent: "agent:1", AmountMinor: 125, Currency: "USD", Counterparty: "api.example", Verdict: authorization.VerdictApproved, HoldMinor: 125, CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute)})
	require.NoError(t, err)
	verifier := handlerVerifier{claims: identity.Claims{Subject: "agent:1"}}
	authorizationService := authorization.NewService(authorizations, verifier, mandates, budget.NewService(budget.NewSQLiteRepository(handle), clock), evidence.NewRecorder(evidence.NewSQLiteRecorder(handle)), clock)
	settlementService := settlement.NewService(settlement.NewSQLiteRepository(handle), authorizations, instruments, registry, verifier, clock)
	_, transport := authorizationconnect.NewAgentSpendHandler(NewConnectHandler(authorizationService, settlementService))
	server := httptest.NewServer(transport)
	t.Cleanup(server.Close)
	client := authorizationconnect.NewAgentSpendClient(server.Client(), server.URL)

	forged := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-1", SettlementId: "settlement-1", InstrumentId: "instrument-1", IdempotencyKey: "settle-key", Outcome: "settled"})
	forged.Header().Set(identity.HeaderAgentIdentityToken, "opaque")
	_, err = client.ReportOutcome(ctx, forged)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	require.Zero(t, adapter.calls.Load())

	request := connect.NewRequest(&authorizationv1.ReportOutcomeRequest{AuthorizationId: "auth-1", SettlementId: "settlement-1", InstrumentId: "instrument-1", IdempotencyKey: "settle-key"})
	request.Header().Set(identity.HeaderAgentIdentityToken, "opaque")
	first, err := client.ReportOutcome(ctx, request)
	require.NoError(t, err)
	require.Equal(t, settlementv1.ChargeOutcome_CHARGE_OUTCOME_SETTLED, first.Msg.GetSettlement().GetOutcome())
	require.Equal(t, authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_SETTLED, first.Msg.GetAuthorization().GetVerdict())
	second, err := client.ReportOutcome(ctx, request)
	require.NoError(t, err)
	require.Equal(t, first.Msg.GetSettlement().GetExternalId(), second.Msg.GetSettlement().GetExternalId())
	require.EqualValues(t, 1, adapter.calls.Load())
}

type handlerCredentialResolver struct{}

func (handlerCredentialResolver) Resolve(context.Context, string, string) (string, error) {
	return "memory-only-handler-secret", nil
}

type handlerRail struct{ calls atomic.Int64 }

func (*handlerRail) Name() string { return "handler-rail" }

func (r *handlerRail) Settle(context.Context, rail.SettleCommand) (rail.Result, error) {
	r.calls.Add(1)
	return rail.Result{Outcome: rail.OutcomeSettled, ExternalID: "external-1", ReceiptReference: "receipt-1", Basis: "processor_confirmation", Detail: "confirmed", OccurredAt: time.Date(2026, 8, 18, 12, 0, 1, 0, time.UTC)}, nil
}

func (*handlerRail) QueryOutcome(context.Context, rail.Query) (rail.Result, error) {
	return rail.Result{Outcome: rail.OutcomeUnknown}, nil
}

var _ rail.Adapter = (*handlerRail)(nil)
