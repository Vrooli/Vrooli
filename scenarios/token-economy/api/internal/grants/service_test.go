package grants_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/grants"
	grantmocks "token-economy/internal/grants/mocks"
)

type tokenTypeReader struct{ value grants.TokenTypeState }

func (r tokenTypeReader) GetTokenType(context.Context, string) (grants.TokenTypeState, error) {
	return r.value, nil
}

// [REQ:TKE-P0-002] Grant creation is the service's only balance-increasing
// operation and it always emits one exactly matching journal credit.
func TestCreateRoutesBalanceAdmissionThroughTypedGrant(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	var createdGrant grants.Grant
	var createdEvent grants.Credit
	repository := &grantmocks.FakeRepository{
		CreateFunc: func(_ context.Context, grant grants.Grant, event grants.Credit) (grants.Grant, error) {
			createdGrant, createdEvent = grant, event
			return grant, nil
		},
	}
	service := grants.NewService(repository, tokenTypeReader{value: grants.TokenTypeState{ID: "chores"}}, nil, schedule.NewFake(now))
	created, err := service.Create(context.Background(), grants.CreateInput{
		TokenTypeID: "chores", GrantSourceID: "weekly-chores", Authorizer: "parent:alex",
		HolderID: "child:sam", AmountMinor: 8, AllowedCatalogScopes: []string{"screen-time"},
		ExpiresAt: now.Add(24 * time.Hour), IdempotencyKey: "grant-week-1",
	})
	require.NoError(t, err)
	require.Equal(t, createdGrant, created)
	require.Equal(t, created.TokenTypeID, createdEvent.TokenTypeID)
	require.Equal(t, created.HolderID, createdEvent.HolderID)
	require.Equal(t, created.AmountMinor, createdEvent.Amount)
	require.Equal(t, "grant:"+created.ID, createdEvent.CauseReference)

	serviceType := reflect.TypeOf((*grants.Service)(nil)).Elem()
	methodNames := make([]string, 0, serviceType.NumMethod())
	for i := 0; i < serviceType.NumMethod(); i++ {
		methodNames = append(methodNames, serviceType.Method(i).Name)
		name := strings.ToLower(serviceType.Method(i).Name)
		require.NotContains(t, name, "append")
		require.NotContains(t, name, "credit")
		require.NotContains(t, name, "mint")
	}
	require.ElementsMatch(t, []string{"Create", "EvaluateRedemption", "Get", "List", "Revoke"}, methodNames)
}

// [REQ:TKE-P0-003] A caller assertion cannot turn a server-side refusal into
// an authorization decision.
func TestCallerAuthorizationClaimDoesNotChangeDecision(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	storedGrant := grants.Grant{
		ID: "grant-1", ExpiresAt: now.Add(time.Hour),
		Rules: []grants.GrantRule{{ID: "scope-rule", Condition: grants.RuleConditionCatalogScopeAllowed, Operands: []string{"books"}}},
	}
	repository := &grantmocks.FakeRepository{
		GetFunc: func(context.Context, string) (grants.Grant, error) { return storedGrant, nil },
	}
	service := grants.NewService(repository, tokenTypeReader{}, grants.NewRuleEvaluator(), schedule.NewFake(now))
	base := grants.EvaluationRequest{CatalogScope: "games", AvailableBalance: 10, RequestedAmount: 1, Now: now}
	withoutClaim, err := service.EvaluateRedemption(context.Background(), "grant-1", base)
	require.NoError(t, err)
	base.CallerClaimsAuthorized = true
	withClaim, err := service.EvaluateRedemption(context.Background(), "grant-1", base)
	require.NoError(t, err)
	require.Equal(t, withoutClaim, withClaim)
	require.False(t, withClaim.Allowed)
	require.Contains(t, withClaim.Reason, "scope-rule")
	require.Contains(t, withClaim.Reason, string(grants.RuleConditionCatalogScopeAllowed))
}
