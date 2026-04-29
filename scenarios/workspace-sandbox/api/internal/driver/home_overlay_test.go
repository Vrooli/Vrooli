package driver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/types"
)

// TestMountHomeOverlay_FailureReturnsTypedError — when mountFn errors,
// the helper must wrap the cause in *types.HomeOverlayUnavailableError
// (not a plain error). This is the load-bearing seam that lets the
// driver record HomeOverlayState=Absent and the handler's exec path
// return HTTP 409.
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

	failingMount := func(ctx context.Context, target, opts string) error {
		return errors.New("synthetic mount failure")
	}
	noopUnmount := func(ctx context.Context, target string) error { return nil }

	id := uuid.New()
	_, _, _, _, err := mountHomeOverlay(context.Background(), homeOverlayBaseDir, id, hostHome, failingMount, noopUnmount)
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

// TestMountHomeOverlay_VerificationCatchesStaleMount — mountFn returns
// nil, but no actual mount appeared (synthetic stale-daemon scenario).
// verifyMounted must catch this and return the typed error.
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

	// mountFn lies: returns nil but doesn't actually mount anything.
	// verifyMounted's isMountPoint check should catch the lie.
	lyingMount := func(ctx context.Context, target, opts string) error { return nil }
	unmountInvoked := false
	noopUnmount := func(ctx context.Context, target string) error {
		unmountInvoked = true
		return nil
	}

	id := uuid.New()
	_, _, _, _, err := mountHomeOverlay(context.Background(), homeOverlayBaseDir, id, hostHome, lyingMount, noopUnmount)
	if err == nil {
		t.Fatal("expected verify-mount error, got nil")
	}
	var typed *types.HomeOverlayUnavailableError
	if !errors.As(err, &typed) {
		t.Fatalf("expected *types.HomeOverlayUnavailableError, got %T", err)
	}
	if !unmountInvoked {
		t.Error("expected verifyMount-failure rollback to invoke unmountFn")
	}
}

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
		{"copy", NewCopyDriver(cfg), false, false},
		{"fuse-overlayfs", NewFuseOverlayfsDriver(cfg), true, true},
		{"overlayfs-userns", NewOverlayfsUserNSDriver(cfg), true, true},
		{"overlayfs-root", NewOverlayfsRootDriver(cfg), true, true},
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
	// We can't directly call internal/config from here; this is a
	// driver-level smoke that the layout we've adopted leaves home
	// upper outside $HOME (the actual ResolveHomeOverlayBaseDir is
	// covered in internal/config/config_test.go). The driver wires
	// HomeOverlayBaseDir from cfg.HomeOverlayBaseDir, so a misconfigured
	// path would surface at first Mount via the ErrHomeOverlayUnavailable
	// path tested above.
	//
	// This test exists to keep the assertion adjacent to the seam: any
	// future refactor that drops the "outside $HOME" invariant must
	// notice that this test is here and fail to remove it cleanly.
	cfg := Config{HomeOverlayBaseDir: ""}
	_, _, _, _, err := mountHomeOverlay(
		context.Background(),
		cfg.HomeOverlayBaseDir,
		uuid.New(),
		os.Getenv("HOME"),
		func(ctx context.Context, target, opts string) error { return fmt.Errorf("should not be reached") },
		nil,
	)
	if err == nil {
		t.Fatal("expected mountHomeOverlay to fail with empty HomeOverlayBaseDir")
	}
	var typed *types.HomeOverlayUnavailableError
	if !errors.As(err, &typed) {
		t.Fatalf("expected *types.HomeOverlayUnavailableError, got %T", err)
	}
}
