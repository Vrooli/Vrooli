package driver

// contract_failure_test.go — failure-mode contract tests for every driver.
//
// The companion file contract_test.go pins the happy path against a real
// kernel/fuse mount (skipped when the host can't satisfy IsAvailable).
// This file pins the failure modes against a FakeMounter so the same
// invariants are observable inside `go test` on any host:
//
//   - Project mount syscall errors → Mount returns error, no leftover
//     sandbox dir, no recorded mount-success.
//   - Home overlay mount syscall errors → Mount succeeds with
//     HomeOverlayState=Absent and the project paths still wired.
//   - Silent mount failure ("syscall returned 0 but no kernel mount
//     attached") → Mount returns error and no leftover sandbox dir.
//     This is the regression guard for the verifyMounted post-mount
//     check that landed in Phase B.
//   - Unmount idempotency → second Unmount on the same target is a no-op.
//   - Cleanup / CleanupOrphan idempotency → second call on the same
//     sandbox is a no-op (no error).
//   - Unmount errors propagate via the returned error.
//   - Partial-approval cycle (write file → GetChangedFiles=Added →
//     RemoveFromUpper → GetChangedFiles=empty) — shared between every
//     overlay-flavored driver since the impl is in helpers.go.
//   - Copy driver lifecycle idempotency.
//
// Parameterization: BackendKernelOverlay covers the overlayfs-userns
// and overlayfs-root flavors (the OverlayDriver shares one body for
// both — only availability + isolation mode differ, neither of which
// the failure paths exercise). BackendFuseOverlayfs covers fuse-
// overlayfs. The copy driver gets its own block since it has no
// Mounter to inject failures through.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/testutil/mocks/fsmountmocks"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/types"
)

// fakeDeps assembles a Deps backed entirely by in-memory fakes. Returned
// alongside the underlying *FakeMounter / *FakeStarter so the test can
// inject errors and inspect recorded calls. Clock is the system clock —
// failure-mode tests don't assert on FileChange.DetectedAt timestamps,
// so deterministic time isn't load-bearing here.
func fakeDeps() (Deps, *fsmountmocks.FakeMounter, *procmocks.FakeStarter) {
	mounter := fsmountmocks.NewFakeMounter()
	starter := procmocks.NewFakeStarter()
	// Tests in this file never invoke a real binary; the failure-mode
	// driver paths only exercise Mounter. SetDefault keeps the Starter
	// lenient: any incidental LookPath call (none expected) returns a
	// not-found error rather than failing on unmatched commands.
	starter.SetDefault(procmocks.CommandBehavior{})
	return Deps{Clock: clock.System{}, Mounter: mounter, Starter: starter}, mounter, starter
}

// overlayFlavor describes one parameterized backend the failure-mode
// tests run against. Fuse and kernel-overlay drivers share the same
// OverlayDriver code path, only differing on backend selection at mount
// time, so a backend axis is sufficient parameterization.
type overlayFlavor struct {
	name    string
	backend fsmount.Backend
	ctor    func(cfg Config, deps Deps) *OverlayDriver
}

func overlayFlavors() []overlayFlavor {
	return []overlayFlavor{
		{
			name:    "fuse-overlayfs",
			backend: fsmount.BackendFuseOverlayfs,
			ctor:    NewFuseOverlayfsDriver,
		},
		{
			name:    "overlayfs-userns",
			backend: fsmount.BackendKernelOverlay,
			ctor:    NewOverlayfsUserNSDriver,
		},
		{
			name:    "overlayfs-root",
			backend: fsmount.BackendKernelOverlay,
			ctor:    NewOverlayfsRootDriver,
		},
	}
}

// failureFixture is the shared per-test scaffold: a tempdir tree with
// scope/ holding a seed file plus a fake-driven driver. The home overlay
// is wired by setting HOME to a writable tempdir under the same base.
type failureFixture struct {
	t                  *testing.T
	tmpDir             string
	scope              string
	fakeHome           string
	baseDir            string
	homeBase           string
	sandbox            *types.Sandbox
	driver             *OverlayDriver
	mounter            *fsmountmocks.FakeMounter
	starter            *procmocks.FakeStarter
	flavor             overlayFlavor
	expectedMergedHome string // path verifyMounted will probe after mountHomeOverlay
	expectedMergedProj string // path verifyMounted will probe after mountProjectOverlay
}

