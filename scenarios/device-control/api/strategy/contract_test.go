package strategy_test

import (
	"context"
	"testing"

	strategy "device-control/strategy"
	"device-control/strategy/androidadb"
	"device-control/strategy/iosmirror"
	"device-control/strategy/iossimctl"
	"device-control/strategy/iosxcuitest"
	"github.com/stretchr/testify/require"
)

type successfulProbeRunner struct{}

func TestHostResolutionIsTerminalBeforePrerequisites(t *testing.T) { // [REQ:DVC-P0-001]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	for _, adapter := range []strategy.Strategy{iossimctl.New(), iosxcuitest.New(), iosmirror.New()} {
		declaration, err := adapter.Describe(context.Background())
		require.NoError(t, err)
		require.Equal(t, strategy.StatusUnsupported, declaration.Status, adapter.ID())
		require.Empty(t, declaration.NextActions, adapter.ID())
		require.Contains(t, declaration.SupportedHostOS, "darwin")
	}
}

func TestSupportedHostUsesPrerequisiteDispositionOnForcedDarwin(t *testing.T) { // [REQ:DVC-P0-001]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "darwin"
	for _, adapter := range []strategy.Strategy{iossimctl.New(), iosxcuitest.New(), iosmirror.New()} {
		declaration, err := adapter.Describe(context.Background())
		require.NoError(t, err, adapter.ID())
		require.Equal(t, strategy.StatusUnavailable, declaration.Status, adapter.ID())
		require.NotEmpty(t, declaration.NextActions, adapter.ID())
	}
}

func TestEvidenceClassIsStructuralAndIosMirrorIsNotPromotable(t *testing.T) { // [REQ:DVC-P1-003]
	old := strategy.HostOS
	t.Cleanup(func() { strategy.HostOS = old })
	strategy.HostOS = "linux"
	for _, adapter := range []strategy.Strategy{iossimctl.New(), iosxcuitest.New(), iosmirror.New()} {
		declaration, err := adapter.Describe(context.Background())
		require.NoError(t, err, adapter.ID())
		require.NotEmpty(t, declaration.EvidenceClass, adapter.ID())
		if adapter.ID() == "ios-mirror" {
			require.False(t, declaration.Promotable)
			require.Equal(t, "advisory-ocr", declaration.EvidenceClass)
		}
	}

	android, err := androidadb.NewWithRunner(successfulProbeRunner{}, "").Describe(context.Background())
	require.NoError(t, err)
	require.Equal(t, "release-grade", android.EvidenceClass)
	require.True(t, android.Promotable)
}

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
	for _, kind := range []string{"observe", "tap", "key", "wait", "swipe", "long-press", "double-tap", "drag", "fling", "scroll-to", "text", "screen", "semantic-target", "semantic-assert", "install", "launch", "stop", "uninstall", "clear-data", "package-state", "grant-permission", "revoke-permission", "device-logs", "logcat-start", "logcat-stop", "clock-sample", "screenshot", "clipboard-read", "clipboard-write", "screenrecord", "rotate", "network", "deep-link", "share"} {
		implemented[kind] = true
	}
	implemented["recording-start"] = true
	implemented["recording-stop"] = true
	for _, kind := range strategy.StepKinds(declaration) {
		require.Truef(t, implemented[kind], "capability declaration exposed unsupported step %q", kind)
	}
}
