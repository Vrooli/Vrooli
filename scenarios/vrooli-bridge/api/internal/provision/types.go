// Package provision is the domain-scoped home for the PRIVILEGED provisioning
// tier (OT-P0-006): bringing a fleet node to a target project git revision R
// (`git fetch`@R + an idempotent `vrooli setup`) and recording the node's
// resulting version. It is the structural counterpart to dispatch/runs — where
// a job runs as the non-privileged runner under an allowlisted verb, a
// provisioning op runs through a separate privileged helper on the node
// (DECISIONS.md "two trust tiers"). The everyday runner has no path to invoke
// provisioning, so remote execution can never escalate.
//
// This one domain owns BOTH the SyncToRevision orchestration (resolve node →
// validate → audit → create durable op → push the privileged ProvisionCommand)
// AND the durable op lifecycle (block-once Wait, live Subscribe fan-out,
// node-event ingest that drives status transitions and records node versions) —
// unlike dispatch, which delegates run management to the runs domain. The op is
// server-owned and survives client AND node disconnect; the node's helper
// reports progress against it.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs, so the domain imports no sibling domain and no proto: the
// handler module is the single translation point to the registry (scopes/
// revocation), the presence hub (online + the channel push), and the audit
// store. The persisted op + event history is the durable source of truth a
// re-attaching client reads; the block-once waiter/subscriber coordination is
// ephemeral (mirrors the runs domain).
package provision

import (
	"fmt"
	"strings"
	"time"
)

// ProvisioningStatus is a provisioning op's lifecycle state. QUEUED/RUNNING are
// non-terminal; COMPLETED/FAILED/ROLLED_BACK are terminal.
type ProvisioningStatus int

const (
	// StatusUnspecified is the zero value; a persisted op never holds it.
	StatusUnspecified ProvisioningStatus = 0
	// StatusQueued — the op record exists and the ProvisionCommand has been
	// delivered, but the node has not yet reported it started.
	StatusQueued ProvisioningStatus = 1
	// StatusRunning — the node's privileged helper is fetching/setting up.
	StatusRunning ProvisioningStatus = 2
	// StatusCompleted — terminal: the node reached the target revision and
	// `vrooli setup` succeeded.
	StatusCompleted ProvisioningStatus = 3
	// StatusFailed — terminal: setup failed and the node could NOT be rolled
	// back (degraded; needs operator attention).
	StatusFailed ProvisioningStatus = 4
	// StatusRolledBack — terminal: setup failed but the node rolled back to its
	// prior revision and re-validated (the safe failure outcome).
	StatusRolledBack ProvisioningStatus = 5
)

// Terminal reports whether the status is a terminal one. Wait returns once an op
// reaches a terminal status; a late event for a terminal op is ignored
// (stale-completion safety).
func (s ProvisioningStatus) Terminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusRolledBack:
		return true
	default:
		return false
	}
}

// String renders the status as a short lowercase label for logs/CLI.
func (s ProvisioningStatus) String() string {
	switch s {
	case StatusQueued:
		return "queued"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusRolledBack:
		return "rolled_back"
	default:
		return "unspecified"
	}
}

// EventKind discriminates a ProvisionEvent's payload. It mirrors the
// provision.proto ProvisionEventKind so the domain never imports proto; the
// handler translates at the boundary.
type EventKind int

const (
	// EventUnspecified is the zero value.
	EventUnspecified EventKind = 0
	// EventLog carries a chunk of combined stdout/stderr.
	EventLog EventKind = 1
	// EventStatus carries a human-readable lifecycle transition label.
	EventStatus EventKind = 2
	// EventVersion reports the node's resulting checked-out revision.
	EventVersion EventKind = 3
	// EventExit carries the terminal helper exit code.
	EventExit EventKind = 4
)

