// Package channel is the node-agent's side of the dial-out wire protocol
// (packages/proto/.../v1/channel/channel.proto). Phase 0 ships the contract
// and a stub Dial that constructs the Handshake the agent WILL send and
// reports it, without opening a real network connection — the live SSE dial +
// heartbeat loop + mutual-auth signing land in Phase 1/2. Keeping the proto
// types in use from day one means the wire shape can never silently drift from
// the agent that speaks it.
package channel

import (
	"context"
	"errors"
	"runtime"

	"vrooli-bridge/agent/internal/buildinfo"
	"vrooli-bridge/agent/internal/config"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// ProtocolVersion is the agent's implemented wire protocol version. It MUST
// match CHANNEL_PROTOCOL_VERSION documented in channel.proto; the control
// plane negotiates against it in the handshake and flags a mismatch
// NEEDS_UPDATE rather than silently mis-driving the node.
const ProtocolVersion uint32 = 1

// ErrNotConfigured is returned by Dial when the agent has no control plane to
// dial out to yet (unpaired). It is an expected, non-fatal condition for the
// Phase 0 skeleton.
var ErrNotConfigured = errors.New("agent not configured to dial a control plane (no control-plane URL / node id)")

// Client holds the agent's channel configuration. In later phases it gains the
// HTTP/SSE transport, the Ed25519 signer, and the pinned control-plane key.
type Client struct {
	cfg config.Config
}

// NewClient constructs a channel client from resolved config.
func NewClient(cfg config.Config) *Client {
	return &Client{cfg: cfg}
}

// Handshake builds the Handshake frame the agent presents when it opens the
// channel. It is pure (no I/O) so both the stub Dial and tests can assert on
// it. The node identity, OS/arch, and advertised capabilities come from
// config + the build target; the agent version is the build fingerprint.
func (c *Client) Handshake() *channelv1.Handshake {
	return &channelv1.Handshake{
		ProtocolVersion: ProtocolVersion,
		NodeId:          c.cfg.NodeID,
		AgentVersion:    buildinfo.Fingerprint(),
		Os:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Capabilities:    append([]string(nil), c.cfg.Capabilities...),
	}
}

// Dial opens (in later phases) the persistent dial-out channel and blocks
// holding it. The Phase 0 skeleton validates that the agent is configured and
// returns ErrNotConfigured when it is not; when configured it returns nil
// after constructing the handshake (no network yet). The ctx is honoured by
// the real implementation's connect/retry loop.
func (c *Client) Dial(ctx context.Context) error {
	if !c.cfg.Paired() {
		return ErrNotConfigured
	}
	_ = ctx
	// Phase 1 replaces this stub with: open SSE stream to ControlPlaneURL,
	// send signed Handshake, await HandshakeAck, then run the heartbeat loop.
	_ = c.Handshake()
	return nil
}
