package androidadb

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWithConfigUsesResolvedEnvironmentValues(t *testing.T) {
	adapter := NewWithConfig(Config{Serial: "usb:device-1", RequireDefaultServer: true})
	require.Equal(t, "usb:device-1", adapter.serial)
	runner, ok := adapter.runner.(commandRunner)
	require.True(t, ok)
	require.True(t, runner.requireDefaultADBServer)
}

func TestShellQuotePreservesIntentExtraBoundaries(t *testing.T) {
	require.Equal(t, "'Hello Mobile share probe'", shellQuote("Hello Mobile share probe"))
	require.Equal(t, "'operator'\\''s note'", shellQuote("operator's note"))
}

func TestRestartUsesOnlyExactBoundTransportAndReturnsAcceptance(t *testing.T) {
	runner := &scriptedRunner{}
	require.Error(t, NewWithRunner(runner, "").RestartDevice(context.Background()))
	require.Empty(t, runner.calls)
	adapter := NewWithRunner(runner, "serial-1")
	require.NoError(t, adapter.RestartDevice(context.Background()))
	require.Equal(t, []string{"adb -s serial-1 reboot"}, runner.calls)
	runner.errors = map[string]error{"adb -s serial-1 reboot": errors.New("transport refused")}
	require.ErrorContains(t, adapter.RestartDevice(context.Background()), "transport refused")
	require.Len(t, runner.calls, 2, "owner must not retry a restart with uncertain effect")
}
