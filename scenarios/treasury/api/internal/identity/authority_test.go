package identity_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"

	"treasury/internal/identity"
	"treasury/internal/testutil/agentclaims"
)

func TestTreasuryAdmissionUsesCanonicalGrantRules(t *testing.T) {
	now := time.Now()
	claims := agentclaims.For("agent")
	grant := requestidentity.OperationGrant{ID: "g", Approver: "operator", ExpiresAt: now.Add(time.Hour), Rules: []requestidentity.OperationRule{
		{Operations: []string{"treasury.spend"}, Targets: []string{"treasury:budget/research/*"}},
		{Operations: []string{"treasury.read"}, Targets: []string{"treasury:budget/private"}},
	}}
	raw, err := json.Marshal(grant)
	require.NoError(t, err)
	claims.Meta["operation_grant"] = string(raw)
	require.NoError(t, identity.Authorize(claims, "treasury.spend", "treasury:budget/research/one", now))
	require.Error(t, identity.Authorize(claims, "treasury.spend", "treasury:budget/private", now), "operations and targets from different policies cannot cross-match")
	require.NoError(t, identity.Authorize(claims, "treasury.read", "treasury:budget/private", now))
	require.Error(t, identity.Authorize(claims, "treasury.spend", "treasury:budget/research/one", now.Add(time.Hour)))
}

func TestTreasurySpendGrantBindsOneAuthorizationTarget(t *testing.T) {
	now := time.Now()
	exactIntent := cliutil.TreasurySpendIntent{
		AuthorizationID: "auth-expected", MandateID: "mandate-1", AmountMinor: 110, Currency: "USD", Counterparty: "merchant:green",
		AuthorizationIdempotencyKey: "idem-1", SettlementID: "settlement-1", InstrumentID: "base-usdc-wallet", SettlementIdempotencyKey: "settle-idem-1",
	}
	exactProposal, exactSettlement := exactIntent.Targets()
	grant := requestidentity.OperationGrant{ID: "one-spend", Approver: "operator", ExpiresAt: now.Add(time.Hour), Rules: []requestidentity.OperationRule{
		{Operations: []string{"treasury.spend"}, Targets: []string{exactProposal, exactSettlement}},
		{Operations: []string{"treasury.read"}, Targets: []string{"treasury:budget/budget-1"}},
	}}
	raw, err := json.Marshal(grant)
	require.NoError(t, err)
	claims := agentclaims.For("agent")
	claims.Meta["operation_grant"] = string(raw)
	require.NoError(t, identity.Authorize(claims, "treasury.spend", exactProposal, now))
	require.Error(t, identity.Authorize(claims, "treasury.spend", cliutil.TreasurySpendIntentAuthorizationTarget(func() cliutil.TreasurySpendIntent { i := exactIntent; i.AuthorizationID = "auth-other"; return i }()), now))
	for name, changedIntent := range map[string]cliutil.TreasurySpendIntent{
		"mandate":      func() cliutil.TreasurySpendIntent { i := exactIntent; i.MandateID = "mandate-2"; return i }(),
		"amount":       func() cliutil.TreasurySpendIntent { i := exactIntent; i.AmountMinor++; return i }(),
		"currency":     func() cliutil.TreasurySpendIntent { i := exactIntent; i.Currency = "EUR"; return i }(),
		"counterparty": func() cliutil.TreasurySpendIntent { i := exactIntent; i.Counterparty = "merchant:blue"; return i }(),
		"idempotency": func() cliutil.TreasurySpendIntent {
			i := exactIntent
			i.AuthorizationIdempotencyKey = "idem-2"
			return i
		}(),
		"instrument":    func() cliutil.TreasurySpendIntent { i := exactIntent; i.InstrumentID = "other-wallet"; return i }(),
		"settlement ID": func() cliutil.TreasurySpendIntent { i := exactIntent; i.SettlementID = "settlement-2"; return i }(),
		"settlement key": func() cliutil.TreasurySpendIntent {
			i := exactIntent
			i.SettlementIdempotencyKey = "settle-idem-2"
			return i
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			changedProposal, changedSettlement := changedIntent.Targets()
			require.Error(t, identity.Authorize(claims, "treasury.spend", changedProposal, now))
			if changedSettlement != exactSettlement {
				require.Error(t, identity.Authorize(claims, "treasury.spend", changedSettlement, now))
			}
		})
	}
	require.Error(t, identity.Authorize(claims, "treasury.spend", "treasury:budget/budget-1", now))
	require.NoError(t, identity.Authorize(claims, "treasury.read", "treasury:budget/budget-1", now))
}

func TestTreasurySettlementGrantBindsInstrumentAndExactIntent(t *testing.T) {
	now := time.Now()
	exact := cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "merchant:green", "settlement-1", "base-usdc-wallet", "settle-idem-1")
	grant := requestidentity.OperationGrant{ID: "one-settlement", Approver: "operator", ExpiresAt: now.Add(time.Hour), Rules: []requestidentity.OperationRule{
		{Operations: []string{"treasury.spend"}, Targets: []string{exact}},
	}}
	raw, err := json.Marshal(grant)
	require.NoError(t, err)
	claims := agentclaims.For("agent")
	claims.Meta["operation_grant"] = string(raw)
	require.NoError(t, identity.Authorize(claims, "treasury.spend", exact, now))
	for name, changed := range map[string]string{
		"cross-authorization": cliutil.TreasurySettlementTarget("auth-2", "mandate-1", 110, "USD", "merchant:green", "settlement-1", "base-usdc-wallet", "settle-idem-1"),
		"amount":              cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 111, "USD", "merchant:green", "settlement-1", "base-usdc-wallet", "settle-idem-1"),
		"counterparty":        cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "merchant:blue", "settlement-1", "base-usdc-wallet", "settle-idem-1"),
		"settlement":          cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "merchant:green", "settlement-2", "base-usdc-wallet", "settle-idem-1"),
		"instrument":          cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "merchant:green", "settlement-1", "different-wallet", "settle-idem-1"),
		"idempotency":         cliutil.TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "merchant:green", "settlement-1", "base-usdc-wallet", "settle-idem-2"),
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, identity.Authorize(claims, "treasury.spend", changed, now))
		})
	}
	require.Error(t, identity.Authorize(claims, "treasury.spend", exact, now.Add(time.Hour)), "expired grant cannot replay")
}
