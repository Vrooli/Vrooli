package agentspend

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/evidence"
	"treasury/internal/identity"
	"treasury/internal/mandate"
	mandateflow "treasury/internal/mandate/flow"

	authorizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/authorization"
)

type handlerVerifier struct {
	claims identity.Claims
	err    error
}

type capturingVerifier struct{ token string }

func (v *capturingVerifier) Verify(_ context.Context, token string) (identity.Claims, error) {
	v.token = token
	return identity.Claims{Subject: "agent:1"}, nil
}

type mandateLister struct{ values []mandate.Mandate }

func (l mandateLister) List(context.Context) ([]mandate.Mandate, error) { return l.values, nil }

func TestIdentityTokenHeadersSupportGeneratedCLIAndExplicitAgentRealm(t *testing.T) {
	require.Equal(t, "cli-agent-token", identityTokenFromHeaders(http.Header{"Authorization": []string{"Bearer cli-agent-token"}}))
	require.Equal(t, "explicit-agent-token", identityTokenFromHeaders(http.Header{identity.HeaderAgentIdentityToken: []string{"explicit-agent-token"}, "Authorization": []string{"Bearer ignored"}}))
	require.Empty(t, identityTokenFromHeaders(http.Header{"Authorization": []string{"Basic nope"}}))
}

func TestListMandatesRequiresVerifiedAgentAndReturnsTypedGrants(t *testing.T) {
	verifier := &capturingVerifier{}
	now := time.Now().UTC()
	handler := NewConnectHandler(mandateLister{values: []mandate.Mandate{{ID: "mandate-1", Authorizer: "local-operator", ExpiresAt: now.Add(time.Hour), IssuedAt: now, Status: mandateflow.MandateLive}}}, verifier).(*connectHandler)
	request := connect.NewRequest(&authorizationv1.ListMandatesRequest{})
	request.Header().Set("Authorization", "Bearer generated-cli-agent-token")
	response, err := handler.ListMandates(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, "generated-cli-agent-token", verifier.token)
	require.Len(t, response.Msg.GetMandates(), 1)
	require.Equal(t, "mandate-1", response.Msg.GetMandates()[0].GetId())
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

type authorizationReadRepository struct {
	authorization.Repository
	record  authorization.Record
	lookups int
}

func (r *authorizationReadRepository) Get(context.Context, string) (authorization.Record, error) {
	r.lookups++
	return r.record, nil
}

type authorizationReadVerifier struct {
	claims identity.Claims
	err    error
	calls  int
}

func (v *authorizationReadVerifier) Verify(_ context.Context, token string) (identity.Claims, error) {
	v.calls++
	if token == "" {
		return identity.Claims{}, errors.New("identity token is absent")
	}
	return v.claims, v.err
}

// AgentSpend may read only attempts belonging to the live verified owner
// subject. Verification must precede even the attempt lookup.
func TestGetAuthorizationRequiresLiveOwnerIdentity(t *testing.T) {
	for _, test := range []struct {
		name        string
		token       string
		subject     string
		verifyErr   error
		noVerifier  bool
		wantCode    connect.Code
		wantLookups int
	}{
		{name: "missing identity", wantCode: connect.CodeUnauthenticated},
		{name: "rejected identity", token: "public-fake-token", verifyErr: errors.New("revoked"), wantCode: connect.CodeUnauthenticated},
		{name: "different owner", token: "public-fake-token", subject: "another-owner-account", wantCode: connect.CodePermissionDenied, wantLookups: 1},
		{name: "same owner", token: "public-fake-token", subject: "owner-account", wantLookups: 1},
		{name: "verifier unavailable", token: "public-fake-token", noVerifier: true, wantCode: connect.CodeUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &authorizationReadRepository{record: authorization.Record{ID: "public-attempt", RequestingAgent: "owner-account", Counterparty: "private-counterparty.example", AmountMinor: 10}}
			service := authorization.NewService(repository, nil, nil, nil, nil, nil)
			verifier := &authorizationReadVerifier{claims: identity.Claims{Subject: test.subject}, err: test.verifyErr}
			var handler *connectHandler
			if test.noVerifier {
				handler = NewConnectHandler(service).(*connectHandler)
			} else {
				handler = NewConnectHandler(service, verifier).(*connectHandler)
			}
			request := connect.NewRequest(&authorizationv1.GetAuthorizationRequest{Id: "public-attempt"})
			request.Header().Set(identity.HeaderAgentIdentityToken, test.token)
			response, err := handler.GetAuthorization(context.Background(), request)
			if test.wantCode == 0 {
				require.NoError(t, err)
				require.Equal(t, "public-attempt", response.Msg.GetAuthorization().GetId())
				require.Equal(t, "owner-account", response.Msg.GetAuthorization().GetRequestingAgent())
			} else {
				if connect.CodeOf(err) != test.wantCode {
					t.Errorf("got error %v (%v), want %v", err, connect.CodeOf(err), test.wantCode)
				}
				if response != nil {
					t.Error("refused attempt read returned authorization data")
				}
			}
			require.Equal(t, test.wantLookups, repository.lookups, "identity refusal must precede lookup")
			if !test.noVerifier {
				require.Equal(t, 1, verifier.calls)
			}
		})
	}
}

func TestGetAuthorizationRechecksRevokedIdentity(t *testing.T) {
	repository := &authorizationReadRepository{record: authorization.Record{ID: "public-attempt", RequestingAgent: "owner-account"}}
	service := authorization.NewService(repository, nil, nil, nil, nil, nil)
	verifier := &authorizationReadVerifier{claims: identity.Claims{Subject: "owner-account"}}
	handler := NewConnectHandler(service, verifier).(*connectHandler)
	request := connect.NewRequest(&authorizationv1.GetAuthorizationRequest{Id: "public-attempt"})
	request.Header().Set(identity.HeaderAgentIdentityToken, "public-fake-token")
	_, err := handler.GetAuthorization(context.Background(), request)
	require.NoError(t, err)
	verifier.err = errors.New("revoked after first read")
	response, err := handler.GetAuthorization(context.Background(), request)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Nil(t, response)
	require.Equal(t, 2, verifier.calls)
	require.Equal(t, 1, repository.lookups)
}
