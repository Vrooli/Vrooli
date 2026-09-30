package provision

import "context"

// Repository is the persistence seam the provision service depends on.
// Production wires the sqlite-backed implementation from sqlite.go; service unit
// tests wire mocks.FakeRepository. It persists the durable op record, its
// append-only event history, and the per-node version history.
type Repository interface {
	// Create persists op as a new provisioning op. The implementation populates
	// ID (when empty) and CreatedAt. Returns the persisted op.
	Create(ctx context.Context, op ProvisioningOp) (ProvisioningOp, error)

	// Get returns the op with the given id or ErrOpNotFound{id}.
	Get(ctx context.Context, id string) (ProvisioningOp, error)

	// List returns ops newest-first by CreatedAt, narrowed by filter.
	List(ctx context.Context, filter ListFilter) ([]ProvisioningOp, error)

	// Update persists the mutable lifecycle columns of op (status,
	// resulting_revision, exit_code, started_at, finished_at) matched by op.ID
	// and returns the stored op. Returns ErrOpNotFound when no row matches.
	Update(ctx context.Context, op ProvisioningOp) (ProvisioningOp, error)

	// AppendEvent appends one event to the op's history. Append-only: there is
	// no update/delete. The event's Sequence is assigned by the caller (the
	// node's monotonic per-op counter).
	AppendEvent(ctx context.Context, ev ProvisionEvent) error

	// ListEvents returns an op's full event history in Sequence order.
	ListEvents(ctx context.Context, opID string) ([]ProvisionEvent, error)

	// UpsertNodeVersion records the node's current revision (replacing any prior
	// row for the node — the control plane keeps the latest known version).
	UpsertNodeVersion(ctx context.Context, v NodeVersion) error

	// GetNodeVersion returns the node's current recorded version or
	// ErrNoNodeVersion{nodeID}.
	GetNodeVersion(ctx context.Context, nodeID string) (NodeVersion, error)
}
