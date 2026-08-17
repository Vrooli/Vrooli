package main

import (
	"testing"

	"vrooli-bridge/agent/internal/config"

	"github.com/stretchr/testify/require"
)

// [REQ:BRG-P0-007] The `service` verbs pull --json out of the arg stream before
// handing the remainder to config.Load (which owns every other flag), so
// --json can appear anywhere without colliding with the config flag set.
func TestExtractJSONFlag(t *testing.T) {
	rest, asJSON := extractJSONFlag([]string{"--control-plane-url", "https://cp", "--json", "--node-id", "n1"})
	require.True(t, asJSON)
	require.Equal(t, []string{"--control-plane-url", "https://cp", "--node-id", "n1"}, rest)

	rest, asJSON = extractJSONFlag([]string{"--node-id", "n1"})
	require.False(t, asJSON)
	require.Equal(t, []string{"--node-id", "n1"}, rest)

	_, asJSON = extractJSONFlag([]string{"-json"})
	require.True(t, asJSON)
}

// [REQ:BRG-P0-007] serviceDefinition embeds the same control-plane URL / node id
// / state dir the agent dials with, so the installed service reconnects exactly
// as the foreground process would.
func TestServiceDefinition_EmbedsDialArgs(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		ControlPlaneURL: "https://cp.example",
		NodeID:          "n1",
		StateDir:        dir,
		VrooliBin:       "/Users/test/.vrooli/bin/vrooli",
		Capabilities:    []string{"scenario status*", "scenario test*"},
		PresenceOnly:    false,
	}
	def, err := serviceDefinition(cfg)
	require.NoError(t, err)
	require.Equal(t, "vrooli-bridge-agent", def.Name)
	require.NotEmpty(t, def.ExecPath) // the running test binary's path
	require.Equal(t, []string{
		"--control-plane-url", "https://cp.example",
		"--node-id", "n1",
		"--state-dir", dir,
		"--vrooli-bin", "/Users/test/.vrooli/bin/vrooli",
		"--capabilities", "scenario status*,scenario test*",
		"--presence-only=false",
	}, def.Args)
}

func TestServiceDefinition_ProvisionerIsDistinctService(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{
		StateDir:            dir,
		WorkDir:             "/srv/vrooli",
		VrooliBin:           "/opt/vrooli/bin/vrooli",
		ServiceUser:         "vrooli-provisioner",
		ProvisionHelper:     true,
		ProvisionSocket:     "/run/vrooli-bridge/provision.sock",
		ProvisionClientUID:  1000,
		ProvisionClientHome: "/home/vrooli",
	}
	def, err := serviceDefinition(cfg)
	require.NoError(t, err)
	require.Equal(t, "vrooli-bridge-provisioner", def.Name)
	require.Equal(t, "vrooli-provisioner", def.User)
	require.Equal(t, []string{
		"--state-dir", dir,
		"--provision-helper", "--provision-socket", "/run/vrooli-bridge/provision.sock",
		"--provision-client-uid", "1000",
		"--provision-client-home", "/home/vrooli",
		"--work-dir", "/srv/vrooli",
		"--vrooli-bin", "/opt/vrooli/bin/vrooli",
	}, def.Args)
}
