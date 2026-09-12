package metering

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeGateway struct{ reserved, finalized, released bool }

func (g *fakeGateway) Reserve(context.Context, string, int64) (string, error) {
	g.reserved = true
	return "1", nil
}
func (g *fakeGateway) Finalize(context.Context, string, int64) error { g.finalized = true; return nil }
func (g *fakeGateway) Release(context.Context, string) error         { g.released = true; return nil }

// [REQ:SWBD-P1-014]
func TestBYOKDoesNotCharge(t *testing.T) {
	g := &fakeGateway{}
	r, err := Run(context.Background(), g, "operator-key", "b", 2, func() (int64, error) { return 0, nil })
	require.NoError(t, err)
	require.True(t, r.BYOK)
	require.False(t, g.reserved)
	require.False(t, g.finalized)
	require.False(t, g.released)
}

// [REQ:SWBD-P1-014]
func TestHostedReserveExecuteAndFinalize(t *testing.T) {
	g := &fakeGateway{}
	executed := false
	_, err := Run(context.Background(), g, "", "b", 2, func() (int64, error) { executed = true; return 2, nil })
	require.NoError(t, err)
	require.True(t, g.reserved)
	require.True(t, executed)
	require.True(t, g.finalized)
	require.False(t, g.released)
}

func TestHostedExecutionFailureReleasesReservation(t *testing.T) {
	g := &fakeGateway{}
	want := errors.New("inference failed")
	_, err := Run(context.Background(), g, "", "b", 2, func() (int64, error) { return 0, want })
	require.ErrorIs(t, err, want)
	require.True(t, g.reserved)
	require.False(t, g.finalized)
	require.True(t, g.released)
}