func newFailureFixture(t *testing.T, flavor overlayFlavor) *failureFixture {
	t.Helper()
	tmpDir := t.TempDir()
	scope := filepath.Join(tmpDir, "scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scope, "README.md"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("seed scope: %v", err)
	}
	fakeHome := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(fakeHome, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	t.Setenv("HOME", fakeHome)

	baseDir := filepath.Join(tmpDir, "base")
	homeBase := filepath.Join(tmpDir, "homebase")
	if err := os.MkdirAll(homeBase, 0o700); err != nil {
		t.Fatalf("mkdir homeBase: %v", err)
	}
	cfg := Config{BaseDir: baseDir, HomeOverlayBaseDir: homeBase}

	deps, mounter, starter := fakeDeps()
	d := flavor.ctor(cfg, deps)
	// Force the canonical backend even when the constructor would have
	// auto-selected differently (NewOverlayfsDriver). Each flavor here
	// names exactly one expected backend.
	d.backend = flavor.backend

	id := uuid.New()
	sb := &types.Sandbox{ID: id, ScopePath: scope, ProjectRoot: scope}
	return &failureFixture{
		t:                  t,
		tmpDir:             tmpDir,
		scope:              scope,
		fakeHome:           fakeHome,
		baseDir:            baseDir,
		homeBase:           homeBase,
		sandbox:            sb,
		driver:             d,
		mounter:            mounter,
		starter:            starter,
		flavor:             flavor,
		expectedMergedProj: filepath.Join(baseDir, id.String(), "merged"),
		expectedMergedHome: filepath.Join(homeBase, id.String(), "home-merged"),
	}
}

// TestDriverFailure_ProjectMountErrorPropagates pins: when the kernel/
// fuse mount syscall errors, the driver returns the error and
// rolls back the sandbox dir it had just created. No half-state.
func TestDriverFailure_ProjectMountErrorPropagates(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			fx.mounter.SetMountErrFor(fx.expectedMergedProj, errors.New("EPERM"))

			_, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err == nil {
				t.Fatal("expected error from project mount, got nil")
			}
			if !strings.Contains(err.Error(), "EPERM") {
				t.Errorf("error should wrap EPERM, got %v", err)
			}
			// Rollback contract: the sandbox dir must be gone after a
			// failed mount so a subsequent retry starts clean.
			sandboxDir := filepath.Join(fx.baseDir, fx.sandbox.ID.String())
			if _, statErr := os.Stat(sandboxDir); !os.IsNotExist(statErr) {
				t.Errorf("expected sandbox dir cleaned up, stat err=%v", statErr)
			}
			// Mounter recorded exactly one Mount call (no retries).
			if len(fx.mounter.MountCalls) != 1 {
				t.Errorf("expected 1 Mount call, got %d", len(fx.mounter.MountCalls))
			}
		})
	}
}

// TestDriverFailure_HomeOverlayMountFailsSoft pins: a home-overlay
// mount failure does NOT fail the whole Mount. The project paths are
// returned, the sandbox is marked HomeOverlayState=Absent, and the
// home overlay parent dir is cleaned up so the next retry is clean.
func TestDriverFailure_HomeOverlayMountFailsSoft(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			fx.mounter.SetMountErrFor(fx.expectedMergedHome, errors.New("home EPERM"))

			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: expected success despite home failure, got %v", err)
			}
			if paths.MergedDir != fx.expectedMergedProj {
				t.Errorf("MergedDir: got %q, want %q", paths.MergedDir, fx.expectedMergedProj)
			}
			if paths.HomeMergedDir != "" {
				t.Errorf("HomeMergedDir should be empty after home mount failure, got %q", paths.HomeMergedDir)
			}
			if fx.sandbox.HomeOverlayState != types.HomeOverlayAbsent {
				t.Errorf("HomeOverlayState: got %q, want %q", fx.sandbox.HomeOverlayState, types.HomeOverlayAbsent)
			}
			// Project mount stays mounted; verifyMounted (writable
			// probe) ran against the real merged dir.
			if !fx.mounter.IsMountPoint(fx.expectedMergedProj) {
				t.Errorf("expected project mount to remain mounted")
			}
			// Home overlay parent dir was rolled back.
			homeParent := filepath.Join(fx.homeBase, fx.sandbox.ID.String())
			if _, statErr := os.Stat(homeParent); !os.IsNotExist(statErr) {
				t.Errorf("expected home overlay parent cleaned up, stat err=%v", statErr)
			}
		})
	}
}

