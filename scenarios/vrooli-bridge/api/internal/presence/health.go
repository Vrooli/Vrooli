// Package presence manages each node's dial-out channel and the control plane's
// view of which nodes are online plus their self-reported readiness
// (OT-P0-003). Presence is ephemeral, in-memory, single-instance state —
// rebuilt on reconnect, never persisted (the registry persists durable
// identity; the registry handler overlays this live state at read time).
//
// The Hub is the in-memory core (hub.go). The dial-out SSE edge and the
// heartbeat intake live in handlers/channel/. An optional Redis-backed Hub for
// scale-out is a declared future seam (device-sync-hub's pattern), not built
// while the fleet is single-instance.
package presence

import "time"

// HealthSnapshot is the node's self-reported readiness, mirroring the
// channel.HealthSnapshot wire type. The domain layer keeps its own shape so it
// never imports proto; the channel handler translates at the boundary.
type HealthSnapshot struct {
	// ToolchainPresent reports the `vrooli` CLI + setup are runnable.
	ToolchainPresent bool
	// DiskHeadroomBytes is free space on the work volume.
	DiskHeadroomBytes int64
	// ContainerRuntimeUp reports a container runtime some jobs need is up.
	ContainerRuntimeUp bool
	// Details carries free-form extra readiness facts (e.g. "go" -> "1.25.0").
	Details map[string]string
	// ReportedAt is when the agent sampled the snapshot.
	ReportedAt time.Time
}

// Ready reports whether the node is in a state dispatch can target: the
// toolchain must be present. Disk/container checks are job-specific and left to
// the dispatch policy (Phase 3); this is the baseline gate.
func (h HealthSnapshot) Ready() bool { return h.ToolchainPresent }

// NodePresence is a point-in-time view of one node's live state, used by the
// presence snapshot and (later) the SSE presence event.
type NodePresence struct {
	NodeID string
	Online bool
	Health HealthSnapshot
	// HasHealth is false when the node is online but has not yet sent its first
	// heartbeat (so a zero-value Health is not mistaken for "all false").
	HasHealth bool
}
