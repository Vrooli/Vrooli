package budget_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"treasury/handlers/budget"
	domain "treasury/internal/budget"
	"treasury/internal/operatorauth"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	budgetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget"
	budgetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/budget/budget_v1connect"
)

type policyStub struct {
	scope  domain.FreezeScope
	id     string
	frozen bool
}

func (s *policyStub) SetCaps(context.Context, domain.Budget) (domain.Budget, error) {
	return domain.Budget{}, nil
}
func (s *policyStub) SetGating(context.Context, string, bool) (domain.Budget, error) {
	return domain.Budget{}, nil
}
func (s *policyStub) SetFrozen(context.Context, string, bool) (domain.Budget, error) {
	return domain.Budget{}, nil
}
func (s *policyStub) SetScopeFrozen(_ context.Context, scope domain.FreezeScope, id string, frozen bool) (domain.FreezeControl, error) {
	s.scope, s.id, s.frozen = scope, id, frozen
	return domain.FreezeControl{Scope: scope, ScopeID: id, Frozen: frozen, UpdatedAt: time.Unix(1_800_000_000, 0)}, nil
}
func (s *policyStub) ScenarioFreezeStatus(context.Context) (domain.FreezeControl, error) {
	return domain.FreezeControl{Scope: domain.FreezeScopeScenario, ScopeID: "*", Frozen: s.frozen, UpdatedAt: time.Unix(1_800_000_000, 0)}, nil
}

func TestBudgetAdminRejectsAgentRealmBeforePolicyService(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	path, handler := budgetconnect.NewBudgetAdminServiceHandler(budget.NewConnectHandler(authorizer, nil))
	_ = path
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := budgetconnect.NewBudgetAdminServiceClient(server.Client(), server.URL)

	cases := []struct {
		name string
		call func(context.Context, http.Header) error
	}{
		{name: "SetBudgetCaps", call: func(ctx context.Context, headers http.Header) error {
			request := connect.NewRequest(&budgetv1.SetBudgetCapsRequest{})
			request.Header().Set(operatorauth.HeaderAgentToken, headers.Get(operatorauth.HeaderAgentToken))
			request.Header().Set(operatorauth.HeaderOperatorToken, headers.Get(operatorauth.HeaderOperatorToken))
			_, err := client.SetBudgetCaps(ctx, request)
			return err
		}},
		{name: "SetGating", call: func(ctx context.Context, headers http.Header) error {
			request := connect.NewRequest(&budgetv1.SetGatingRequest{})
			request.Header().Set(operatorauth.HeaderAgentToken, headers.Get(operatorauth.HeaderAgentToken))
			request.Header().Set(operatorauth.HeaderOperatorToken, headers.Get(operatorauth.HeaderOperatorToken))
			_, err := client.SetGating(ctx, request)
			return err
		}},
		{name: "FreezeBudget", call: func(ctx context.Context, headers http.Header) error {
			request := connect.NewRequest(&budgetv1.FreezeBudgetRequest{})
			request.Header().Set(operatorauth.HeaderAgentToken, headers.Get(operatorauth.HeaderAgentToken))
			request.Header().Set(operatorauth.HeaderOperatorToken, headers.Get(operatorauth.HeaderOperatorToken))
			_, err := client.FreezeBudget(ctx, request)
			return err
		}},
		{name: "UnfreezeBudget", call: func(ctx context.Context, headers http.Header) error {
			request := connect.NewRequest(&budgetv1.UnfreezeBudgetRequest{})
			request.Header().Set(operatorauth.HeaderAgentToken, headers.Get(operatorauth.HeaderAgentToken))
			request.Header().Set(operatorauth.HeaderOperatorToken, headers.Get(operatorauth.HeaderOperatorToken))
			_, err := client.UnfreezeBudget(ctx, request)
			return err
		}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			agentHeaders := make(http.Header)
			agentHeaders.Set(operatorauth.HeaderAgentToken, "agent-token")
			err := testCase.call(context.Background(), agentHeaders)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "agent authority must be refused before policy service access")

			operatorHeaders := make(http.Header)
			operatorHeaders.Set(operatorauth.HeaderOperatorToken, "operator-secret")
			err = testCase.call(context.Background(), operatorHeaders)
			require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "valid operator passes the realm boundary to the unavailable-service boundary")
		})
	}
}

func TestBookAndScenarioFreezeAreServedByBudgetOwner(t *testing.T) {
	authorizer, err := operatorauth.NewStaticToken("operator-secret")
	require.NoError(t, err)
	service := &policyStub{}
	path, handler := budgetconnect.NewBudgetAdminServiceHandler(budget.NewConnectHandler(authorizer, service))
	_ = path
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := budgetconnect.NewBudgetAdminServiceClient(server.Client(), server.URL)

	bookRequest := connect.NewRequest(&budgetv1.FreezeBookRequest{BookId: "book-1"})
	bookRequest.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	bookResult, err := client.FreezeBook(context.Background(), bookRequest)
	require.NoError(t, err)
	require.Equal(t, "book-1", bookResult.Msg.GetBookId())
	require.Equal(t, domain.FreezeScopeBook, service.scope)
	require.Equal(t, "book-1", service.id)
	require.True(t, service.frozen)

	scenarioRequest := connect.NewRequest(&budgetv1.FreezeAllRequest{})
	scenarioRequest.Header().Set(operatorauth.HeaderOperatorToken, "operator-secret")
	scenarioResult, err := client.FreezeAll(context.Background(), scenarioRequest)
	require.NoError(t, err)
	require.True(t, scenarioResult.Msg.GetFrozen())
	require.Equal(t, domain.FreezeScopeScenario, service.scope)
	require.Equal(t, "*", service.id)
	require.True(t, service.frozen)
}