// ProvisioningOp is the durable, server-owned record of one SyncToRevision.
type ProvisioningOp struct {
	ID               string
	NodeID           string
	TargetRevision   string
	RollbackRevision string
	Status           ProvisioningStatus
	// ResultingRevision is the node's checked-out revision once known (a VERSION
	// event). On success it equals TargetRevision; on rollback, RollbackRevision.
	ResultingRevision string
	ExitCode          int32
	TimeoutSeconds    int64
	CreatedAt         time.Time
	// StartedAt/FinishedAt are zero until the corresponding transition.
	StartedAt  time.Time
	FinishedAt time.Time
}

// ProvisionEvent is one entry in an op's append-only event history.
type ProvisionEvent struct {
	OpID      string
	Kind      EventKind
	Sequence  uint64
	LogChunk  string
	Status    string
	Revision  string
	ExitCode  int32
	EmittedAt time.Time
}

// NodeVersion is the control plane's record of a node's current project
// revision, updated each time a provisioning op reports a VERSION event.
type NodeVersion struct {
	NodeID     string
	Revision   string
	OpID       string
	ReportedAt time.Time
}

// TargetNode is the minimal node shape provisioning needs: its id and whether it
// is revoked. The handler adapter projects a registry node down to this.
type TargetNode struct {
	ID      string
	Kind    string
	Revoked bool
}

type ErrUnsupportedNodeKind struct{ ID, Kind string }

func (e ErrUnsupportedNodeKind) Error() string {
	return fmt.Sprintf("node %q of kind %q cannot receive privileged provisioning", e.ID, e.Kind)
}

// SyncInput is what Service.Sync accepts: the owner actor (for audit), the
// target node + revision, an optional explicit rollback revision, the timeout,
// and whether this is a dry-run (X-Dry-Run).
type SyncInput struct {
	Actor            string
	NodeID           string
	TargetRevision   string
	RollbackRevision string
	TimeoutSeconds   int64
	DryRun           bool
}

// Decision is the result of a Sync: the created op id (empty on a dry-run),
// whether it was a dry-run, and the validated request echoed back.
type Decision struct {
	OpID             string
	DryRun           bool
	NodeID           string
	TargetRevision   string
	RollbackRevision string
}

// ListFilter narrows ListOps. Zero-value fields are not applied.
type ListFilter struct {
	NodeID string
	Limit  int
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrOpNotFound — no op matches the id.
type ErrOpNotFound struct{ ID string }

func (e ErrOpNotFound) Error() string { return fmt.Sprintf("provisioning op %q not found", e.ID) }

// ErrNoNodeVersion — the node has never been provisioned (no version recorded).
type ErrNoNodeVersion struct{ NodeID string }

func (e ErrNoNodeVersion) Error() string {
	return fmt.Sprintf("node %q has no recorded version", e.NodeID)
}

// ErrInvalidOp — a structural validation failure (empty required field).
type ErrInvalidOp struct {
	Field  string
	Reason string
}

func (e ErrInvalidOp) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrNodeNotFound — the target node id is unknown.
type ErrNodeNotFound struct{ ID string }

func (e ErrNodeNotFound) Error() string { return fmt.Sprintf("node %q not found", e.ID) }

// ErrNodeRevoked — the target node has been revoked; it can be provisioned no
// further.
type ErrNodeRevoked struct{ ID string }

func (e ErrNodeRevoked) Error() string { return fmt.Sprintf("node %q is revoked", e.ID) }

// ErrNodeOffline — the target node holds no dial-out channel, so the privileged
// command cannot be delivered.
type ErrNodeOffline struct{ ID string }

func (e ErrNodeOffline) Error() string {
	return fmt.Sprintf("node %q is offline (no dial-out channel)", e.ID)
}

// ErrDeliveryFailed — the ProvisionCommand could not be pushed to the (briefly)
// reachable node; the op is marked failed and the request fails.
type ErrDeliveryFailed struct{ NodeID string }

func (e ErrDeliveryFailed) Error() string {
	return fmt.Sprintf("provisioning command could not be delivered to node %q", e.NodeID)
}

// trimRevision normalises a revision token.
func trimRevision(s string) string { return strings.TrimSpace(s) }
