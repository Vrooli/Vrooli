package channel

import (
	"context"
	"runtime"
	"testing"

	"vrooli-bridge/agent/internal/config"

	"github.com/stretchr/testify/require"
)

func TestHandshake_ReflectsConfigAndBuildTarget(t *testing.T) {
	cfg := config.Config{
		NodeID:       "node-1",
		Capabilities: []string{"scenario test*"},
	}
	hs := NewClient(cfg).Handshake()

	require.Equal(t, ProtocolVersion, hs.GetProtocolVersion())
	require.Equal(t, "node-1", hs.GetNodeId())
	require.Equal(t, runtime.GOOS, hs.GetOs())
	require.Equal(t, runtime.GOARCH, hs.GetArch())
	require.Equal(t, []string{"scenario test*"}, hs.GetCapabilities())
	require.NotEmpty(t, hs.GetAgentVersion(), "agent version is the build fingerprint")
}

func TestDial_UnpairedReturnsNotConfigured(t *testing.T) {
	err := NewClient(config.Config{}).Dial(context.Background())
	require.ErrorIs(t, err, ErrNotConfigured)
}

func TestDial_PairedStubSucceeds(t *testing.T) {
	cfg := config.Config{ControlPlaneURL: "https://cp.example", NodeID: "node-1"}
	require.NoError(t, NewClient(cfg).Dial(context.Background()))
}

// TestProtocolVersionMatchesProto guards the one invariant that would silently
// mis-drive a node: the agent's implemented ProtocolVersion must equal the
// CHANNEL_PROTOCOL_VERSION the proto documents (1).
func TestProtocolVersionMatchesProto(t *testing.T) {
	require.Equal(t, uint32(1), ProtocolVersion)
}
