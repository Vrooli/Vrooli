package fleet

import "context"

// NodeRef is the minimal node shape a roll needs to enumerate + classify: its id,
// whether it is revoked, and whether it was onboarded from a working tree (dirty
// provenance). The handler adapter projects a registry node down to this.
type NodeRef struct {
	ID      string
	Revoked bool
	// WorkingTree is true when the node's provenance revision carries the dirty
	// working-tree marker, so a revision roll excludes it (needs-reprovision).
	WorkingTree bool
}

// NodeLister is the registry enumeration seam: the roll needs every registered
// node (or a resolvable subset) with its revocation state. The handler adapter
// wraps the registry service.
type NodeLister interface {
	// ListNodes returns every registered node. An empty fleet returns an empty
	// slice (not an error).
	ListNodes(ctx context.Context) ([]NodeRef, error)
}

// Presence is the live seam the roll gates on: online state + the
// protocol-compatibility verdict. The presence hub satisfies it.
type Presence interface {
	IsOnline(nodeID string) bool
	// Dispatchable reports online AND protocol-compatible (not flagged). A
	// version-drifted node is excluded from the roll's dispatch with a
	// needs-update disposition.
	Dispatchable(nodeID string) bool
}

// Provisioner is the privileged-provisioning seam: dispatch one node's
// SyncToRevision and return its durable op id. The handler adapter wraps the
// provision service so fleet never reimplements provisioning (or imports the
// provision domain / proto).
type Provisioner interface {
	Provision(ctx context.Context, in ProvisionRequest) (opID string, err error)
}

// RevisionResolver defaults, validates, and preflights the roll's target revision
// ONCE, up front, so the whole fleet pins to one exact commit: an empty or "@cp"
// target resolves to the control plane's current commit, and an unpushed/invalid
// target fails the roll loudly before any node is dispatched. Resolving once
// (rather than per node) is what makes a roll atomic — a push landing mid-roll
// cannot split the fleet across two commits. Production wires api/internal/cprev;
// unit tests fake it or leave it unset (legacy: require a non-empty target, no
// @cp, no preflight).
type RevisionResolver interface {
	Resolve(ctx context.Context, requested string) (string, error)
}

// ProvisionRequest is the fleet-local DTO for one node's provisioning dispatch.
type ProvisionRequest struct {
	Actor          string
	NodeID         string
	TargetRevision string
	TimeoutSeconds int64
}
