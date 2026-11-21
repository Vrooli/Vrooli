package cliutil

import "testing"

func TestTreasuryAuthorizationTargetBindsImmutableSpendFields(t *testing.T) {
	want := TreasuryAuthorizationTarget("auth-1", "mandate-1", 110, "usd", "0xabc", "idem-1")
	if want != TreasuryAuthorizationTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "idem-1") {
		t.Fatal("currency case normalization changed the target")
	}
	for name, target := range map[string]string{
		"authorization ID": TreasuryAuthorizationTarget("auth-2", "mandate-1", 110, "USD", "0xabc", "idem-1"),
		"mandate":          TreasuryAuthorizationTarget("auth-1", "mandate-2", 110, "USD", "0xabc", "idem-1"),
		"amount":           TreasuryAuthorizationTarget("auth-1", "mandate-1", 111, "USD", "0xabc", "idem-1"),
		"currency":         TreasuryAuthorizationTarget("auth-1", "mandate-1", 110, "EUR", "0xabc", "idem-1"),
		"counterparty":     TreasuryAuthorizationTarget("auth-1", "mandate-1", 110, "USD", "0xdef", "idem-1"),
		"idempotency":      TreasuryAuthorizationTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "idem-2"),
	} {
		t.Run(name, func(t *testing.T) {
			if target == want {
				t.Fatalf("changed spend field reused target %q", target)
			}
		})
	}
}

func TestTreasurySettlementTargetBindsExactSettlement(t *testing.T) {
	want := TreasurySettlementTarget("auth-1", "mandate-1", 110, "usd", "0xabc", "settle-1", "instrument-1", "idem-1")
	if want != TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "settle-1", "instrument-1", "idem-1") {
		t.Fatal("currency case normalization changed settlement target")
	}
	for name, target := range map[string]string{
		"authorization ID": TreasurySettlementTarget("auth-2", "mandate-1", 110, "USD", "0xabc", "settle-1", "instrument-1", "idem-1"),
		"mandate":          TreasurySettlementTarget("auth-1", "mandate-2", 110, "USD", "0xabc", "settle-1", "instrument-1", "idem-1"),
		"amount":           TreasurySettlementTarget("auth-1", "mandate-1", 111, "USD", "0xabc", "settle-1", "instrument-1", "idem-1"),
		"counterparty":     TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "0xdef", "settle-1", "instrument-1", "idem-1"),
		"settlement ID":    TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "settle-2", "instrument-1", "idem-1"),
		"instrument":       TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "settle-1", "instrument-2", "idem-1"),
		"idempotency":      TreasurySettlementTarget("auth-1", "mandate-1", 110, "USD", "0xabc", "settle-1", "instrument-1", "idem-2"),
	} {
		t.Run(name, func(t *testing.T) {
			if target == want {
				t.Fatalf("changed settlement field reused target %q", target)
			}
		})
	}
}

func TestTreasurySpendIntentProducesOneCoherentApprovalPair(t *testing.T) {
	intent := TreasurySpendIntent{
		AuthorizationID: "auth-7", MandateID: "mandate-2", AmountMinor: 450, Currency: "USD",
		Counterparty: "research-provider", AuthorizationIdempotencyKey: "auth-idem-7",
		SettlementID: "settlement-7", InstrumentID: "base-usdc-budget-2", SettlementIdempotencyKey: "settle-idem-7",
	}
	proposal, settlement := intent.Targets()
	if proposal == settlement {
		t.Fatal("proposal and settlement targets must authorize different operations")
	}
	if proposal != TreasurySpendIntentAuthorizationTarget(intent) || settlement != TreasurySettlementTarget(intent.AuthorizationID, intent.MandateID, intent.AmountMinor, intent.Currency, intent.Counterparty, intent.SettlementID, intent.InstrumentID, intent.SettlementIdempotencyKey) {
		t.Fatal("coherent intent targets differ from their canonical target derivations")
	}
	changedInstrument := intent
	changedInstrument.InstrumentID = "different-wallet"
	changedProposal, changedSettlement := changedInstrument.Targets()
	if proposal == changedProposal || settlement == changedSettlement {
		t.Fatal("changing the approved instrument did not alter both immutable-intent targets")
	}
	changedAmount := intent
	changedAmount.AmountMinor++
	changedAmountProposal, changedAmountSettlement := changedAmount.Targets()
	if proposal == changedAmountProposal || settlement == changedAmountSettlement {
		t.Fatal("changing the amount failed to alter both proposal and settlement targets")
	}
	changedSettlementKey := intent
	changedSettlementKey.SettlementIdempotencyKey = "settle-idem-other"
	changedSettlementKeyProposal, changedSettlementKeyTarget := changedSettlementKey.Targets()
	if proposal == changedSettlementKeyProposal || settlement == changedSettlementKeyTarget {
		t.Fatal("changing the settlement idempotency key failed to alter both intent targets")
	}
}
