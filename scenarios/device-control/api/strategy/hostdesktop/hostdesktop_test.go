package hostdesktop

import (
	"context"
	"testing"

	"device-control/strategy"

	"github.com/stretchr/testify/require"
)

func TestHostDesktopStrategyCapabilityTier(t *testing.T) { // [REQ:DVC-P1-004]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	declaration, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Contains(t, declaration.SupportedHostOS, "darwin")
	require.NotContains(t, declaration.SupportedHostOS, "windows")
	require.NotEqual(t, strategy.StatusUnsupported, declaration.Status)
}

func TestHostDesktopRejectsUnverifiedWindowsPath(t *testing.T) {
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "windows"
	declaration, err := New().Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusUnsupported, declaration.Status)
	require.Contains(t, declaration.Reason, "unsupported")
}