// TestDriverFailure_SilentMountCaughtByVerify pins the regression for
// "fuse-overlayfs forks-and-dies" / "kernel-mount returned 0 but no
// kernel mount attached." Mount returns nil, the merged dir exists, but
// IsMountPoint stays false — verifyMounted must surface this as an
// error. Without the post-mount check the driver would have happily
// returned a half-mount.
func TestDriverFailure_SilentMountCaughtByVerify(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			fx.mounter.SetSilentMountFor(fx.expectedMergedProj)

			_, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err == nil {
				t.Fatal("expected verifyMounted to surface silent mount failure")
			}
			if !strings.Contains(err.Error(), "verify") && !strings.Contains(err.Error(), "not a mount point") {
				t.Errorf("error should mention verify / mount point, got %v", err)
			}
			// Rolled back exactly like a hard failure.
			sandboxDir := filepath.Join(fx.baseDir, fx.sandbox.ID.String())
			if _, statErr := os.Stat(sandboxDir); !os.IsNotExist(statErr) {
				t.Errorf("expected sandbox dir cleaned up, stat err=%v", statErr)
			}
		})
	}
}

// TestDriverFailure_HomeOverlaySilentVerifySoft pins the same silent-
// mount semantics for the home overlay: silent failure on the home
// merged dir falls into the soft path (HomeOverlayState=Absent) just
// like a hard mount error. The project mount must NOT be torn down.
func TestDriverFailure_HomeOverlaySilentVerifySoft(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			fx.mounter.SetSilentMountFor(fx.expectedMergedHome)

			_, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: expected silent home failure to be soft, got %v", err)
			}
			if fx.sandbox.HomeOverlayState != types.HomeOverlayAbsent {
				t.Errorf("HomeOverlayState: got %q, want %q", fx.sandbox.HomeOverlayState, types.HomeOverlayAbsent)
			}
			if !fx.mounter.IsMountPoint(fx.expectedMergedProj) {
				t.Errorf("project mount should remain mounted after soft home failure")
			}
			// The fake's silent-mount knob does not register the home
			// merged path either, so the rollback path runs and the
			// home overlay parent dir should be gone.
			homeParent := filepath.Join(fx.homeBase, fx.sandbox.ID.String())
			if _, statErr := os.Stat(homeParent); !os.IsNotExist(statErr) {
				t.Errorf("expected home overlay parent cleaned up, stat err=%v", statErr)
			}
		})
	}
}

// TestDriverFailure_UnmountIdempotent pins: a second Unmount on the
// same target after the first succeeded is a no-op (no error, no
// extra Unmount call to the syscall layer). This is the regression
// guard for the "double Unmount blew up the orphan reaper" pattern.
func TestDriverFailure_UnmountIdempotent(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.LowerDir = paths.LowerDir
			fx.sandbox.UpperDir = paths.UpperDir
			fx.sandbox.WorkDir = paths.WorkDir
			fx.sandbox.MergedDir = paths.MergedDir
			fx.sandbox.HomeMergedDir = paths.HomeMergedDir

			if err := fx.driver.Unmount(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("Unmount #1: %v", err)
			}
			if fx.mounter.IsMountPoint(fx.expectedMergedProj) {
				t.Error("expected unmounted after first Unmount")
			}
			callsAfterFirst := len(fx.mounter.UnmountCalls)
			if callsAfterFirst == 0 {
				t.Error("expected first Unmount to call the mounter")
			}

			if err := fx.driver.Unmount(context.Background(), fx.sandbox); err != nil {
				t.Errorf("Unmount #2 should be a no-op, got %v", err)
			}
			// Second Unmount must NOT touch the mounter again — the
			// !IsMountPoint short-circuit lives in OverlayDriver.Unmount.
			if got := len(fx.mounter.UnmountCalls); got != callsAfterFirst {
				t.Errorf("Unmount #2 made extra mounter calls: got %d, want %d", got, callsAfterFirst)
			}
		})
	}
}

