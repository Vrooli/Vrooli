package mocks

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// FakeDriver is the canonical driver.Driver implementation for tests.
// It supports the entire Driver surface (MountDriver + ChangeTracker)
// plus optional MountVerifier semantics; tests select that variant by
// holding a *FakeDriver and calling VerifyMountIntegrity directly.
//
// The default FakeDriver reports available=true and a fixed set of
// /tmp paths from Mount. Tests override fields to drive specific
// scenarios:
//
//	drv := mocks.NewFakeDriver()
//	drv.MountErr = errors.New("disk full")
//	// Mount now returns the canned error.
//
// Mount tracking (the `Mounted` flag) flips on successful Mount and
// off on Unmount/Cleanup so tests can assert lifecycle correctness.
//
// CleanupOrphan records every ID it was asked to clean (in
// `OrphanCleanups`) and removes the ID from `ListDirsResult` so
// re-runs are no-ops, matching the production fuse-overlayfs cleanup
// contract used by orphan reconcilers.
type FakeDriver struct {
	mu sync.Mutex

	// Identity
	IDValue           driver.DriverID
	VersionValue      string
	IsolationModeVal  driver.IsolationMode
	CapabilitiesValue driver.DriverCapabilities

	// Default Mount return value. Mount always succeeds with this
	// payload unless MountErr is set.
	MountPaths *driver.MountPaths

	// Default ChangeTracker return value.
	ChangedFiles []*types.FileChange

	// Lifecycle state. Flipped by Mount/Unmount/Cleanup.
	Mounted bool

	// Orphan-reconciler hooks.
	ListDirsResult   []uuid.UUID
	OrphanCleanups   []uuid.UUID
	CleanupFailIDs   map[uuid.UUID]bool
	CleanedSandboxes []uuid.UUID

	// Availability. Default true.
	Available bool

	// Per-method error injection. Zero value means no error.
	IsAvailableErr     error
	MountErr           error
	UnmountErr         error
	CleanupErr         error
	GetChangedFilesErr error
	RemoveFromUpperErr error
	VerifyMountErr     error
	ListSandboxDirsErr error
	CleanupOrphanErr   error
}

// NewFakeDriver returns a FakeDriver with sensible defaults: ID="mock",
// available=true, MountPaths set to /tmp/{lower,upper,work,merged},
// no errors.
func NewFakeDriver() *FakeDriver {
	return &FakeDriver{
		IDValue:          "mock",
		VersionValue:     "1.0.0",
		IsolationModeVal: driver.ModeNone,
		CapabilitiesValue: driver.DriverCapabilities{
			HomeOverlay:        false,
			CoW:                false,
			NamespaceIsolation: driver.ModeNone,
		},
		MountPaths: &driver.MountPaths{
			LowerDir:  "/tmp/lower",
			UpperDir:  "/tmp/upper",
			WorkDir:   "/tmp/work",
			MergedDir: "/tmp/merged",
		},
		ChangedFiles:   []*types.FileChange{},
		Available:      true,
		CleanupFailIDs: make(map[uuid.UUID]bool),
	}
}

// --- driver.MountDriver implementation ---

func (d *FakeDriver) ID() driver.DriverID                     { return d.IDValue }
func (d *FakeDriver) Version() string                         { return d.VersionValue }
func (d *FakeDriver) RequiresBwrap() driver.IsolationMode     { return d.IsolationModeVal }
func (d *FakeDriver) Capabilities() driver.DriverCapabilities { return d.CapabilitiesValue }

func (d *FakeDriver) IsAvailable(ctx context.Context) (bool, error) {
	if d.IsAvailableErr != nil {
		return false, d.IsAvailableErr
	}
	return d.Available, nil
}

func (d *FakeDriver) Mount(ctx context.Context, s *types.Sandbox) (*driver.MountPaths, error) {
	if d.MountErr != nil {
		return nil, d.MountErr
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Mounted = true
	return d.MountPaths, nil
}

func (d *FakeDriver) Unmount(ctx context.Context, s *types.Sandbox) error {
	if d.UnmountErr != nil {
		return d.UnmountErr
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Mounted = false
	return nil
}

func (d *FakeDriver) Cleanup(ctx context.Context, s *types.Sandbox) error {
	d.mu.Lock()
	d.CleanedSandboxes = append(d.CleanedSandboxes, s.ID)
	if len(d.CleanupFailIDs) > 0 {
		if d.CleanupFailIDs[s.ID] && d.CleanupErr != nil {
			d.mu.Unlock()
			return d.CleanupErr
		}
	} else if d.CleanupErr != nil {
		d.mu.Unlock()
		return d.CleanupErr
	}
	d.Mounted = false
	d.mu.Unlock()
	return nil
}

func (d *FakeDriver) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	if d.ListSandboxDirsErr != nil {
		return nil, d.ListSandboxDirsErr
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]uuid.UUID, len(d.ListDirsResult))
	copy(out, d.ListDirsResult)
	return out, nil
}

func (d *FakeDriver) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.CleanupOrphanErr != nil {
		return d.CleanupOrphanErr
	}
	d.OrphanCleanups = append(d.OrphanCleanups, id)
	// Remove the ID from ListDirsResult so a second pass treats this
	// as cleaned. Mirrors production driver behavior.
	out := d.ListDirsResult[:0]
	for _, x := range d.ListDirsResult {
		if x != id {
			out = append(out, x)
		}
	}
	d.ListDirsResult = out
	return nil
}

// --- driver.ChangeTracker implementation ---

func (d *FakeDriver) GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error) {
	if d.GetChangedFilesErr != nil {
		return nil, d.GetChangedFilesErr
	}
	return d.ChangedFiles, nil
}

func (d *FakeDriver) RemoveFromUpper(ctx context.Context, s *types.Sandbox, relPath string) error {
	if d.RemoveFromUpperErr != nil {
		return d.RemoveFromUpperErr
	}
	return nil
}

// --- driver.MountVerifier (opt-in) implementation ---

// VerifyMountIntegrity satisfies driver.MountVerifier so callers that
// reach FakeDriver via VerifyIfSupported see the same behavior they'd
// see from a real mount-backed driver. Tests that want CopyDriver-like
// semantics (no MountVerifier) can wrap the fake to strip this method.
func (d *FakeDriver) VerifyMountIntegrity(ctx context.Context, s *types.Sandbox) error {
	return d.VerifyMountErr
}

// Compile-time interface guards.
var (
	_ driver.Driver        = (*FakeDriver)(nil)
	_ driver.MountVerifier = (*FakeDriver)(nil)
)
