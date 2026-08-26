// Package pairing owns one-touch bootstrap and mutual-auth enrollment
// (OT-P0-002): single-use short-TTL pairing codes, the request/approve
// fallback, and the per-node Ed25519 credentials the control plane verifies
// node calls against (the node keeps the private key; this stores the public
// key). It is the write side of mutual auth; internal/nodeauth is the read
// side (verification), and internal/cpkeys is the control plane's own identity.
//
// Layering mirrors the registry domain:
//
//	HTTP → handler → Service (codes, mint, approve) → Repository (persists)
//	                     ↑                                ↑
//	                     FakeService (handler tests)       FakeRepository (service tests)
//	                                                       Real sqlite (repository tests)
//
// The service depends on a NodeRegistrar seam to create the durable node record
// (owned by the registry domain) so pairing never imports registry types.
package pairing

import (
	"context"
	"fmt"
	"time"
)

// PairingCode is the stored (hashed) bootstrap token. The plaintext is returned
// only once, by Service.IssueCode; it is never persisted or re-derivable.
type PairingCode struct {
	ID             string
	CodeHash       string
	Name           string
	Scopes         []string
	CorrelationID  string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	ClaimedAt      time.Time
	RedeemedAt     time.Time
	RedeemedNodeID string
}

// Redeemed reports whether the code has already been burned.
func (c PairingCode) Redeemed() bool { return !c.RedeemedAt.IsZero() }

// Credential is a node's stored Ed25519 public key (standard base64).
type Credential struct {
	NodeID    string
	PublicKey string
	CreatedAt time.Time
	RevokedAt time.Time
}

type EncryptionKey struct {
	NodeID    string
	PublicKey string
	Algorithm string
	CreatedAt time.Time
	RevokedAt time.Time
}

// Revoked reports whether the credential has been severed.
func (c Credential) Revoked() bool { return !c.RevokedAt.IsZero() }

// RequestStatus is the lifecycle of a request/approve enrollment.
type RequestStatus string

const (
	RequestPending  RequestStatus = "pending"
	RequestApproved RequestStatus = "approved"
	RequestRejected RequestStatus = "rejected"
)

// PairingRequest is a pending/decided join request (the no-code fallback).
type PairingRequest struct {
	ID           string
	PublicKey    string
	Name         string
	OS           string
	Arch         string
	Endpoint     string
	Capabilities []string
	Status       RequestStatus
	NodeID       string
	CreatedAt    time.Time
	DecidedAt    time.Time
}

// NodeFacts is the node-self-reported identity a redeem/approve registers. It
// is the decoupled DTO the NodeRegistrar seam accepts so pairing does not import
// registry types.
type NodeFacts struct {
	Name         string
	OS           string
	Arch         string
	Endpoint     string
	Capabilities []string
	Scopes       []string
}

// NodeRegistrar is the seam pairing creates durable node records through (owned
// by the registry domain). main.go wires an adapter over registry.Service.
type NodeRegistrar interface {
	RegisterNode(ctx context.Context, facts NodeFacts) (nodeID string, err error)
}

// CorrelatedNodeRegistrar extends NodeRegistrar for the durable enrollment
// path. The Registry owns the stored correlation; pairing only uses it to
// reconcile a crash without identity inference.
type CorrelatedNodeRegistrar interface {
	NodeRegistrar
	RegisterNodeWithCorrelation(context.Context, NodeFacts, string) (string, error)
	FindNodeByPairingCorrelation(context.Context, string) (string, error)
}

// CorrelatedNodeScopeUpdater is the optional registry seam used when a fresh,
// owner-issued correlated enrollment reuses an existing node identity. The
// pairing code is the authority for the new grant; the node's self-report is
// not allowed to preserve stale execution scopes across re-enrollment.
type CorrelatedNodeScopeUpdater interface {
	UpdateNodeScopes(context.Context, string, []string) error
}

type EnrollmentSaga struct {
	CorrelationID string
	CodeID        string
	PublicKey     string
	Facts         NodeFacts
	State         string
	NodeID        string
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   time.Time
}

// Typed sentinels — handlers translate these to Connect codes.

// ErrCodeNotFound: no code matched the presented value (Connect NotFound /
// Unauthenticated at the boundary — an unknown code must not reveal which part
// was wrong).
var ErrCodeNotFound = fmt.Errorf("pairing code not found")

// ErrCodeExpired: the code's TTL elapsed.
var ErrCodeExpired = fmt.Errorf("pairing code expired")

// ErrCodeUsed: the code was already redeemed (single-use).
var ErrCodeUsed = fmt.Errorf("pairing code already redeemed")

// ErrRequestNotFound: no pairing request matched the id.
var ErrRequestNotFound = fmt.Errorf("pairing request not found")

// ErrRequestDecided: the request was already approved/rejected.
var ErrRequestDecided = fmt.Errorf("pairing request already decided")

// ErrInvalid is a validation failure carrying the offending field.
type ErrInvalid struct {
	Field  string
	Reason string
}

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
