package agentspend

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"
)

type handlerVerifier struct {
	claims identity.Claims
	err    error
}

func (v handlerVerifier) Verify(context.Context, string) (identity.Claims, error) {
	return v.claims, v.err
}

type handlerMandates struct{ value mandate.Mandate }

func (m handlerMandates) RequireLive(context.Context, string) (mandate.Mandate, error) {
	return m.value, nil
}

type handlerBudgets struct{ value budget.Budget }

func (b handlerBudgets) Get(context.Context, string) (budget.Budget, error) { return b.value, nil }

// [REQ:TRS-P0-005] The agent-facing transport refuses a missing/unverifiable identity and persists only refusal evidence.
func TestProposeChargeFailsClosedWhenIdentityAuthorityIsUnavailable(t *testing.T) {
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(evidence.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	service := authorization.NewService(authorization.NewSQLiteRepository(handle), handlerVerifier{err: errors.New("connection refused")}, handlerMandates{}, handlerBudgets{}, evidence.NewRecorder(evidence.NewSQLiteRecorder(handle)), schedule.NewFake(now))
	handler := NewConnectHandler(service).(*connectHandler)
	request := connect.NewRequest(&authorizationv1.ProposeChargeRequest{Id: "auth-1", IdempotencyKey: "key-1", MandateId: "mandate-1", AmountMinor: 10, Currency: "USD", Counterparty: "api.example"})

	response, err := handler.ProposeCharge(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED, response.Msg.GetAuthorization().GetVerdict())
	require.Equal(t, "identity", response.Msg.GetAuthorization().GetViolatedConstraint())
	var evidenceCount, authorizationCount int
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM evidence_records`).Scan(&evidenceCount))
	require.NoError(t, handle.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM authorizations`).Scan(&authorizationCount))
	require.Equal(t, 1, evidenceCount)
	require.Zero(t, authorizationCount)
}

// [REQ:TRS-P0-002] The wire request has no verdict override and an out-of-cap charge is recomputed and refused.
func TestProposeChargeCannotSupplyItsOwnVerdict(t *testing.T) {
	handle := db.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), handle, database.SchemaProviderFunc(authorization.Schema), database.SchemaProviderFunc(evidence.Schema)))
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	grant := mandate.Mandate{ID: "mandate-1", BookID: "book-1", BudgetID: "budget-1", CapMinor: 100, Currency: "USD", AllowedCounterparties: []string{"api.example"}, ExpiresAt: now.Add(time.Hour), Status: mandateflow.MandateLive}
	policy := budget.Budget{ID: "budget-1", BookID: "book-1", Currency: "USD", TotalCapMinor: 100, PeriodicCapMinor: 100, PerTransactionCapMinor: 50, Period: time.Hour, AllowedCounterparties: []string{"api.example"}}
	service := authorization.NewService(authorization.NewSQLiteRepository(handle), handlerVerifier{claims: identity.Claims{Subject: "operator:1"}}, handlerMandates{grant}, handlerBudgets{policy}, evidence.NewRecorder(evidence.NewSQLiteRecorder(handle)), schedule.NewFake(now))
	handler := NewConnectHandler(service).(*connectHandler)
	request := connect.NewRequest(&authorizationv1.ProposeChargeRequest{Id: "auth-1", IdempotencyKey: "key-1", MandateId: "mandate-1", AmountMinor: 75, Currency: "USD", Counterparty: "api.example"})
	request.Header().Set(identity.HeaderAgentIdentityToken, "opaque")

	response, err := handler.ProposeCharge(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, authorizationv1.AuthorizationVerdict_AUTHORIZATION_VERDICT_REFUSED, response.Msg.GetAuthorization().GetVerdict())
	require.Equal(t, "per_transaction_cap", response.Msg.GetAuthorization().GetViolatedConstraint())
	require.NotEmpty(t, response.Msg.GetAuthorization().GetRemediation())
}
