// Package fsmountmocks — FakeMounter for fsmount.Mounter.
//
// Lives in a subpackage of testutil/mocks to avoid an import cycle:
// fsmount imports process, and the top-level mocks package imports
// process indirectly via clock — placing FakeMounter alongside
// FakeClock would create a cycle when process tests use FakeClock.
package fsmountmocks

import (
	"context"
	"fmt"
	"sync"

	"workspace-sandbox/internal/fsmount"
)

// FakeMounter is the test double for fsmount.Mounter.
type FakeMounter struct {
	mu sync.Mutex

	// MountCalls / UnmountCalls record every operation in order.
	MountCalls   []fsmount.MountOpts
	UnmountCalls []UnmountCall

	// mounted tracks the set of Merged paths currently considered
	// mounted. Mount adds an entry; Unmount removes one.
	mounted map[string]struct{}

	// extraMountPoints lets tests pre-seed IsMountPoint with paths that
	// were not Mounted via this fake (e.g. simulating leftover host
	// mounts).
	extraMountPoints map[string]struct{}

	// per-op errors. Per-target overrides take precedence over the
	// default *Err fields.
	mountErr      error
	unmountErr    error
	mountErrPer   map[string]error // keyed by Merged
	unmountErrPer map[string]error // keyed by target

	// silentMount lists merged targets for which Mount returns nil but
	// the merged path is NOT added to the mounted set. Models the
	// "fuse-overlayfs forks-and-dies before signalling failure" /
	// "kernel mount returned 0 but no kernel mount appeared"
	// stale-daemon scenarios that drove the verifyMounted post-mount
	// check; tests use this to force the verify path to fire.
	silentMount map[string]struct{}
}

// UnmountCall records a single Unmount invocation.
type UnmountCall struct {
	Target string
	Lazy   bool
}

// NewFakeMounter constructs a FakeMounter with empty state.
func NewFakeMounter() *FakeMounter {
	return &FakeMounter{
		mounted:          map[string]struct{}{},
		extraMountPoints: map[string]struct{}{},
		mountErrPer:      map[string]error{},
		unmountErrPer:    map[string]error{},
		silentMount:      map[string]struct{}{},
	}
}

// Mount implements fsmount.Mounter.
func (m *FakeMounter) Mount(ctx context.Context, opts fsmount.MountOpts) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MountCalls = append(m.MountCalls, opts)
	if err, ok := m.mountErrPer[opts.Merged]; ok {
		return err
	}
	if m.mountErr != nil {
		return m.mountErr
	}
	if opts.Backend == fsmount.BackendUnset {
		return fmt.Errorf("FakeMounter.Mount: opts.Backend is required")
	}
	if _, silent := m.silentMount[opts.Merged]; silent {
		// Silent failure: pretend the mount syscall returned 0 but the
		// kernel/userspace daemon never actually attached. Drivers must
		// catch this via verifyMounted; this branch is the regression
		// guard for exactly that contract.
		return nil
	}
	m.mounted[opts.Merged] = struct{}{}
	return nil
}

// Unmount implements fsmount.Mounter.
func (m *FakeMounter) Unmount(ctx context.Context, target string, lazy bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UnmountCalls = append(m.UnmountCalls, UnmountCall{Target: target, Lazy: lazy})
	if err, ok := m.unmountErrPer[target]; ok {
		return err
	}
	if m.unmountErr != nil {
		return m.unmountErr
	}
	delete(m.mounted, target)
	delete(m.extraMountPoints, target)
	return nil
}

// IsMountPoint implements fsmount.Mounter.
func (m *FakeMounter) IsMountPoint(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.mounted[path]; ok {
		return true
	}
	_, ok := m.extraMountPoints[path]
	return ok
}

// SetMountErr makes every subsequent Mount call return err.
func (m *FakeMounter) SetMountErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mountErr = err
}

// SetUnmountErr makes every subsequent Unmount call return err.
func (m *FakeMounter) SetUnmountErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unmountErr = err
}

// SetMountErrFor scopes a Mount error to a specific Merged target.
func (m *FakeMounter) SetMountErrFor(target string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err == nil {
		delete(m.mountErrPer, target)
		return
	}
	m.mountErrPer[target] = err
}

// SetUnmountErrFor scopes an Unmount error to a specific target.
func (m *FakeMounter) SetUnmountErrFor(target string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err == nil {
		delete(m.unmountErrPer, target)
		return
	}
	m.unmountErrPer[target] = err
}

// AddMountPoint pre-seeds the mounted set with path. Use this to
// simulate "this path was already a mount point before the test
// started."
func (m *FakeMounter) AddMountPoint(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.extraMountPoints[path] = struct{}{}
}

// SetSilentMountFor marks merged so that the next Mount targeting it
// returns nil but does NOT register it as mounted. The driver's
// post-mount verify (IsMountPoint) will then observe a missing mount
// and surface an error — exactly the "fuse-overlayfs forks-and-dies
// before signalling failure" / "kernel returned 0 but no mount
// appeared" stale-daemon scenarios that motivated verifyMounted.
//
// Pass merged="" to clear all silent-mount markers.
func (m *FakeMounter) SetSilentMountFor(merged string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if merged == "" {
		m.silentMount = map[string]struct{}{}
		return
	}
	m.silentMount[merged] = struct{}{}
}

// IsMounted reports whether the merged path is in the mounted set.
func (m *FakeMounter) IsMounted(path string) bool { return m.IsMountPoint(path) }

// Reset clears recorded calls and mounted state.
func (m *FakeMounter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MountCalls = nil
	m.UnmountCalls = nil
	m.mounted = map[string]struct{}{}
	m.extraMountPoints = map[string]struct{}{}
	m.silentMount = map[string]struct{}{}
}

// Compile-time interface assertion.
var _ fsmount.Mounter = (*FakeMounter)(nil)
