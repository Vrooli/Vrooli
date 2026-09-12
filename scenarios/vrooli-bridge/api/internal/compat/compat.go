// Package compat is the protocol-compatibility evaluator (OT-P1-001). The
// dial-out channel is versioned (channel.proto CHANNEL_PROTOCOL_VERSION); a node
// reports its agent's wire-protocol version when it connects, and the control
// plane decides whether that version is fully drivable, drivable-for-presence-
// only (the agent needs updating), or not drivable at all. A version-drifted
// node is FLAGGED and excluded from work rather than silently mis-driven — the
// mechanism that keeps a fleet coherent without per-machine hand-maintenance.
//
// The evaluator is a pure function over (node protocol version, control-plane
// current version, control-plane minimum-supported version). The live layer
// (presence.Hub) stores each online node's evaluated Status; dispatch and the
// fleet roll read it to gate work.
package compat

// ProtocolVersion is the control plane's CURRENT channel wire-protocol version.
// It mirrors channel.proto's CHANNEL_PROTOCOL_VERSION; bump both together when
// the wire contract changes incompatibly.
const ProtocolVersion uint32 = 2

// MinSupportedProtocolVersion is the OLDEST agent protocol version the control
// plane can still drive. A node below this is INCOMPATIBLE (its channel is held
// for presence but it receives no work, and the operator is told to update it).
const MinSupportedProtocolVersion uint32 = 1

// Status is the control plane's verdict on a node's protocol version. It mirrors
// channel.CompatibilityStatus so the channel handler can translate at the edge
// without the live layer importing proto.
type Status int

const (
	// StatusUnspecified is the zero value: no version has been negotiated yet.
	// It is not dispatchable because compatibility must be measured before work
	// is sent to a node.
	StatusUnspecified Status = 0

	// StatusOK — the node's protocol version is fully drivable; it may receive
	// jobs and provisioning commands.
	StatusOK Status = 1

	// StatusNeedsUpdate — older than current but still within the supported
	// window: presence is held, but no work is dispatched until the agent is
	// updated.
	StatusNeedsUpdate Status = 2

	// StatusIncompatible — below the minimum supported version: the control
	// plane cannot drive it at all.
	StatusIncompatible Status = 3
)

// String renders the status as a short lowercase label for logs/CLI.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusNeedsUpdate:
		return "needs_update"
	case StatusIncompatible:
		return "incompatible"
	default:
		return "unspecified"
	}
}

// Dispatchable reports whether a node with this status may receive WORK.
// Provisioning is allowed regardless so an unreported or out-of-date node can
// be repaired, but dispatch requires an explicitly measured OK status.
func (s Status) Dispatchable() bool { return s == StatusOK }

// Evaluate classifies a node's reported protocol version against the control
// plane's current + minimum-supported versions.
func Evaluate(nodeProtocolVersion uint32) Status {
	return EvaluateAt(nodeProtocolVersion, ProtocolVersion, MinSupportedProtocolVersion)
}

// EvaluateAt is the pure evaluator with explicit current/min thresholds (the
// testable core). The bands are:
//
//   - nodePV == 0           → Unspecified (not dispatchable until reported)
//   - nodePV >= current     → OK (equal, or newer — DiscardUnknown lets the
//     control plane drive the subset it understands)
//   - min <= nodePV < current → NeedsUpdate (drivable for presence; no work)
//   - nodePV < min          → Incompatible
func EvaluateAt(nodeProtocolVersion, current, minSupported uint32) Status {
	switch {
	case nodeProtocolVersion == 0:
		return StatusUnspecified
	case nodeProtocolVersion >= current:
		return StatusOK
	case nodeProtocolVersion >= minSupported:
		return StatusNeedsUpdate
	default:
		return StatusIncompatible
	}
}
