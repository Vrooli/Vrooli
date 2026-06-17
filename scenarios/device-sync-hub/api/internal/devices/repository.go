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

	// ResolveOwner returns the single owner this hub belongs to, derived from
	// existing device rows (v1 is single-owner-many-devices). Returns
	// ErrDeviceConflict when no device exists yet — the very first device must
	// bootstrap via the authenticated code path, not the fallback request path.
	ResolveOwner(ctx context.Context) (string, error)

	// CreatePairingCode persists a freshly-issued code (hash + metadata).
	CreatePairingCode(ctx context.Context, c PairingCode) error

	// ClaimPairingCode atomically marks the single-use code consumed iff it is
	// known, unredeemed, and unexpired as of now, returning the claimed code's
	// metadata (owner, device name). Returns ErrInvalidPairingCode when the
	// conditional claim matches no row — the caller cannot tell unknown from
	// expired from used (no probing oracle).
	ClaimPairingCode(ctx context.Context, codeHash string, now time.Time) (PairingCode, error)
}
