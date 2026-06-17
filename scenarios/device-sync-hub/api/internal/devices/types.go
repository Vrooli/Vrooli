// Package devices is the trust boundary of Device Sync Hub: it owns the device
// registry, the pairing handshake (short-TTL single-use code/QR primary path +
// request→approve fallback), the trust group, and device revocation.
//
// Layering mirrors the canonical Vrooli pattern:
//
//	HTTP → handler → Service (validation, pairing policy, token issuance)
//	                     ↓
//	                 Repository (persists devices + pairing_codes)
//
// Owner identity, JWTs, and authenticator sessions are NOT owned here — the
// auth package (internal/auth) delegates those to scenario-authenticator. This
// package keys every device to an OwnerID resolved from a validated owner
// token, and asks auth to revoke a device's authenticator session on un-pair.
//
// Device trust is carried by a hub-issued device token, distinct from the
// owner JWT: the raw token is returned exactly once (at pair/redeem time) and
// only its SHA-256 hash is persisted. Trust enforcement (Phase 3 transfer)
// resolves a presented token to a TRUSTED device via the repository.
package devices

import (
	"fmt"
	"time"
)

// TrustState is a device's position in the trust lifecycle. The string values
// are the persisted form (devices.trust_state column) and map 1:1 to the proto
// TrustState enum at the handler edge.
type TrustState string

const (
	// TrustPending — created via the request→approve fallback, awaiting the
	// owner's approval. Its token is inert until promoted.
	TrustPending TrustState = "pending"
	// TrustTrusted — in the trust group; full access.
	TrustTrusted TrustState = "trusted"
	// TrustRevoked — access severed; token rejected, excluded from broadcast.
	TrustRevoked TrustState = "revoked"
)

// Device is the internal domain shape for a registered device. Distinct from
// the proto wire type — the handler translates at the boundary so this layer
// never imports proto.
type Device struct {
	ID           string
	OwnerID      string
	Name         string
	Kind         string
	Platform     string
	Capabilities []string
	TrustState   TrustState
	Online       bool
	// SessionID is the authenticator session bound to this device, if any.
	// Revocation asks auth to drop it. Blank when the device has no bound
	// authenticator session (e.g. a redeem that did not carry one).
	SessionID  string
	LastSeenAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Profile is the self-description a joining device supplies on both pairing
// paths. Name falls back to a derived default when blank (service policy).
type Profile struct {
	Name         string
	Kind         string
	Platform     string
	Capabilities []string
}

// PairingCode is a short-TTL, single-use credential a new device redeems to
// join the trust group. The raw Code is returned only at issue time; the
// repository stores its hash.
type PairingCode struct {
	// Code is the raw value — populated only on the issue path, never read back.
	Code       string
	CodeHash   string
	OwnerID    string
	DeviceName string
	ExpiresAt  time.Time
	RedeemedAt time.Time
	CreatedAt  time.Time
}

// Redeemed reports whether the code has already been consumed (single-use).
func (p PairingCode) Redeemed() bool { return !p.RedeemedAt.IsZero() }

// Expired reports whether the code is past its TTL as of now.
func (p PairingCode) Expired(now time.Time) bool { return now.After(p.ExpiresAt) }

// IssuedToken is the result of a successful pair/redeem: the persisted Device
// plus the raw device token to hand back to the caller exactly once.
type IssuedToken struct {
	Device Device
	// Token is the raw hub device token. Only the hash is persisted; this is
	// the only moment the raw value exists.
	Token string
}

// ---- Typed sentinels (translated at the handler edge) -----------------------

// ErrDeviceNotFound is returned when no device matches an id+owner scope.
type ErrDeviceNotFound struct{ ID string }

func (e ErrDeviceNotFound) Error() string { return fmt.Sprintf("device %q not found", e.ID) }

// ErrInvalidDevice is returned when validation fails (bad name, missing field).
type ErrInvalidDevice struct {
	Field  string
	Reason string
}

func (e ErrInvalidDevice) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrInvalidPairingCode is returned when a redeemed code is unknown, expired,
// or already used. Reason is human-safe; it deliberately does not distinguish
// "unknown" from "expired" to a redeeming caller (avoid a code-probing oracle).
type ErrInvalidPairingCode struct{ Reason string }

func (e ErrInvalidPairingCode) Error() string {
	if e.Reason == "" {
		return "pairing code is invalid or expired"
	}
	return e.Reason
}

// ErrDeviceConflict is returned when a state transition is illegal (e.g.
// approving a device that is not pending).
type ErrDeviceConflict struct{ Reason string }

func (e ErrDeviceConflict) Error() string { return e.Reason }
