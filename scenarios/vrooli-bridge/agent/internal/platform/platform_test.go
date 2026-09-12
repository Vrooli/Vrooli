package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNativeServiceManager_KnownForBuildTarget(t *testing.T) {
	got := NativeServiceManager()
	switch runtime.GOOS {
	case "linux":
		require.Equal(t, ServiceManagerSystemd, got)
	case "darwin":
		require.Equal(t, ServiceManagerLaunchd, got)
	case "windows":
		require.Equal(t, ServiceManagerWindows, got)
	default:
		require.Equal(t, ServiceManagerUnknown, got)
	}
}

func TestStateDir_OverrideIsCreated(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "nested", "agent")
	t.Setenv("BRIDGE_AGENT_STATE_DIR", target)

	got, err := StateDir()
	require.NoError(t, err)
	require.Equal(t, target, got)

	info, err := os.Stat(got)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestStateDir_DefaultUnderUserConfigDir(t *testing.T) {
	// Force a clean override-free resolution by pointing the OS user config
	// dir at a temp location via the platform-appropriate env var.
	t.Setenv("BRIDGE_AGENT_STATE_DIR", "")
	base := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", base)
	case "darwin":
		// os.UserConfigDir uses $HOME/Library/Application Support on darwin.
		t.Setenv("HOME", base)
	default:
		t.Setenv("XDG_CONFIG_HOME", base)
	}

	got, err := StateDir()
	require.NoError(t, err)
	require.Contains(t, got, "vrooli-bridge-agent")

	info, err := os.Stat(got)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}
