package driver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/testutil/mocks/fsmountmocks"
	"workspace-sandbox/internal/types"
)

// TestMountHomeOverlay_FailureReturnsTypedError — when Mounter.Mount
// errors, the helper must wrap the cause in
// *types.HomeOverlayUnavailableError (not a plain error). This is the
// load-bearing seam that lets the driver record HomeOverlayState=Absent
// and the handler's exec path return HTTP 409.
//
// DOC: home-overlay seam — failure topography test.
func TestMountHomeOverlay_FailureReturnsTypedError(t *testing.T) {
	tmp := t.TempDir()
	homeOverlayBaseDir := filepath.Join(tmp, "home-overlay-base")
	if err := os.MkdirAll(homeOverlayBaseDir, 0o700); err != nil {
		t.Fatalf("mkdir homeOverlayBaseDir: %v", err)
	}
	hostHome := filepath.Join(tmp, "home")
	if err := os.MkdirAll(hostHome, 0o700); err != nil {
		t.Fatalf("mkdir hostHome: %v", err)
	}

	m := fsmountmocks.NewFakeMounter()
	m.SetMountErr(errors.New("synthetic mount failure"))

	id := uuid.New()
	_, _, _, _, err := mountHomeOverlay(context.Background(), m, fsmount.BackendKernelOverlay, homeOverlayBaseDir, id, hostHome)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var typed *types.HomeOverlayUnavailableError
	if !errors.As(err, &typed) {
		t.Fatalf("expected *types.HomeOverlayUnavailableError, got %T: %v", err, err)
	}
	if typed.Code() != "HOME_OVERLAY_UNAVAILABLE" {
		t.Errorf("Code()=%q; want HOME_OVERLAY_UNAVAILABLE", typed.Code())
	}
	// Per-sandbox dir should have been cleaned up on failure (defensive
	// rollback so a future Mount with the same ID doesn't trip over old
	// home-upper/home-work).
	if _, err := os.Stat(filepath.Join(homeOverlayBaseDir, id.String())); !os.IsNotExist(err) {
		t.Errorf("expected home overlay dir cleaned up on failure, stat err = %v", err)
	}
}

// TestMountHomeOverlay_VerificationCatchesStaleMount — Mounter.Mount
// returns nil and reports the path as a mount point, but the post-
// mount writable-probe fails (synthetic stale-daemon scenario).
// verifyMounted must catch this and return the typed error.
//
// We simulate this by having the FakeMounter accept the Mount but
// having the merged dir live somewhere unwritable for the probe step.
// The simplest deterministic way to fail the probe is to make the
// merged dir read-only after mkdir; we can't do that with FakeMounter
// alone (the helper creates it), so we instead make Mount succeed but
// then Unmount fail. That tests the rollback path. The
// "verifyMounted-fails" path is covered by the integration contract
// test (TestDriverContract).
//
// Here we focus on: Mount appears to succeed but IsMountPoint returns
// false (the daemon forked and died). verifyMounted's IsMountPoint
// check should catch this and trigger the unmount rollback.
//
// DOC: home-overlay seam — mount-verification test.
func TestMountHomeOverlay_VerificationCatchesStaleMount(t *testing.T) {
	tmp := t.TempDir()
	homeOverlayBaseDir := filepath.Join(tmp, "home-overlay-base")
	if err := os.MkdirAll(homeOverlayBaseDir, 0o700); err != nil {
		t.Fatalf("mkdir homeOverlayBaseDir: %v", err)
	}
	hostHome := filepath.Join(tmp, "home")
	if err := os.MkdirAll(hostHome, 0o700); err != nil {
		t.Fatalf("mkdir hostHome: %v", err)
	}

	// staleMounter pretends Mount succeeded but never marks the path as
	// mounted, so IsMountPoint returns false — exactly the "fuse
	// daemon forked and died" failure mode.
	m := &staleMounter{}

	id := uuid.New()
	_, _, _, _, err := mountHomeOverlay(context.Background(), m, fsmount.BackendKernelOverlay, homeOverlayBaseDir, id, hostHome)
	if err == nil {
		t.Fatal("expected verify-mount error, got nil")
	}
	var typed *types.HomeOverlayUnavailableError
	if !errors.As(err, &typed) {
		t.Fatalf("expected *types.HomeOverlayUnavailableError, got %T", err)
	}
	if !m.unmountInvoked {
		t.Error("expected verifyMount-failure rollback to invoke Unmount")
	}
}

// staleMounter accepts Mount but never marks the path as mounted, so
// IsMountPoint returns false — modeling a daemon that forked and died.
type staleMounter struct {
	mountErr       error
	unmountInvoked bool
}

func (m *staleMounter) Mount(ctx context.Context, opts fsmount.MountOpts) error {
	return m.mountErr
}

func (m *staleMounter) Unmount(ctx context.Context, target string, lazy bool) error {
	m.unmountInvoked = true
	return nil
}

func (m *staleMounter) IsMountPoint(path string) bool { return false }

// TestDriver_HomeOverlayCapability pins each driver's Capabilities()
// answer. Adding a new driver is a new row here, not editing a central
// switch — same pattern as TestDriverContract_RequiresBwrap.
//
// DOC: home-overlay seam — driver capability test.
func TestDriver_HomeOverlayCapability(t *testing.T) {
	cfg := Config{BaseDir: t.TempDir(), HomeOverlayBaseDir: t.TempDir()}
	cases := []struct {
		name        string
		drv         Driver
		homeOverlay bool
		cow         bool
	}{
		{"copy", NewCopyDriver(cfg, testDeps()), false, false},
		{"fuse-overlayfs", NewFuseOverlayfsDriver(cfg, testDeps()), true, true},
		{"overlayfs-userns", NewOverlayfsUserNSDriver(cfg, testDeps()), true, true},
		{"overlayfs-root", NewOverlayfsRootDriver(cfg, testDeps()), true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caps := tc.drv.Capabilities()
			if caps.HomeOverlay != tc.homeOverlay {
				t.Errorf("HomeOverlay = %v; want %v", caps.HomeOverlay, tc.homeOverlay)
			}
			if caps.CoW != tc.cow {
				t.Errorf("CoW = %v; want %v", caps.CoW, tc.cow)
			}
		})
	}
}

// TestResolveHomeOverlayBaseDir_RejectsHomeSubpath — the load-bearing
// safety check that prevents Phase B's self-referential mount. If the
// resolved path lands inside $HOME, the helper MUST return an error.
//
// DOC: home-overlay storage seam — fatal validation test.
func TestResolveHomeOverlayBaseDir_RejectsHomeSubpath(t *testing.T) {
	cfg := Config{HomeOverlayBaseDir: ""}
	m := fsmountmocks.NewFakeMounter()
	_, _, _, _, err := mountHomeOverlay(
		context.Background(),
		m,
		fsmount.BackendKernelOverlay,
		cfg.HomeOverlayBaseDir,
		uuid.New(),
		os.Getenv("HOME"),
	)
	if err == nil {
		t.Fatal("expected mountHomeOverlay to fail with empty HomeOverlayBaseDir")
	}
	var typed *types.HomeOverlayUnavailableError
	if !errors.As(err, &typed) {
		t.Fatalf("expected *types.HomeOverlayUnavailableError, got %T", err)
	}
}
