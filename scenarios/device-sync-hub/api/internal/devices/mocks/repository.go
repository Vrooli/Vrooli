// Package mocks holds devices-domain test fakes co-located with the domain they
// double for. mocks imports devices; devices never imports mocks, so deleting
// the domain folder takes its fakes with it (no central residue).
package mocks

import (
	"context"
	"sort"
	"time"

	"device-sync-hub/internal/devices"
)

// FakeRepository is an in-memory devices.Repository for service unit tests.
// It faithfully models the behaviours the service relies on — owner scoping,
// single-use conditional pairing-code claim, trust transitions — so a service
// test exercises real policy rather than a stub that always says yes.
//
// Arrange by mutating the exported error hooks (e.g. CreateDeviceErr) before
// the code under test runs; inspect Devices / Codes afterwards.
type FakeRepository struct {
	Devices map[string]devices.Device // keyed by device id
	Codes   map[string]storedCode     // keyed by code hash

	// Owner / OwnerClaimed model the single-row hub_owner claim. Inspect Owner
	// after a SetupOwnerDevice to assert the hub got claimed.
	Owner        string
	OwnerClaimed bool

	// nextID lets tests get stable ids; when "" the repo derives seq ids.
	seq int

	// Error injection hooks — set to force a failure on the matching call.
	CreateDeviceErr error
	GetDeviceErr    error
	ListErr         error
	SetTrustErr     error
	RenameErr       error
	ByTokenErr      error
	ClaimOwnerErr   error
	HubOwnerErr     error
	CreateCodeErr   error
	ClaimErr        error

	// TokenHashes records the token hash persisted per device id, so token
	// issuance tests can assert the raw token hashes to the stored value.
	TokenHashes map[string]string
}

type storedCode struct {
	code devices.PairingCode
}

// NewFakeRepository returns an empty, ready-to-arrange fake.
func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		Devices:     map[string]devices.Device{},
		Codes:       map[string]storedCode{},
		TokenHashes: map[string]string{},
	}
}

// Compile-time guarantee.
var _ devices.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) CreateDevice(_ context.Context, d devices.Device, tokenHash string) (devices.Device, error) {
	if f.CreateDeviceErr != nil {
		return devices.Device{}, f.CreateDeviceErr
	}
	if d.ID == "" {
		f.seq++
		d.ID = idFromSeq(f.seq)
	}
	now := time.Now().UTC()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	}
	if d.LastSeenAt.IsZero() {
		d.LastSeenAt = d.CreatedAt
	}
	f.Devices[d.ID] = d
	f.TokenHashes[d.ID] = tokenHash
	return d, nil
}

func (f *FakeRepository) GetDevice(_ context.Context, ownerID, id string) (devices.Device, error) {
	if f.GetDeviceErr != nil {
		return devices.Device{}, f.GetDeviceErr
	}
	d, ok := f.Devices[id]
	if !ok || d.OwnerID != ownerID {
		return devices.Device{}, devices.ErrDeviceNotFound{ID: id}
	}
	return d, nil
}

func (f *FakeRepository) ListDevices(_ context.Context, ownerID string) ([]devices.Device, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	var out []devices.Device
	for _, d := range f.Devices {
		if d.OwnerID == ownerID {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (f *FakeRepository) SetTrust(_ context.Context, ownerID, id string, state devices.TrustState) (devices.Device, error) {
	if f.SetTrustErr != nil {
		return devices.Device{}, f.SetTrustErr
	}
	d, ok := f.Devices[id]
	if !ok || d.OwnerID != ownerID {
		return devices.Device{}, devices.ErrDeviceNotFound{ID: id}
	}
	d.TrustState = state
	d.UpdatedAt = time.Now().UTC()
	f.Devices[id] = d
	return d, nil
}

func (f *FakeRepository) Rename(_ context.Context, ownerID, id, name string) (devices.Device, error) {
	if f.RenameErr != nil {
		return devices.Device{}, f.RenameErr
	}
	d, ok := f.Devices[id]
	if !ok || d.OwnerID != ownerID {
		return devices.Device{}, devices.ErrDeviceNotFound{ID: id}
	}
	d.Name = name
	d.UpdatedAt = time.Now().UTC()
	f.Devices[id] = d
	return d, nil
}

func (f *FakeRepository) DeviceByToken(_ context.Context, tokenHash string) (devices.Device, error) {
	if f.ByTokenErr != nil {
		return devices.Device{}, f.ByTokenErr
	}
	if tokenHash == "" {
		return devices.Device{}, devices.ErrDeviceNotFound{}
	}
	for id, h := range f.TokenHashes {
		if h == tokenHash {
			return f.Devices[id], nil
		}
	}
	return devices.Device{}, devices.ErrDeviceNotFound{}
}

func (f *FakeRepository) ClaimOwner(_ context.Context, ownerID string, _ time.Time) (string, error) {
	if f.ClaimOwnerErr != nil {
		return "", f.ClaimOwnerErr
	}
	if !f.OwnerClaimed {
		f.Owner = ownerID
		f.OwnerClaimed = true
	}
	return f.Owner, nil
}

func (f *FakeRepository) HubOwner(context.Context) (string, error) {
	if f.HubOwnerErr != nil {
		return "", f.HubOwnerErr
	}
	if !f.OwnerClaimed {
		return "", devices.ErrNoOwner
	}
	return f.Owner, nil
}

func (f *FakeRepository) CreatePairingCode(_ context.Context, c devices.PairingCode) error {
	if f.CreateCodeErr != nil {
		return f.CreateCodeErr
	}
	f.Codes[c.CodeHash] = storedCode{code: c}
	return nil
}

func (f *FakeRepository) ClaimPairingCode(_ context.Context, codeHash string, now time.Time) (devices.PairingCode, error) {
	if f.ClaimErr != nil {
		return devices.PairingCode{}, f.ClaimErr
	}
	sc, ok := f.Codes[codeHash]
	if !ok || sc.code.Redeemed() || sc.code.Expired(now) {
		return devices.PairingCode{}, devices.ErrInvalidPairingCode{}
	}
	sc.code.RedeemedAt = now
	f.Codes[codeHash] = sc
	out := sc.code
	out.CodeHash = codeHash
	return out, nil
}

// idFromSeq is a tiny deterministic id generator for fake-created devices.
func idFromSeq(n int) string {
	return "dev-" + string(rune('0'+n%10)) + string(rune('a'+n/10%26))
}