// TestDriverFailure_UnmountErrorPropagates pins: a real mounter error
// during Unmount surfaces all the way back to the caller.
func TestDriverFailure_UnmountErrorPropagates(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.MergedDir = paths.MergedDir
			fx.sandbox.HomeMergedDir = paths.HomeMergedDir

			fx.mounter.SetUnmountErrFor(fx.expectedMergedProj, errors.New("EBUSY"))
			err = fx.driver.Unmount(context.Background(), fx.sandbox)
			if err == nil {
				t.Fatal("expected EBUSY error from Unmount")
			}
			if !strings.Contains(err.Error(), "EBUSY") {
				t.Errorf("expected EBUSY in error, got %v", err)
			}
		})
	}
}

// TestDriverFailure_CleanupIdempotent pins: Cleanup on a sandbox
// whose dir has already been removed is a no-op. Same for CleanupOrphan.
func TestDriverFailure_CleanupIdempotent(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.MergedDir = paths.MergedDir
			fx.sandbox.HomeMergedDir = paths.HomeMergedDir

			if err := fx.driver.Cleanup(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("Cleanup #1: %v", err)
			}
			sandboxDir := filepath.Join(fx.baseDir, fx.sandbox.ID.String())
			if _, statErr := os.Stat(sandboxDir); !os.IsNotExist(statErr) {
				t.Errorf("expected sandbox dir gone after Cleanup, stat err=%v", statErr)
			}
			if err := fx.driver.Cleanup(context.Background(), fx.sandbox); err != nil {
				t.Errorf("Cleanup #2 should be a no-op, got %v", err)
			}
			if err := fx.driver.CleanupOrphan(context.Background(), fx.sandbox.ID); err != nil {
				t.Errorf("CleanupOrphan after Cleanup should be a no-op, got %v", err)
			}
			if err := fx.driver.CleanupOrphan(context.Background(), fx.sandbox.ID); err != nil {
				t.Errorf("CleanupOrphan #2 should be a no-op, got %v", err)
			}
		})
	}
}

// TestDriverFailure_CleanupOrphanWhenStillMounted pins: when the
// repo lost a sandbox but its merged dir is still a mount point on
// disk, CleanupOrphan must Unmount it before rm -rf'ing the dir.
// Without the unmount the rm fails with ENOTEMPTY.
func TestDriverFailure_CleanupOrphanWhenStillMounted(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			if _, err := fx.driver.Mount(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("Mount: %v", err)
			}
			// Don't tear down the mount via Unmount(); CleanupOrphan
			// must do it on our behalf.
			if err := fx.driver.CleanupOrphan(context.Background(), fx.sandbox.ID); err != nil {
				t.Fatalf("CleanupOrphan: %v", err)
			}
			if fx.mounter.IsMountPoint(fx.expectedMergedProj) {
				t.Error("CleanupOrphan should have unmounted the merged dir")
			}
			sandboxDir := filepath.Join(fx.baseDir, fx.sandbox.ID.String())
			if _, statErr := os.Stat(sandboxDir); !os.IsNotExist(statErr) {
				t.Errorf("CleanupOrphan should have removed sandboxDir, stat err=%v", statErr)
			}
		})
	}
}

