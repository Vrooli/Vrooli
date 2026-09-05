package desktophelper

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/targetmodel"
)

func TestOwnerBootstrapRefusesUnprotectedOrCollidingConfiguration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux bootstrap")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "owner.json")
	config := OwnerBootstrap{DeviceID: "host-desktop", HelperConfigPath: filepath.Join(dir, "helper.json"), Helper: Config{Version: 1, Surface: targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "host", HostNodeID: "host"}, OwnerScenario: "device-control", SurfaceID: "desktop"}, SessionID: "2", XAuthorityFile: filepath.Join(dir, "authority"), StateDirectory: filepath.Join(dir, "state"), GrantStatusFile: filepath.Join(dir, "grants")}}
	write := func(c OwnerBootstrap) {
		data, err := json.Marshal(c)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, data, 0600))
	}
	write(config)
	loaded, err := LoadOwnerBootstrap(path)
	require.NoError(t, err)
	require.Equal(t, config, loaded)
	bound := config
	bound.Helper.AccessibilitySocket = "/run/user/1000/atspi-test.sock"
	bound.Helper.AccessibilityBusID = strings.Repeat("a", 32)
	write(bound)
	loaded, err = LoadOwnerBootstrap(path)
	require.NoError(t, err)
	require.Equal(t, bound, loaded)
	for _, mutate := range []func(*OwnerBootstrap){
		func(c *OwnerBootstrap) { c.Helper.AccessibilitySocket = "/tmp/bus" },
		func(c *OwnerBootstrap) { c.Helper.AccessibilityBusID = strings.Repeat("a", 32) },
		func(c *OwnerBootstrap) {
			c.Helper.AccessibilitySocket = "relative"
			c.Helper.AccessibilityBusID = strings.Repeat("a", 32)
		},
		func(c *OwnerBootstrap) {
			c.Helper.AccessibilitySocket = path
			c.Helper.AccessibilityBusID = strings.Repeat("a", 32)
		},
		func(c *OwnerBootstrap) {
			c.Helper.AccessibilitySocket = c.HelperConfigPath
			c.Helper.AccessibilityBusID = strings.Repeat("a", 32)
		},
		func(c *OwnerBootstrap) {
			c.Helper.AccessibilitySocket = "/tmp/bus"
			c.Helper.AccessibilityBusID = "invalid"
		},
		func(c *OwnerBootstrap) { c.Helper.GrantStatusFile = path },
		func(c *OwnerBootstrap) { c.HelperConfigPath = c.Helper.XAuthorityFile },
		func(c *OwnerBootstrap) { c.Helper.GrantStatusFile = c.Helper.XAuthorityFile },
		func(c *OwnerBootstrap) { c.Helper.PublicKey = "caller-supplied-pin" },
	} {
		bad := config
		mutate(&bad)
		write(bad)
		_, err = LoadOwnerBootstrap(path)
		require.Error(t, err)
	}
	write(config)
	require.NoError(t, os.Chmod(path, 0644))
	_, err = LoadOwnerBootstrap(path)
	require.Error(t, err)
}
