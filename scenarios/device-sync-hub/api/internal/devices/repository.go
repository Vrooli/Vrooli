package devices

import (
	"context"
	"time"
)

// Repository is the persistence seam the devices service depends on. Production
// wires the sqlite-backed implementation (sqlite.go); service unit tests wire
// mocks.FakeRepository. The surface stays narrow — methods land when the
// service proves it needs them.
type Repository interface {
	// CreateDevice persists d together with the SHA-256 hash of its hub device
	// token (tokenHash; "" for a device that has none yet). The implementation
	// populates ID/CreatedAt/UpdatedAt/LastSeenAt when zero. Returns the stored
	// Device (token hash never surfaced on the returned value).
	CreateDevice(ctx context.Context, d Device, tokenHash string) (Device, error)

	// GetDevice returns the owner-scoped device or ErrDeviceNotFound{ID}.
	GetDevice(ctx context.Context, ownerID, id string) (Device, error)

	// ListDevices returns the owner's devices, newest-first by created_at.
	ListDevices(ctx context.Context, ownerID string) ([]Device, error)

	// SetTrust transitions an owner-scoped device to state and returns the
	// updated row. ErrDeviceNotFound when no row matches.
	SetTrust(ctx context.Context, ownerID, id string, state TrustState) (Device, error)

	// Rename updates an owner-scoped device's name. ErrDeviceNotFound otherwise.
	Rename(ctx context.Context, ownerID, id, name string) (Device, error)

	// DeviceByToken resolves a device-token hash to its device (any trust
	// state — the caller enforces TRUSTED). ErrDeviceNotFound when no row
	// carries that hash. Backs Phase 3 transfer trust enforcement.
	DeviceByToken(ctx context.Context, tokenHash string) (Device, error)

	// ClaimOwner records ownerID as the hub's single owner, first-owner-wins:
	// the claim is an atomic conditional insert, so concurrent claims can never
	// both succeed. It returns the owner that holds the hub *after* the call —
	// ownerID when the hub was unclaimed (or already theirs), or the existing
	// owner when someone else got there first. The caller compares the returned
	// id to ownerID to learn whether the claim is theirs.
	ClaimOwner(ctx context.Context, ownerID string, now time.Time) (string, error)

	// HubOwner returns the hub's single owner id, or ErrNoOwner when the hub is
	// unclaimed (no SetupOwnerDevice has run yet). Ownership is an explicit hub
	// fact (hub_owner row), never derived from device rows.
	HubOwner(ctx context.Context) (string, error)

	// CreatePairingCode persists a freshly-issued code (hash + metadata).
	CreatePairingCode(ctx context.Context, c PairingCode) error

	// ClaimPairingCode atomically marks the single-use code consumed iff it is
	// known, unredeemed, and unexpired as of now, returning the claimed code's
	// metadata (owner, device name). Returns ErrInvalidPairingCode when the
	// conditional claim matches no row — the caller cannot tell unknown from
	// expired from used (no probing oracle).
	ClaimPairingCode(ctx context.Context, codeHash string, now time.Time) (PairingCode, error)
}
