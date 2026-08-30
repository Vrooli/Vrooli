package registry

import (
	"context"
	"time"
)

// Repository is the persistence seam the registry service depends on.
// Production wires the sqlite-backed implementation from sqlite.go; service
// unit tests wire mocks.FakeRepository. Keep the surface narrow — new methods
// land here only when the service proves it needs them.
type Repository interface {
	// Create persists n. The implementation populates ID (when empty),
	// CreatedAt, and UpdatedAt. Returns the persisted Node.
	Create(ctx context.Context, n Node) (Node, error)

	// Get returns the node with the given id or ErrNodeNotFound{id}.
	Get(ctx context.Context, id string) (Node, error)

	// GetByPairingCorrelation returns the Node created by a durable pairing
	// correlation, if any. It is used only by reconciliation.
	GetByPairingCorrelation(ctx context.Context, correlationID string) (Node, error)

	// List returns all nodes ordered newest-first by CreatedAt (including
	// revoked nodes — the read path decides how to present them).
	List(ctx context.Context) ([]Node, error)

	// Update persists the owner-editable fields of n (matched by n.ID) and
	// bumps UpdatedAt. Returns ErrNodeNotFound when no row matches.
	Update(ctx context.Context, n Node) (Node, error)
	// UpdateArchitecture records the agent's self-reported machine and binary
	// architectures without allowing the node to edit owner-controlled fields.
	UpdateArchitecture(ctx context.Context, id, machineArch, binaryArch string) error

	// Revoke stamps revoked_at on the node (idempotent: a second revoke is a
	// no-op that returns the already-revoked record). Returns ErrNodeNotFound
	// when no row matches. Atomic credential destruction is layered on in
	// Phase 2; this is the durable lifecycle half.
	Revoke(ctx context.Context, id string) (Node, error)
	Remove(ctx context.Context, id string) error

	// TouchLastSeen records that a heartbeat was received from the node at t.
	// A no-op (nil) when no row matches — a heartbeat from an unknown/removed
	// node should not error the presence path.
	TouchLastSeen(ctx context.Context, id string, t time.Time) error
}

const (
	KindAgent        = "agent"
	KindSSH          = "ssh"
	KindAttached     = "attached"
	KindControlPlane = "control_plane"
)

func ValidKind(kind string) bool {
	return kind == KindAgent || kind == KindSSH || kind == KindAttached || kind == KindControlPlane
}

// PushTargets is the single structural target gate for control-plane initiated
// delivery. The source host is never a recipient, even when its API is ready.
func PushTargets(nodes []Node) []Node {
	result := make([]Node, 0, len(nodes))
	for _, node := range nodes {
		if node.Revoked() || node.Kind == KindControlPlane {
			continue
		}
		result = append(result, node)
	}
	return result
}
