package mocks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/capabilities"
)

func TestFakeChecker_Smoke(t *testing.T) {
	c := NewFakeChecker(capabilities.StatusAvailable, "fine")
	st, msg := c.Check(context.Background())
	require.Equal(t, capabilities.StatusAvailable, st)
	require.Equal(t, "fine", msg)
	require.Equal(t, int64(1), c.CallCount())
	c.ResetCalls()
	require.Equal(t, int64(0), c.CallCount())
}
