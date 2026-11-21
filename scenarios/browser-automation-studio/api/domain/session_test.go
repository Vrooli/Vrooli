package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRecordingSessionTransitionStatus(t *testing.T) {
	tests := []struct {
		name    string
		from    SessionStatus
		to      SessionStatus
		wantErr bool
	}{
		{name: "active closes", from: SessionStatusActive, to: SessionStatusClosed},
		{name: "closed is idempotent", from: SessionStatusClosed, to: SessionStatusClosed},
		{name: "closed cannot reopen", from: SessionStatusClosed, to: SessionStatusActive, wantErr: true},
		{name: "unknown current state", from: "paused", to: SessionStatusClosed, wantErr: true},
		{name: "unknown next state", from: SessionStatusActive, to: "paused", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &RecordingSession{Status: test.from}
			err := session.TransitionStatus(test.to)
			if test.wantErr {
				if !errors.Is(err, ErrInvalidSessionStatusTransition) {
					t.Fatalf("TransitionStatus error = %v, want invalid transition", err)
				}
				if session.Status != test.from {
					t.Fatalf("invalid transition changed status to %q", session.Status)
				}
				return
			}
			if err != nil {
				t.Fatalf("TransitionStatus: %v", err)
			}
			if session.Status != test.to {
				t.Fatalf("status = %q, want %q", session.Status, test.to)
			}
		})
	}
}

func TestRecordingSessionCloseOwnsStatusAndTimestamp(t *testing.T) {
	closedAt := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	session := &RecordingSession{Status: SessionStatusActive}
	if err := session.Close(closedAt); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if session.Status != SessionStatusClosed || session.ClosedAt == nil || !session.ClosedAt.Equal(closedAt) {
		t.Fatalf("closed session = %+v", session)
	}
}
