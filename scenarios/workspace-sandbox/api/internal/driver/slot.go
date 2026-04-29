package driver

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// Slot is a thread-safe holder for the currently active Driver. It
// replaces the heavier RWMutex-backed Manager with an atomic.Pointer.
//
// Slot itself implements Driver (and MountVerifier) by proxying every
// call to the currently loaded driver. In-flight operations capture the
// driver reference once at the top of the call (via atomic.Load); a
// concurrent SwitchDriver only affects subsequent operations. The proxy
// is necessary so long-lived holders (sandbox.Service, gc.Service) see
// hot-swaps without re-wiring.
type Slot struct {
	p atomic.Pointer[Driver]
}

// Compile-time guarantees: Slot implements both the composite Driver
// interface and MountVerifier (via runtime delegation; see
// VerifyMountIntegrity below for why MountVerifier is wired even though
// the inner driver may not support it).
var (
	_ Driver        = (*Slot)(nil)
	_ MountVerifier = (*Slot)(nil)
)

// NewSlot creates a Slot pre-populated with initial.
func NewSlot(initial Driver) *Slot {
	s := &Slot{}
	s.p.Store(&initial)
	return s
}

// Current returns the driver currently in the slot. Returns nil only if
// the slot itself is nil (defensive: never happens in normal operation).
func (s *Slot) Current() Driver {
	if s == nil {
		return nil
	}
	p := s.p.Load()
	if p == nil {
		return nil
	}
	return *p
}

// Store installs d as the active driver. SwitchDriver is the only path
// that should call this in production code; tests may use Store directly.
func (s *Slot) Store(d Driver) {
	if s == nil {
		return
	}
	s.p.Store(&d)
}

// SwitchDriver atomically switches to a new driver based on the option ID.
// Sequence: NewDriverFor → IsAvailable → Store → SaveDriverPreference.
// A failure at any step before Store leaves the slot untouched.
func SwitchDriver(ctx context.Context, slot *Slot, cfg Config, optionID DriverOptionID) error {
	if slot == nil {
		return fmt.Errorf("driver slot is nil")
	}
	newDriver, err := NewDriverFor(cfg, optionID)
	if err != nil {
		return err
	}
	available, err := newDriver.IsAvailable(ctx)
	if err != nil {
		return fmt.Errorf("failed to check driver availability: %w", err)
	}
	if !available {
		return fmt.Errorf("driver %s is not available on this system", optionID)
	}

	old := slot.Current()
	slot.Store(newDriver)
	if old != nil {
		log.Printf("driver: switched from %s to %s", old.Type(), newDriver.Type())
	} else {
		log.Printf("driver: set to %s", newDriver.Type())
	}

	if err := SaveDriverPreference(cfg.BaseDir, string(optionID)); err != nil {
		log.Printf("driver: warning: failed to save preference: %v", err)
	}
	return nil
}

// --- Driver interface proxy methods ---
//
// Each method captures the inner driver via a single atomic.Load, then
// delegates. This matches the prior Manager semantics: in-flight calls
// continue with the driver they captured; new calls see the post-Switch
// driver.

func (s *Slot) Type() DriverType                   { return s.Current().Type() }
func (s *Slot) Version() string                    { return s.Current().Version() }
func (s *Slot) IsAvailable(ctx context.Context) (bool, error) {
	return s.Current().IsAvailable(ctx)
}

func (s *Slot) Mount(ctx context.Context, sb *types.Sandbox) (*MountPaths, error) {
	return s.Current().Mount(ctx, sb)
}

func (s *Slot) Unmount(ctx context.Context, sb *types.Sandbox) error {
	return s.Current().Unmount(ctx, sb)
}

func (s *Slot) Cleanup(ctx context.Context, sb *types.Sandbox) error {
	return s.Current().Cleanup(ctx, sb)
}

func (s *Slot) ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error) {
	return s.Current().ListSandboxDirs(ctx)
}

func (s *Slot) CleanupOrphan(ctx context.Context, id uuid.UUID) error {
	return s.Current().CleanupOrphan(ctx, id)
}

func (s *Slot) GetChangedFiles(ctx context.Context, sb *types.Sandbox) ([]*types.FileChange, error) {
	return s.Current().GetChangedFiles(ctx, sb)
}

func (s *Slot) RemoveFromUpper(ctx context.Context, sb *types.Sandbox, relPath string) error {
	return s.Current().RemoveFromUpper(ctx, sb, relPath)
}

// VerifyMountIntegrity delegates to VerifyIfSupported on the inner driver
// so a Slot wrapping CopyDriver (no real mount) returns nil rather than
// panicking on a missing method. Callers that hold a *Slot can therefore
// either type-assert to MountVerifier and call this directly, or use
// driver.VerifyIfSupported(ctx, slot, sb) — both produce the correct
// result for every driver type.
func (s *Slot) VerifyMountIntegrity(ctx context.Context, sb *types.Sandbox) error {
	return VerifyIfSupported(ctx, s.Current(), sb)
}
