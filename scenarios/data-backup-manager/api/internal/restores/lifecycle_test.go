package restores

import (
	"testing"
	"time"
)

// TestTransition_Matrix covers the legal transitions and rejects illegal ones
// (FLOWS.md: verifying before requesting, any escape from a terminal state).
func TestTransition_Matrix(t *testing.T) {
	states := []RestoreStatus{RestoreRequested, RestoreRestoring, RestoreVerifying, RestoreVerified, RestoreRestored, RestoreFailed}
	events := []RestoreEvent{EventStartRestore, EventStartVerify, EventRestored, EventVerified, EventFail}
	legal := map[struct {
		from  RestoreStatus
		event RestoreEvent
	}]RestoreStatus{
		{RestoreRequested, EventStartRestore}: RestoreRestoring,
		{RestoreRequested, EventStartVerify}:  RestoreVerifying,
		{RestoreRestoring, EventRestored}:     RestoreRestored,
		{RestoreVerifying, EventVerified}:     RestoreVerified,
		{RestoreRequested, EventFail}:         RestoreFailed,
		{RestoreRestoring, EventFail}:         RestoreFailed,
		{RestoreVerifying, EventFail}:         RestoreFailed,
	}
	for _, from := range states {
		for _, event := range events {
			key := struct {
				from  RestoreStatus
				event RestoreEvent
			}{from, event}
			want, wantOK := legal[key]
			got, gotOK := Transition(from, event)
			if gotOK != wantOK || (wantOK && got != want) || (!wantOK && got != from) {
				t.Errorf("Transition(%s,%s) = %s,%v; want %s,%v", from, event, got, gotOK, want, wantOK)
			}
		}
	}
}

func TestRestoreTransitionRejectsWithoutMutation(t *testing.T) {
	tests := []struct {
		name   string
		status RestoreStatus
		event  RestoreEvent
	}{
		{name: "unknown state", status: "future-state", event: EventFail},
		{name: "wrong branch", status: RestoreRestoring, event: EventVerified},
		{name: "terminal state", status: RestoreVerified, event: EventFail},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			record := Restore{Status: tc.status}
			if err := record.Transition(tc.event); err == nil {
				t.Fatal("expected invalid transition error")
			}
			if record.Status != tc.status {
				t.Fatalf("invalid transition mutated state to %q; want %q", record.Status, tc.status)
			}
		})
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
