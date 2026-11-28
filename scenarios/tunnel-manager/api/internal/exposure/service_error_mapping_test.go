package exposure

import (
	"testing"

	"connectrpc.com/connect"
)

func TestToConnectErrorMapsLeaseTransitionFailuresToPrecondition(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "invalid edge", err: ErrInvalidLeaseTransition{From: LeaseRevoked, To: LeaseActive}},
		{name: "stale compare and swap", err: ErrLeaseWriteConflict{ID: "lease-1", ExpectedStatus: LeaseActive, CurrentStatus: LeaseRevoked}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := connect.CodeOf(ToConnectError(test.err))
			if got != connect.CodeFailedPrecondition {
				t.Fatalf("Connect code = %s, want %s", got, connect.CodeFailedPrecondition)
			}
		})
	}
}