// TestDriverFailure_PartialApprovalCycle pins the change-tracking
// half of the contract end-to-end on a fake-mounted driver: write a
// file directly to the upper layer (representing what an agent would
// write through the merged view), GetChangedFiles should report it as
// Added, RemoveFromUpper drops it, GetChangedFiles is then empty.
//
// This runs against every overlay flavor since they share
// changedetect.OverlayStrategy + removeFromUpperSecure, but the
// parameterization keeps the contract observable per-flavor.
func TestDriverFailure_PartialApprovalCycle(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.LowerDir = paths.LowerDir
			fx.sandbox.UpperDir = paths.UpperDir
			fx.sandbox.WorkDir = paths.WorkDir
			fx.sandbox.MergedDir = paths.MergedDir
			fx.sandbox.HomeMergedDir = paths.HomeMergedDir

			// Write a file into the upper layer.
			added := filepath.Join(fx.sandbox.UpperDir, "new.txt")
			if err := os.WriteFile(added, []byte("hi"), 0o644); err != nil {
				t.Fatalf("write upper: %v", err)
			}

			changes, err := fx.driver.GetChangedFiles(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("GetChangedFiles: %v", err)
			}
			if !containsAdded(changes, "new.txt") {
				t.Errorf("expected new.txt as Added, got %v", changes)
			}

			if err := fx.driver.RemoveFromUpper(context.Background(), fx.sandbox, "new.txt"); err != nil {
				t.Fatalf("RemoveFromUpper: %v", err)
			}
			changes, err = fx.driver.GetChangedFiles(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("GetChangedFiles after remove: %v", err)
			}
			for _, c := range changes {
				if c.FilePath == "new.txt" {
					t.Errorf("new.txt still listed after RemoveFromUpper: %v", c)
				}
			}
		})
	}
}

// TestDriverFailure_GetChangedFilesAfterUnmount pins: Unmount tears
// down the merged view but leaves the upper layer intact, so
// GetChangedFiles still returns the recorded changes. This protects
// the partial-approval flow when an operator unmounts a sandbox
// before approving — the API still has to be able to enumerate the
// pending changes for the approval UI.
func TestDriverFailure_GetChangedFilesAfterUnmount(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.LowerDir = paths.LowerDir
			fx.sandbox.UpperDir = paths.UpperDir
			fx.sandbox.WorkDir = paths.WorkDir
			fx.sandbox.MergedDir = paths.MergedDir
			fx.sandbox.HomeMergedDir = paths.HomeMergedDir

			if err := os.WriteFile(filepath.Join(fx.sandbox.UpperDir, "lingering.txt"), []byte("x"), 0o644); err != nil {
				t.Fatalf("write upper: %v", err)
			}
			if err := fx.driver.Unmount(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("Unmount: %v", err)
			}

			changes, err := fx.driver.GetChangedFiles(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("GetChangedFiles after unmount: %v", err)
			}
			if !containsAdded(changes, "lingering.txt") {
				t.Errorf("expected lingering.txt as Added after Unmount, got %v", changes)
			}
		})
	}
}

// TestDriverFailure_RemoveFromUpperBlocksTraversal pins the security
// invariant that's defense-in-depth on top of the path validation in
// removeFromUpperSecure. The driver wrapper must surface the error so
// callers cannot escape the upper dir via "../etc/passwd"-style paths.
func TestDriverFailure_RemoveFromUpperBlocksTraversal(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.UpperDir = paths.UpperDir
			fx.sandbox.MergedDir = paths.MergedDir

			if err := fx.driver.RemoveFromUpper(context.Background(), fx.sandbox, "../etc/passwd"); err == nil {
				t.Error("expected path traversal to be rejected")
			}
		})
	}
}

// TestDriverFailure_VerifyMountIntegrityAfterUnmount pins: after
// Unmount, the MountVerifier surface returns an error (not nil). This
// is what the auto-heal loop keys off of to detect stale sandboxes.
func TestDriverFailure_VerifyMountIntegrityAfterUnmount(t *testing.T) {
	for _, flavor := range overlayFlavors() {
		t.Run(flavor.name, func(t *testing.T) {
			fx := newFailureFixture(t, flavor)
			paths, err := fx.driver.Mount(context.Background(), fx.sandbox)
			if err != nil {
				t.Fatalf("Mount: %v", err)
			}
			fx.sandbox.UpperDir = paths.UpperDir
			fx.sandbox.MergedDir = paths.MergedDir

			if err := fx.driver.VerifyMountIntegrity(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("VerifyMountIntegrity pre-Unmount: %v", err)
			}
			if err := fx.driver.Unmount(context.Background(), fx.sandbox); err != nil {
				t.Fatalf("Unmount: %v", err)
			}
			if err := fx.driver.VerifyMountIntegrity(context.Background(), fx.sandbox); err == nil {
				t.Error("expected VerifyMountIntegrity to return an error after Unmount")
			}
		})
	}
}

// =============================================================================
// CopyDriver failure-mode coverage
// =============================================================================

