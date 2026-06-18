package fleet

import "context"

// NodeRef is the minimal node shape a roll needs to enumerate + classify: its id
// and whether it is revoked. The handler adapter projects a registry node down
// to this.
type NodeRef struct {
	ID      string
	Revoked bool
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

// ProvisionRequest is the fleet-local DTO for one node's provisioning dispatch.
type ProvisionRequest struct {
	Actor          string
	NodeID         string
	TargetRevision string
	TimeoutSeconds int64
}
