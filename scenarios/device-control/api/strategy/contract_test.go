package strategy_test

import (
	"context"
	"testing"

	strategy "device-control/strategy"
	"device-control/strategy/androidadb"
	"github.com/stretchr/testify/require"
)

type successfulProbeRunner struct{}

func (successfulProbeRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	if name == "adb" && len(args) == 1 && args[0] == "devices" {
		return []byte("List of devices attached\nserial-1 device\n"), nil
	}
	if name == "adb" && len(args) >= 4 && args[0] == "-s" && args[2] == "shell" && args[3] == "getprop" {
		return []byte("13\n"), nil
	}
	return []byte("probe ok"), nil
}

func TestDeclaredAndroidStepsHaveExecutorImplementations(t *testing.T) {
	declaration, err := androidadb.NewWithRunner(successfulProbeRunner{}, "").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, strategy.StatusAvailable, declaration.Status)
	implemented := map[string]bool{}
	for _, kind := range []string{"observe", "tap", "key", "wait", "swipe", "text", "semantic-target", "semantic-assert", "install", "launch", "stop", "uninstall", "clear-data", "package-state", "grant-permission", "revoke-permission", "device-logs", "screenrecord", "rotate", "network", "deep-link"} {
		implemented[kind] = true
	}
	for _, kind := range strategy.StepKinds(declaration) {
		require.Truef(t, implemented[kind], "capability declaration exposed unsupported step %q", kind)
	}
}
