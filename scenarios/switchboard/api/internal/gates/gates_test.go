package gates

import (
	"errors"
	"testing"
	"time"
)

// [REQ:SWBD-P1-009]
func TestGateRoutesOnlyToOwnerAndExpires(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	s := New(func() time.Time { return now })
	g, err := s.Raise("thread-1", "owner-1", "filesystem.write", "writing outside the workspace", "owner approval", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !g.GrantOnce || g.Status != Pending {
		t.Fatalf("initial gate = %+v", g)
	}
	if _, err := s.Answer(g.ID, "viewer-1", true); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("non-owner error = %v", err)
	}
	now = now.Add(2 * time.Minute)
	expired, ok := s.Get(g.ID)
	if !ok || expired.Status != Expired {
		t.Fatalf("expired gate = %+v, ok=%v", expired, ok)
	}
	if _, err := s.Answer(g.ID, "owner-1", true); !errors.Is(err, ErrNotPending) {
		t.Fatalf("expired answer error = %v", err)
	}
}

// [REQ:SWBD-P1-009]
func TestGateDefaultsToOneTurnGrant(t *testing.T) {
	s := New(nil)
	g, err := s.Raise("thread-1", "owner-1", "read", "private context", "owner approval", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if !g.GrantOnce {
		t.Fatal("gate must grant once by default")
	}
	got, err := s.Answer(g.ID, "owner-1", true)
	if err != nil || got.Status != Granted {
		t.Fatalf("grant result = %+v, err=%v", got, err)
	}
}

func TestGateStatusOwnsTerminalTransitions(t *testing.T) {
	for _, tc := range []struct {
		from, to Status
		wantErr  bool
	}{
		{Pending, Granted, false},
		{Pending, Denied, false},
		{Pending, Expired, false},
		{Granted, Denied, true},
		{Expired, Granted, true},
		{Status("unknown"), Granted, true},
		{Pending, Status("unknown"), true},
	} {
		gate := Gate{Status: tc.from}
		err := gate.TransitionTo(tc.to)
		if (err != nil) != tc.wantErr {
			t.Errorf("transition %q -> %q error = %v, wantErr %v", tc.from, tc.to, err, tc.wantErr)
		}
		if tc.wantErr && gate.Status != tc.from {
			t.Errorf("invalid transition mutated status to %q", gate.Status)
		}
	}
}
