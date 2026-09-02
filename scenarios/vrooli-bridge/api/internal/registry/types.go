// Package registry is the domain-scoped home for trusted-node identity
// (OT-P0-001). It owns the durable `nodes` record: a node's stable id, OS/arch,
// provisioned revision, reachable endpoint, self-reported capabilities, and the
// registry-owned execution scopes that authorize what it may be asked to run.
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
	ID   string
	Name string
	Kind string
	OS   string
	Arch string
	// MachineArch is the physical/OS-reported architecture (for example arm64
	// on an Apple Silicon host). Arch remains the legacy wire field.
	MachineArch string
	// BinaryArch is the architecture of the running agent executable. It can
	// differ from MachineArch when a translation layer is in use.
	BinaryArch   string
	Revision     string
	Endpoint     string
	Capabilities []string
	Scopes       []string
	// PairingCorrelationID is an opaque enrollment correlation written only by
	// the pairing saga. It makes a post-registration crash recoverable without
	// inferring identity from a hostname, IP, or display name.
	PairingCorrelationID string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	// LastSeenAt is the last time a heartbeat was received; zero if never seen.
	// Persisted so "last seen 2h ago" survives a control-plane restart.
	LastSeenAt time.Time
	// RevokedAt is set when the node is revoked; zero on active nodes.
	RevokedAt           time.Time
	CapabilityInventory []CapabilityObservation
	CapabilityProbedAt  time.Time
	ConfigurationOpID   string
	ConfigurationState  string
	ConfigurationAt     time.Time
	ConfigurationUnmet  []string
}

type CapabilityObservation struct {
	Capability string    `json:"capability"`
	ID         string    `json:"id"`
	Label      string    `json:"label"`
	State      string    `json:"state"`
	Path       string    `json:"path,omitempty"`
	Version    string    `json:"version,omitempty"`
	ProbedAt   time.Time `json:"probed_at"`
	Detail     string    `json:"detail,omitempty"`
}

// Revoked reports whether the node has been revoked. A revoked node is always
// surfaced as REVOKED regardless of any lingering channel.
func (n Node) Revoked() bool { return !n.RevokedAt.IsZero() }

// RegisterInput is the explicit DTO Service.Register accepts. Distinct from
// Node so callers cannot pass an id/timestamp the service has no way to honour.
type RegisterInput struct {
	Name                 string
	Kind                 string
	OS                   string
	Arch                 string
	MachineArch          string
	BinaryArch           string
	Endpoint             string
	Capabilities         []string
	Scopes               []string
	PairingCorrelationID string
}

// UpdateInput is the desired post-state of a node's owner-editable surface.
type UpdateInput struct {
	ID                 string
	Name               string
	Endpoint           string
	Capabilities       []string
	Scopes             []string
	Revision           string
	Kind               string
	ConfigurationOpID  string
	ConfigurationState string
	ConfigurationAt    time.Time
	ConfigurationUnmet []string
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

// ErrInvalidGrant is the typed refusal returned when an owner attempts to
// persist a node grant outside the derived catalog vocabulary.
type ErrInvalidGrant struct {
	Scope  string
	Reason string
}

func (e ErrInvalidGrant) Error() string {
	return fmt.Sprintf("scope %q: %s", e.Scope, e.Reason)
}

type ErrNodeActive struct{ ID string }

func (e ErrNodeActive) Error() string {
	return fmt.Sprintf("node %q must be revoked before removal", e.ID)
}
