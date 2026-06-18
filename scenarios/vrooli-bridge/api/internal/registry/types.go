// Package registry is the domain-scoped home for trusted-node identity
// (OT-P0-001). It owns the durable `nodes` record: a node's stable id, OS/arch,
// provisioned revision, reachable endpoint, self-reported capabilities, and the
// granted verb-namespace scopes that authorize what it may be asked to run.
//
// Layering mirrors the canonical Vrooli domain pattern (see the device-sync-hub
// devices domain):
//
//	HTTP → handler → Service (validates, applies defaults) → Repository (persists)
//	                     ↑                                       ↑
//	                     FakeService (handler tests)              FakeRepository (service tests)
//	                                                              Real sqlite (repository tests)
//
// Presence (online/offline) is NOT persisted here — it is an ephemeral overlay
// the presence hub layers onto the read path at the handler. The repository is
// the source of truth only for durable identity + last-seen + revocation.
package registry

import (
	"fmt"
	"time"
)

// Node is the internal domain shape for a trusted node. Distinct from the
// proto wire type at packages/proto/.../v1/registry.Node — the handler
// translates at the boundary so the domain never imports proto.
type Node struct {
	ID           string
	Name         string
	OS           string
	Arch         string
	Revision     string
	Endpoint     string
	Capabilities []string
	Scopes       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	// LastSeenAt is the last time a heartbeat was received; zero if never seen.
	// Persisted so "last seen 2h ago" survives a control-plane restart.
	LastSeenAt time.Time
	// RevokedAt is set when the node is revoked; zero on active nodes.
	RevokedAt time.Time
}

// Revoked reports whether the node has been revoked. A revoked node is always
// surfaced as REVOKED regardless of any lingering channel.
func (n Node) Revoked() bool { return !n.RevokedAt.IsZero() }

// RegisterInput is the explicit DTO Service.Register accepts. Distinct from
// Node so callers cannot pass an id/timestamp the service has no way to honour.
type RegisterInput struct {
	Name         string
	OS           string
	Arch         string
	Endpoint     string
	Capabilities []string
	Scopes       []string
}

// UpdateInput is the desired post-state of a node's owner-editable surface.
type UpdateInput struct {
	ID           string
	Name         string
	Endpoint     string
	Capabilities []string
	Scopes       []string
	Revision     string
}

// ErrNodeNotFound is the typed sentinel returned when no row matches an id.
// Handlers translate it into a Connect NotFound.
type ErrNodeNotFound struct {
	ID string
}

func (e ErrNodeNotFound) Error() string {
	return fmt.Sprintf("node %q not found", e.ID)
}

// ErrInvalidNode is the typed sentinel returned on validation failure. Handlers
// translate it into a Connect InvalidArgument carrying "<field>: <reason>".
type ErrInvalidNode struct {
	Field  string
	Reason string
}

func (e ErrInvalidNode) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
