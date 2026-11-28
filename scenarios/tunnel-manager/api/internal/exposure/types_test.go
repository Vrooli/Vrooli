package exposure

import (
	"errors"
	"testing"
)

func TestLeaseTransitionStatusRejectsIllegalEdgesWithoutMutation(t *testing.T) {
	tests := []struct {
		name string
		from LeaseStatus
		to   LeaseStatus
		want LeaseStatus
		ok   bool
	}{
		{name: "active extension", from: LeaseActive, to: LeaseActive, want: LeaseActive, ok: true},
		{name: "expire active", from: LeaseActive, to: LeaseExpired, want: LeaseExpired, ok: true},
		{name: "revoke active", from: LeaseActive, to: LeaseRevoked, want: LeaseRevoked, ok: true},
		{name: "revoke expired", from: LeaseExpired, to: LeaseRevoked, want: LeaseRevoked, ok: true},
		{name: "expired cannot revive", from: LeaseExpired, to: LeaseActive, want: LeaseExpired},
		{name: "revoked cannot revive", from: LeaseRevoked, to: LeaseActive, want: LeaseRevoked},
		{name: "terminal cannot expire", from: LeaseRevoked, to: LeaseExpired, want: LeaseRevoked},
		{name: "unknown source", from: "corrupt", to: LeaseActive, want: "corrupt"},
		{name: "unknown target", from: LeaseActive, to: "corrupt", want: LeaseActive},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lease := Lease{Status: test.from}
			err := lease.TransitionStatus(test.to)
			if test.ok && err != nil {
				t.Fatalf("TransitionStatus() error = %v", err)
			}
			if !test.ok {
				var invalid ErrInvalidLeaseTransition
				if !errors.As(err, &invalid) {
					t.Fatalf("TransitionStatus() error = %v, want ErrInvalidLeaseTransition", err)
				}
			}
			if lease.Status != test.want {
				t.Fatalf("status = %q, want %q", lease.Status, test.want)
			}
		})
	}
}
