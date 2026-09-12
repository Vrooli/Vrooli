package grants_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"token-economy/internal/grants"
)

// [REQ:TKE-P0-003] Every closed rule condition is evaluated server-side and
// every refusal names the exact rule and condition that refused it.
func TestRuleEvaluatorRefusalTable(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		grant   grants.Grant
		request grants.EvaluationRequest
		rule    grants.GrantRule
	}{
		{name: "scope not allowed", rule: grants.GrantRule{ID: "allow-books", Condition: grants.RuleConditionCatalogScopeAllowed, Operands: []string{"books"}}, request: grants.EvaluationRequest{CatalogScope: "games", AvailableBalance: 10, RequestedAmount: 1, Now: now}},
		{name: "scope denied", rule: grants.GrantRule{ID: "deny-games", Condition: grants.RuleConditionCatalogScopeDenied, Operands: []string{"games"}}, request: grants.EvaluationRequest{CatalogScope: "games", AvailableBalance: 10, RequestedAmount: 1, Now: now}},
		{name: "expired", grant: grants.Grant{ExpiresAt: now}, rule: grants.GrantRule{ID: "expiry", Condition: grants.RuleConditionBeforeExpiry}, request: grants.EvaluationRequest{AvailableBalance: 10, RequestedAmount: 1, Now: now}},
		{name: "evidence missing", rule: grants.GrantRule{ID: "receipt", Condition: grants.RuleConditionRequiredEvidence, Operands: []string{"photo"}}, request: grants.EvaluationRequest{AvailableBalance: 10, RequestedAmount: 1, Now: now}},
		{name: "insufficient balance", rule: grants.GrantRule{ID: "balance", Condition: grants.RuleConditionSufficientBalance}, request: grants.EvaluationRequest{AvailableBalance: 2, RequestedAmount: 3, Now: now}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			grant := testCase.grant
			if grant.ExpiresAt.IsZero() {
				grant.ExpiresAt = now.Add(time.Hour)
			}
			grant.Rules = []grants.GrantRule{testCase.rule}
			testCase.request.CallerClaimsAuthorized = true
			decision, err := grants.NewRuleEvaluator().Evaluate(context.Background(), grant, testCase.request)
			require.NoError(t, err)
			require.False(t, decision.Allowed)
			require.Equal(t, testCase.rule.ID, decision.RuleID)
			require.Equal(t, testCase.rule.Condition, decision.Condition)
			require.Contains(t, decision.Reason, testCase.rule.ID)
			require.Contains(t, decision.Reason, string(testCase.rule.Condition))
		})
	}
}

func TestRuleEvaluatorAllowsOnlyWhenEveryRulePasses(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	grant := grants.Grant{
		ExpiresAt: now.Add(time.Hour),
		Rules: []grants.GrantRule{
			{ID: "allow-books", Condition: grants.RuleConditionCatalogScopeAllowed, Operands: []string{"books"}},
			{ID: "deny-games", Condition: grants.RuleConditionCatalogScopeDenied, Operands: []string{"games"}},
			{ID: "expiry", Condition: grants.RuleConditionBeforeExpiry},
			{ID: "receipt", Condition: grants.RuleConditionRequiredEvidence, Operands: []string{"photo"}},
			{ID: "balance", Condition: grants.RuleConditionSufficientBalance, AmountLimit: 5},
		},
	}
	decision, err := grants.NewRuleEvaluator().Evaluate(context.Background(), grant, grants.EvaluationRequest{
		CatalogScope: "books", Evidence: []string{"photo"}, AvailableBalance: 8,
		RequestedAmount: 4, Now: now,
	})
	require.NoError(t, err)
	require.True(t, decision.Allowed)
}
