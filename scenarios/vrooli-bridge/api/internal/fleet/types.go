// Package fleet is the domain-scoped home for fleet-wide version rolls
// (OT-P1-001): pinning the WHOLE fleet (or a named subset) to one target git
// revision in a single operation instead of hand-running `provision sync` per
// node. A roll enumerates the eligible nodes, classifies each (online +
// protocol-compatible + not-revoked → dispatch; otherwise skip with a reason),
// delegates the actual privileged provisioning to the provision domain — fleet
// NEVER reimplements provisioning — and records one durable Rollout with a
// per-node ledger so the operator sees, at a glance, which nodes were dispatched
// and which were skipped (and why).
//
// Protocol-compatibility gating keeps the fleet coherent: a node flagged
// needs-update / incompatible by the channel handshake is recorded as a SKIPPED
// result, not silently mis-driven. Each dispatched node gets a provisioning op
// id the operator blocks on with `provision wait` — the rollout is the fan-out +
// the per-node ledger, not a second op-lifecycle engine.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs, so the domain imports no sibling domain and no proto: the
// handler module is the single translation point to the registry (enumerate +
// revocation), the presence hub (online + protocol compatibility), and the
// provision service (dispatch a privileged op).
package fleet

import (
	"fmt"
	"strings"
	"time"
)

// RolloutStatus is a roll's aggregate disposition, derived from its per-node
// results at dispatch time.
type RolloutStatus int

const (
	// StatusUnspecified is the zero value; a persisted rollout never holds it.
	StatusUnspecified RolloutStatus = 0
	// StatusDispatched — every eligible node was dispatched a provisioning op
	// (none skipped, none failed to dispatch).
	StatusDispatched RolloutStatus = 1
	// StatusPartial — some nodes were dispatched and some were skipped or failed.
	StatusPartial RolloutStatus = 2
	// StatusFailed — no node was dispatched (all skipped/failed); nothing rolled.
	StatusFailed RolloutStatus = 3
)

// String renders the rollout status as a short lowercase label.
func (s RolloutStatus) String() string {
	switch s {
	case StatusDispatched:
		return "dispatched"
	case StatusPartial:
		return "partial"
	case StatusFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// NodeDisposition is the per-node outcome of a roll at dispatch time.
type NodeDisposition int

const (
	// DispositionUnspecified is the zero value.
	DispositionUnspecified NodeDisposition = 0
	// DispositionDispatched — a provisioning op was created and pushed (OpID set).
	DispositionDispatched NodeDisposition = 1
	// DispositionSkippedOffline — the node holds no dial-out channel.
	DispositionSkippedOffline NodeDisposition = 2
	// DispositionSkippedNeedsUpdate — the node's agent protocol is flagged.
	DispositionSkippedNeedsUpdate NodeDisposition = 3
	// DispositionSkippedRevoked — the node is revoked.
	DispositionSkippedRevoked NodeDisposition = 4
	// DispositionFailed — the provisioning op could not be dispatched.
	DispositionFailed NodeDisposition = 5
	// DispositionSkippedWorkingTree — the node was onboarded from the control
	// plane's working tree (dirty provenance), so it is pinned to no fetchable
	// commit and a revision roll cannot converge it. It is excluded and flagged
	// needs-reprovision; re-onboard it pinned to make it rollable.
	DispositionSkippedWorkingTree NodeDisposition = 6
)

// String renders the disposition as a short lowercase label.
func (d NodeDisposition) String() string {
	switch d {
	case DispositionDispatched:
		return "dispatched"
	case DispositionSkippedOffline:
		return "skipped_offline"
	case DispositionSkippedNeedsUpdate:
		return "skipped_needs_update"
	case DispositionSkippedRevoked:
		return "skipped_revoked"
	case DispositionSkippedWorkingTree:
		return "skipped_working_tree"
	case DispositionFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// dispatched reports whether the disposition counts as a successful dispatch.
func (d NodeDisposition) dispatched() bool { return d == DispositionDispatched }

// NodeResult is one node's line in a rollout's ledger.
type NodeResult struct {
	NodeID      string
	Disposition NodeDisposition
	// OpID is set when Disposition == DispositionDispatched.
	OpID string
	// Detail carries the skip reason or the dispatch error.
	Detail string
}

// Rollout is the durable, server-owned record of one RollFleet.
type Rollout struct {
	ID             string
	TargetRevision string
	Status         RolloutStatus
	TotalNodes     int
	Dispatched     int
	Skipped        int
	Failed         int
	CreatedAt      time.Time
}

// RollInput is what Service.Roll accepts.
type RollInput struct {
	Actor          string
	TargetRevision string
	// NodeIDs optionally narrows the roll to a subset; empty rolls every
	// registered, non-revoked node.
	NodeIDs        []string
	TimeoutSeconds int64
	DryRun         bool
}

// RollDecision is the result of a Roll: the persisted rollout id (empty on a
// dry-run), whether it was a dry-run, the aggregate status, and the per-node
// ledger.
type RollDecision struct {
	RolloutID string
	DryRun    bool
	Status    RolloutStatus
	Results   []NodeResult
}

// ListFilter narrows ListRollouts.
type ListFilter struct {
	Limit int
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrInvalidRoll — a structural validation failure (empty required field).
type ErrInvalidRoll struct {
	Field  string
	Reason string
}

func (e ErrInvalidRoll) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrRolloutNotFound — no rollout matches the id.
type ErrRolloutNotFound struct{ ID string }

func (e ErrRolloutNotFound) Error() string { return fmt.Sprintf("rollout %q not found", e.ID) }

// aggregateStatus derives a rollout's status from its per-node results.
func aggregateStatus(results []NodeResult) RolloutStatus {
	dispatched, other := 0, 0
	for _, r := range results {
		if r.Disposition.dispatched() {
			dispatched++
		} else {
			other++
		}
	}
	switch {
	case dispatched == 0:
		return StatusFailed
	case other == 0:
		return StatusDispatched
	default:
		return StatusPartial
	}
}

// trimRevision normalises a revision token.
func trimRevision(s string) string { return strings.TrimSpace(s) }
