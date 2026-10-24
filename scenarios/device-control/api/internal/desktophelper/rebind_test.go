package desktophelper

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccessibilityRebindPreservesKeyAndRequiresStoppedHelper(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0700))
	path := filepath.Join(dir, "helper.json")
	requested := Config{Version: 1, Surface: targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}, SessionID: "2", XAuthorityFile: filepath.Join(dir, "authority"), GrantStatusFile: filepath.Join(dir, "grants"), StateDirectory: filepath.Join(dir, "state")}
	pin := base64.StdEncoding.EncodeToString(make([]byte, 32))
	current, err := ProvisionConfig(path, requested, func(previous string) (string, error) { require.Empty(t, previous); return pin, nil })
	require.NoError(t, err)
	bound := requested
	bound.AccessibilitySocket = "/run/user/1000/atspi-test.sock"
	bound.AccessibilityBusID = strings.Repeat("a", 32)
	ownerPath := filepath.Join(dir, "owner.json")
	require.NoError(t, writePrivateJSON(ownerPath, OwnerBootstrap{DeviceID: "host", HelperConfigPath: path, Helper: bound}))
	require.Error(t, checkOwnerConfiguration(ownerPath, path, current), "helper must wait for regenerated binding")
	require.NoError(t, privateDirectory(requested.StateDirectory))
	release, err := lockDirectory(requested.StateDirectory)
	require.NoError(t, err)
	called := false
	_, err = ProvisionConfig(path, bound, func(previous string) (string, error) { called = true; return pin, nil })
	require.Error(t, err)
	require.False(t, called, "active helper refusal precedes credential resolution")
	release()
	changed, err := ProvisionConfig(path, bound, func(previous string) (string, error) { require.Equal(t, pin, previous); return pin, nil })
	require.NoError(t, err)
	require.Equal(t, pin, changed.PublicKey)
	require.NoError(t, checkOwnerConfiguration(ownerPath, path, changed))
	wrong := bound
	wrong.SessionID = "other-session"
	_, err = ProvisionConfig(path, wrong, func(string) (string, error) { t.Fatal("destination change reached key resolver"); return pin, nil })
	require.Error(t, err)
	// Rebinding may not replace a surviving public pin.
	unbound := requested
	_, err = ProvisionConfig(path, unbound, func(previous string) (string, error) {
		require.Equal(t, pin, previous)
		return base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32))), nil
	})
	require.Error(t, err)
	unchanged, err := LoadConfig(path)
	require.NoError(t, err)
	require.Equal(t, changed, unchanged)
	restored, err := ProvisionConfig(path, unbound, func(previous string) (string, error) { require.Equal(t, pin, previous); return pin, nil })
	require.NoError(t, err)
	require.Empty(t, restored.AccessibilitySocket)
	require.Equal(t, pin, restored.PublicKey)
}
