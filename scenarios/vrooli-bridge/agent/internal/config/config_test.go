package config

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad_DefaultsAndStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRIDGE_AGENT_STATE_DIR", dir)

	cfg, err := Load(nil)
	require.NoError(t, err)

	require.Equal(t, dir, cfg.StateDir)
	require.Equal(t, filepath.Join(dir, credentialFileName), cfg.CredentialPath)
	require.Equal(t, filepath.Join(dir, controlPlaneKeyFileName), cfg.ControlPlaneKeyPath)
	require.Equal(t, defaultHeartbeatInterval, cfg.HeartbeatInterval)
	require.False(t, cfg.Paired(), "no URL/node id yet")
	require.Empty(t, cfg.Capabilities)
}

func TestLoad_FlagsOverrideEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "https://env.example")
	t.Setenv("BRIDGE_NODE_ID", "env-node")

	cfg, err := Load([]string{
		"--control-plane-url", "https://flag.example",
		"--node-id", "flag-node",
		"--state-dir", dir,
		"--heartbeat-interval", "5s",
		"--capabilities", "scenario test*, registry list ,",
	})
	require.NoError(t, err)

	require.Equal(t, "https://flag.example", cfg.ControlPlaneURL)
	require.Equal(t, "flag-node", cfg.NodeID)
	require.Equal(t, 5*time.Second, cfg.HeartbeatInterval)
	require.Equal(t, []string{"scenario test*", "registry list"}, cfg.Capabilities)
	require.True(t, cfg.Paired())
}

func TestLoad_EnvFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRIDGE_AGENT_STATE_DIR", dir)
	t.Setenv("BRIDGE_CONTROL_PLANE_URL", "https://env.example")
	t.Setenv("BRIDGE_NODE_ID", "env-node")
	t.Setenv("BRIDGE_HEARTBEAT_INTERVAL", "30s")

	cfg, err := Load(nil)
	require.NoError(t, err)

	require.Equal(t, "https://env.example", cfg.ControlPlaneURL)
	require.Equal(t, "env-node", cfg.NodeID)
	require.Equal(t, 30*time.Second, cfg.HeartbeatInterval)
	require.True(t, cfg.Paired())
}

func TestLoad_DiscoverFlagAndEnv(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load([]string{"--state-dir", dir})
	require.NoError(t, err)
	require.False(t, cfg.Discover, "discovery is off by default")

	cfg, err = Load([]string{"--state-dir", dir, "--discover"})
	require.NoError(t, err)
	require.True(t, cfg.Discover, "the flag enables mDNS discovery")

	t.Setenv("BRIDGE_DISCOVER", "true")
	cfg, err = Load([]string{"--state-dir", dir})
	require.NoError(t, err)
	require.True(t, cfg.Discover, "BRIDGE_DISCOVER enables discovery")
}

func TestLoad_RejectsNonPositiveHeartbeat(t *testing.T) {
	dir := t.TempDir()
	_, err := Load([]string{"--state-dir", dir, "--heartbeat-interval", "0s"})
	require.Error(t, err)
}

func TestLoad_InvalidEnvDurationFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BRIDGE_AGENT_STATE_DIR", dir)
	t.Setenv("BRIDGE_HEARTBEAT_INTERVAL", "not-a-duration")

	cfg, err := Load(nil)
	require.NoError(t, err)
	require.Equal(t, defaultHeartbeatInterval, cfg.HeartbeatInterval)
}
