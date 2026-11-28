package entitlements

import "testing"

func TestDisabledBillingCannotMutateEntitlements(t *testing.T) {
	if _, err := (DisabledBilling{}).VerifyAndNormalize(nil, nil); err != ErrBillingUnavailable {
		t.Fatalf("err=%v", err)
	}
}

func TestEntitlementOnlyGatesOptionalComputeAndIgnoresOldEvents(t *testing.T) {
	state := New("w1")
	state.MonthlyLimit = 2
	var err error
	state, err = state.ReserveOptional(2)
	if err != nil {
		t.Fatal(err)
	}
	if err = state.CanUseOptional(1); err == nil {
		t.Fatal("expected budget refusal")
	}
	if !state.CoreAllowed() {
		t.Fatal("core product must remain available")
	}
	state, err = state.ApplyEvent(Event{ID: "new", Version: 2, OptionalCompute: false, MonthlyLimit: 0})
	if err != nil {
		t.Fatal(err)
	}
	state, err = state.ApplyEvent(Event{ID: "old", Version: 1, OptionalCompute: true, MonthlyLimit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 || state.OptionalCompute {
		t.Fatalf("old event rewrote entitlement: %#v", state)
	}
}
