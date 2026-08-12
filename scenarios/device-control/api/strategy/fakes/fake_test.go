package fakes

import (
	"context"
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestStrategyProvidesReplayableFloor(t *testing.T) { // [REQ:DVC-P0-001]
	s := New("fake", strategy.StatusAvailable, strategy.CapInput, strategy.CapScreenshot)
	frame, err := s.Observe(context.Background())
	require.NoError(t, err)
	require.Equal(t, "image/png", frame.MediaType)
	require.NotEmpty(t, frame.Bytes)
	require.NoError(t, s.Actuate(context.Background(), strategy.Actuation{Key: &strategy.KeyEvent{Kind: "press", Key: "ENTER"}}))
	require.Len(t, s.Calls(), 1)
}
