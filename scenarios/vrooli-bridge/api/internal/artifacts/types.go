// Package artifacts is the domain-scoped home for non-git artifact distribution
// (OT-P1-003): shipping a built desktop installer or a large test fixture to a
// fleet node. Bridge does NOT move the bytes — it delegates to device-sync-hub's
// directed delivery via the DirectedDelivery seam and records a durable
// Distribution that tracks the delivery's reference + status. "Bridge
// orchestrates, device-sync-hub moves the bytes." The artifact bytes never
// transit bridge's SQLite store (DATA.md); only the reference + metadata do.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs, so the domain imports no sibling domain and no proto: the
// handler module is the single translation point to the registry (revocation)
// and the device-sync-hub directed-delivery client.
package artifacts

import (
	"fmt"
	"strings"
	"time"
)

// DeliveryStatus is a distribution's lifecycle state.
type DeliveryStatus int

const (
	// StatusUnspecified is the zero value; a persisted distribution never holds it.
	StatusUnspecified DeliveryStatus = 0
	// StatusPending — the directed delivery was accepted by device-sync-hub and
	// is in flight; the node has not yet confirmed receipt.
	StatusPending DeliveryStatus = 1
	// StatusDelivered — terminal: device-sync-hub reports the artifact reached
	// the node and is available to the job at destination_path.
	StatusDelivered DeliveryStatus = 2
	// StatusFailed — terminal: the directed delivery failed.
	StatusFailed DeliveryStatus = 3
)

// String renders the status as a short lowercase label.
func (s DeliveryStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusDelivered:
		return "delivered"
	case StatusFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// Distribution is the durable, server-owned record of one artifact delivery. It
// holds only the reference + metadata — never the bytes.
type Distribution struct {
	ID              string
	NodeID          string
	Name            string
	SourceRef       string
	DestinationPath string
	Status          DeliveryStatus
	DeliveryRef     string
	Detail          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TargetNode is the minimal node shape distribution needs: its id and whether it
// is revoked.
type TargetNode struct {
	ID      string
	Revoked bool
}

// DistributeInput is what Service.Distribute accepts.
type DistributeInput struct {
	Actor           string
	NodeID          string
	Name            string
	SourceRef       string
	DestinationPath string
	DryRun          bool
}

// Decision is the result of a Distribute.
type Decision struct {
	DistributionID string
	DryRun         bool
	Status         DeliveryStatus
	DeliveryRef    string
}

// ListFilter narrows ListDistributions.
type ListFilter struct {
	NodeID string
	Limit  int
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrInvalidDistribution — a structural validation failure (empty required field).
type ErrInvalidDistribution struct {
	Field  string
	Reason string
}

func (e ErrInvalidDistribution) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrDistributionNotFound — no distribution matches the id.
type ErrDistributionNotFound struct{ ID string }

func (e ErrDistributionNotFound) Error() string {
	return fmt.Sprintf("distribution %q not found", e.ID)
}

// ErrNodeNotFound — the target node id is unknown.
type ErrNodeNotFound struct{ ID string }

func (e ErrNodeNotFound) Error() string { return fmt.Sprintf("node %q not found", e.ID) }

// ErrNodeRevoked — the target node has been revoked; nothing can be delivered.
type ErrNodeRevoked struct{ ID string }

func (e ErrNodeRevoked) Error() string { return fmt.Sprintf("node %q is revoked", e.ID) }

// ErrDeliveryFailed — device-sync-hub could not accept the directed delivery.
type ErrDeliveryFailed struct{ NodeID string }

func (e ErrDeliveryFailed) Error() string {
	return fmt.Sprintf("directed delivery to node %q failed", e.NodeID)
}

func trim(s string) string { return strings.TrimSpace(s) }