// TestCopyDriverFailure_CleanupIdempotent pins: a second Cleanup on the
// same sandbox is a no-op; CleanupOrphan twice in a row is a no-op.
// The copy driver doesn't take a Mounter so Mounter-injection is not
// applicable; the coverage focus is filesystem idempotency.
func TestCopyDriverFailure_CleanupIdempotent(t *testing.T) {
	tmp := t.TempDir()
	scope := filepath.Join(tmp, "scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	if err := os.WriteFile(filepath.Join(scope, "file.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	deps, _, _ := fakeDeps()
	d := NewCopyDriver(Config{BaseDir: filepath.Join(tmp, "base")}, deps)
	id := uuid.New()
	sb := &types.Sandbox{ID: id, ScopePath: scope, ProjectRoot: scope}

	if _, err := d.Mount(context.Background(), sb); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if err := d.Cleanup(context.Background(), sb); err != nil {
		t.Fatalf("Cleanup #1: %v", err)
	}
	if err := d.Cleanup(context.Background(), sb); err != nil {
		t.Errorf("Cleanup #2 should be a no-op, got %v", err)
	}
	if err := d.CleanupOrphan(context.Background(), id); err != nil {
		t.Errorf("CleanupOrphan #1 (already gone): %v", err)
	}
	if err := d.CleanupOrphan(context.Background(), id); err != nil {
		t.Errorf("CleanupOrphan #2: %v", err)
	}
}

// TestCopyDriverFailure_MissingScopePath pins: a non-existent scope
// path causes Mount to error and roll back the sandbox dir.
func TestCopyDriverFailure_MissingScopePath(t *testing.T) {
	tmp := t.TempDir()
	missingScope := filepath.Join(tmp, "does-not-exist")
	deps, _, _ := fakeDeps()
	d := NewCopyDriver(Config{BaseDir: filepath.Join(tmp, "base")}, deps)
	id := uuid.New()
	sb := &types.Sandbox{ID: id, ScopePath: missingScope, ProjectRoot: missingScope}

	_, err := d.Mount(context.Background(), sb)
	if err == nil {
		t.Fatal("expected Mount to fail with missing scope")
	}
	sandboxDir := filepath.Join(tmp, "base", id.String())
	if _, statErr := os.Stat(sandboxDir); !os.IsNotExist(statErr) {
		t.Errorf("expected sandbox dir cleaned up, stat err=%v", statErr)
	}
}

// TestCopyDriverFailure_UnsupportedHomeOverlayState pins: the copy
// driver always reports HomeOverlayState=Unsupported after Mount,
// regardless of $HOME. The handler-side gate (HomeOverlayRequirement)
// keys off this exact value to refuse vrooli-aware exec.
func TestCopyDriverFailure_UnsupportedHomeOverlayState(t *testing.T) {
	tmp := t.TempDir()
	scope := filepath.Join(tmp, "scope")
	if err := os.MkdirAll(scope, 0o755); err != nil {
		t.Fatalf("mkdir scope: %v", err)
	}
	t.Setenv("HOME", filepath.Join(tmp, "home"))
	if err := os.MkdirAll(filepath.Join(tmp, "home"), 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	deps, _, _ := fakeDeps()
	d := NewCopyDriver(Config{BaseDir: filepath.Join(tmp, "base")}, deps)
	id := uuid.New()
	sb := &types.Sandbox{ID: id, ScopePath: scope, ProjectRoot: scope}

	if _, err := d.Mount(context.Background(), sb); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if sb.HomeOverlayState != types.HomeOverlayUnsupported {
		t.Errorf("HomeOverlayState: got %q, want %q", sb.HomeOverlayState, types.HomeOverlayUnsupported)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// containsAdded reports whether changes contains a FileChange for
// filePath with ChangeType=Added.
func containsAdded(changes []*types.FileChange, filePath string) bool {
	for _, c := range changes {
		if c == nil {
			continue
		}
		if c.FilePath == filePath && c.ChangeType == types.ChangeTypeAdded {
			return true
		}
	}
	return false
}

// =============================================================================
// Compile-time anchor — keeps the clock import grounded and documents
// that the failure tests run against the production Clock.
// =============================================================================

var _ clock.Clock = clock.System{}
