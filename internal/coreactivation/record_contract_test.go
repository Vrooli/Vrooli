package coreactivation

import (
	"context"
	"strings"
	"testing"
)

// A compatible transition may carry a different whole-source identity: the
// versioned manifest is the authority, and the recorded identities are audit
// evidence. Requiring equality here would make the legacy identity digest
// override a directional answer the frozen images already proved safe.
func TestRecordValidationDoesNotRequireIdentityContractEquality(t *testing.T) {
	e, r, _, _ := fixture(t)
	r.SchemaVersion = recordSchemaVersion
	r.Revision = 1
	r.Phase = Prepared
	r.Fence = Ready
	r.CandidateStateContract = strings.Repeat("c", 64)
	r.PriorStateContract = strings.Repeat("d", 64)
	if err := r.validate(e.Store.home); err != nil {
		t.Fatalf("a compatible record with different identity contracts was rejected: %v", err)
	}
}

func TestPrepareAdmitsACompatibleRecordWithDifferentIdentityContracts(t *testing.T) {
	e, r, _, _ := fixture(t)
	r.CandidateStateContract = strings.Repeat("c", 64)
	r.PriorStateContract = strings.Repeat("d", 64)
	// The directional qualification is what decides; it answers for the frozen
	// images, not for the identity constants recorded beside them.
	e.qualifyFrozen = func(context.Context, Record) error { return nil }
	admitted, err := e.Begin(context.Background(), r, false)
	if err != nil {
		t.Fatalf("compatible activation refused because the legacy identities differ: %v", err)
	}
	if admitted.CandidateStateContract == admitted.PriorStateContract {
		t.Fatal("fixture did not keep the identities distinct")
	}
}

func TestPrepareStillRefusesAnIncompatibleRecordWithEqualIdentityContracts(t *testing.T) {
	e, r, _, _ := fixture(t)
	e.qualifyFrozen = func(context.Context, Record) error { return ErrStateContractChanged }
	if _, err := e.Begin(context.Background(), r, false); err == nil {
		t.Fatal("equal identity contracts overrode an incompatible frozen pair")
	}
}
