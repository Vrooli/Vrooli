package restores

import (
	"testing"
	"time"
)

// TestTransition_Matrix covers the legal transitions and rejects illegal ones
// (FLOWS.md: verifying before requesting, any escape from a terminal state).
func TestTransition_Matrix(t *testing.T) {
	legal := []struct {
		from RestoreStatus
		ev   RestoreEvent
		want RestoreStatus
	}{
		{RestoreRequested, EventStartRestore, RestoreRestoring},
		{RestoreRequested, EventStartVerify, RestoreVerifying},
		{RestoreRestoring, EventRestored, RestoreRestored},
		{RestoreVerifying, EventVerified, RestoreVerified},
		{RestoreRequested, EventFail, RestoreFailed},
		{RestoreRestoring, EventFail, RestoreFailed},
		{RestoreVerifying, EventFail, RestoreFailed},
	}
	for _, tc := range legal {
		got, ok := Transition(tc.from, tc.ev)
		if !ok || got != tc.want {
			t.Errorf("Transition(%s,%s) = %s,%v; want %s,true", tc.from, tc.ev, got, ok, tc.want)
		}
	}

	illegal := []struct {
		from RestoreStatus
		ev   RestoreEvent
	}{
		{RestoreRequested, EventRestored},   // restored before restoring
		{RestoreRequested, EventVerified},   // verified before verifying
		{RestoreRestoring, EventVerified},   // can't verify from restoring path
		{RestoreVerifying, EventRestored},   // can't restore from verifying path
		{RestoreVerified, EventStartVerify}, // escape terminal
		{RestoreRestored, EventStartRestore},
		{RestoreFailed, EventFail},
	}
	for _, tc := range illegal {
		if _, ok := Transition(tc.from, tc.ev); ok {
			t.Errorf("Transition(%s,%s) should be illegal", tc.from, tc.ev)
		}
	}
}

// TestCheckInvariants_VerifiedRequiresLastVerifiedAt is the load-bearing safety
// test: a record with status=verified but no last_verified_at must fail
// invariants (OT-P0-006).
func TestCheckInvariants_VerifiedRequiresLastVerifiedAt(t *testing.T) {
	// Good: verified + last_verified_at + checksum.
	good := Restore{
		Status:         RestoreVerified,
		LastVerifiedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Checksum:       "abc123",
	}
	if err := CheckInvariants(good); err != nil {
		t.Errorf("good verified record failed invariants: %v", err)
	}

	// Bad: verified but zero last_verified_at.
	bad := Restore{Status: RestoreVerified, Checksum: "abc123"}
	if err := CheckInvariants(bad); err == nil {
		t.Error("verified record with zero last_verified_at must violate invariants")
	}

	// Bad: verified but empty checksum.
	bad2 := Restore{
		Status:         RestoreVerified,
		LastVerifiedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := CheckInvariants(bad2); err == nil {
		t.Error("verified record with empty checksum must violate invariants")
	}

	// Bad: failed but last_verified_at is set.
	bad3 := Restore{
		Status:         RestoreFailed,
		LastVerifiedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	if err := CheckInvariants(bad3); err == nil {
		t.Error("failed record with non-zero last_verified_at must violate invariants")
	}

	// Good: failed with zero last_verified_at.
	goodFailed := Restore{Status: RestoreFailed, Error: "boom"}
	if err := CheckInvariants(goodFailed); err != nil {
		t.Errorf("good failed record failed invariants: %v", err)
	}
}
