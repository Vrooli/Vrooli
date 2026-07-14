// Package fsmount — unit tests.
//
// The SystemMounter integration tests are guarded by a "want real
// kernel" environment variable so go test in plain dev environments
// only runs the contract logic against the FakeStarter from procmocks.
// Real-mount tests (TestSystemMounter_*Real*) only run when set.
package fsmount_test

import (
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"

	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
)

func TestBackend_String(t *testing.T) {
	cases := []struct {
		b    fsmount.Backend
		want string
	}{
		{fsmount.BackendKernelOverlay, "kernel-overlay"},
		{fsmount.BackendFuseOverlayfs, "fuse-overlayfs"},
		{fsmount.BackendUnset, "backend(0)"},
	}
	for _, c := range cases {
		if got := c.b.String(); got != c.want {
			t.Errorf("Backend(%d).String()=%q, want %q", int(c.b), got, c.want)
		}
	}
}

func TestNewSystemMounter_RequiresStarter(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil starter")
		}
	}()
	fsmount.NewSystemMounter(nil)
}

func TestSystemMounter_Mount_RejectsUnsetBackend(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetDefault(procmocks.CommandBehavior{})
	m := fsmount.NewSystemMounter(starter)
	err := m.Mount(context.Background(), fsmount.MountOpts{
		Lower:  "/lower",
		Upper:  "/upper",
		Work:   "/work",
		Merged: "/merged",
	})
	if err == nil {
		t.Fatal("expected error for unset backend")
	}
	if !strings.Contains(err.Error(), "Backend") {
		t.Errorf("error mentions Backend? got %v", err)
	}
}

func TestSystemMounter_Mount_RejectsMissingFields(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.SetDefault(procmocks.CommandBehavior{})
	m := fsmount.NewSystemMounter(starter)
	cases := []struct {
		name string
		opts fsmount.MountOpts
	}{
		{"missing lower", fsmount.MountOpts{Backend: fsmount.BackendKernelOverlay, Upper: "/u", Work: "/w", Merged: "/m"}},
		{"missing upper", fsmount.MountOpts{Backend: fsmount.BackendKernelOverlay, Lower: "/l", Work: "/w", Merged: "/m"}},
		{"missing work", fsmount.MountOpts{Backend: fsmount.BackendKernelOverlay, Lower: "/l", Upper: "/u", Merged: "/m"}},
		{"missing merged", fsmount.MountOpts{Backend: fsmount.BackendKernelOverlay, Lower: "/l", Upper: "/u", Work: "/w"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := m.Mount(context.Background(), c.opts); err == nil {
				t.Errorf("expected error for %s", c.name)
			}
		})
	}
}

func TestSystemMounter_FuseOverlayfs_Spawn(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("fuse-overlayfs -o", procmocks.CommandBehavior{
		Exit: process.ProcessExit{ExitCode: 0},
	})
	m := fsmount.NewSystemMounter(starter)
	err := m.Mount(context.Background(), fsmount.MountOpts{
		Backend: fsmount.BackendFuseOverlayfs,
		Lower:   "/lower",
		Upper:   "/upper",
		Work:    "/work",
		Merged:  "/merged",
	})
	if err != nil {
		t.Fatalf("Mount fuse: %v", err)
	}
	calls := starter.MatchedCalls("fuse-overlayfs")
	if len(calls) != 1 {
		t.Fatalf("expected 1 fuse-overlayfs call, got %d", len(calls))
	}
	args := calls[0].Args
	if len(args) < 3 || args[0] != "-o" || args[2] != "/merged" {
		t.Errorf("fuse-overlayfs args: got %v, want [-o <opts> /merged]", args)
	}
	if !strings.Contains(args[1], "lowerdir=/lower") {
		t.Errorf("fuse-overlayfs opts: got %q, want contains lowerdir=/lower", args[1])
	}
}

func TestSystemMounter_FuseOverlayfs_NonZeroExitSurfacesError(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("fuse-overlayfs -o", procmocks.CommandBehavior{
		Exit:   process.ProcessExit{ExitCode: 1},
		Stdout: []byte("permission denied"),
	})
	m := fsmount.NewSystemMounter(starter)
	err := m.Mount(context.Background(), fsmount.MountOpts{
		Backend: fsmount.BackendFuseOverlayfs,
		Lower:   "/lower",
		Upper:   "/upper",
		Work:    "/work",
		Merged:  "/merged",
	})
	if err == nil {
		t.Fatal("expected error from non-zero exit")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error includes stdout? got %v", err)
	}
}

func TestSystemMounter_FuseOverlayfs_StartErrorPropagates(t *testing.T) {
	starter := procmocks.NewFakeStarter()
	starter.AddCommand("fuse-overlayfs -o", procmocks.CommandBehavior{
		StartErr: errors.New("fork failed"),
	})
	m := fsmount.NewSystemMounter(starter)
	err := m.Mount(context.Background(), fsmount.MountOpts{
		Backend: fsmount.BackendFuseOverlayfs,
		Lower:   "/lower",
		Upper:   "/upper",
		Work:    "/work",
		Merged:  "/merged",
	})
	if err == nil || !strings.Contains(err.Error(), "fork failed") {
		t.Errorf("expected fork-failed error, got %v", err)
	}
}

// TestSystemMounter_KernelOverlay_RealMount only runs when the test
// host can actually mount overlayfs. Provides end-to-end coverage of
// the syscall.Mount path. Requires linux + an overlay-capable kernel
// + permissions.
func TestSystemMounter_KernelOverlay_RealMount(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("kernel overlay test requires linux; have %s", runtime.GOOS)
	}
	if os.Getenv("WSB_REAL_OVERLAY_TESTS") != "1" {
		t.Skip("set WSB_REAL_OVERLAY_TESTS=1 to enable real kernel-overlay tests")
	}
	starter := process.NewOSExecStarter()
	m := fsmount.NewSystemMounter(starter)
	if err := fsmount.ProbeKernelOverlayMount(context.Background(), m); err != nil {
		t.Fatalf("ProbeKernelOverlayMount: %v", err)
	}
}

func TestProbeKernelOverlayMount_DelegatesToMounter(t *testing.T) {
	// Use the FakeMounter from fsmountmocks indirectly: the probe makes
	// a Mount + Unmount call against any Mounter, so we can verify
	// behavior with a small in-test fake here.
	rec := &recordMounter{}
	if err := fsmount.ProbeKernelOverlayMount(context.Background(), rec); err != nil {
		t.Fatalf("ProbeKernelOverlayMount: %v", err)
	}
	if rec.mountCalls != 1 {
		t.Errorf("Mount calls: got %d, want 1", rec.mountCalls)
	}
	if rec.unmountCalls != 1 {
		t.Errorf("Unmount calls: got %d, want 1", rec.unmountCalls)
	}
}

func TestProbeKernelOverlayMount_PropagatesMountErr(t *testing.T) {
	rec := &recordMounter{mountErr: errors.New("EPERM")}
	err := fsmount.ProbeKernelOverlayMount(context.Background(), rec)
	if err == nil || !strings.Contains(err.Error(), "EPERM") {
		t.Errorf("expected EPERM error, got %v", err)
	}
}

// recordMounter is a tiny in-test Mounter recorder. Used only for
// ProbeKernelOverlayMount tests to avoid pulling fsmountmocks into
// fsmount's own test (which would create a fsmount→fsmountmocks→fsmount
// cycle in test-only graphs).
type recordMounter struct {
	mountCalls   int
	unmountCalls int
	mountErr     error
}

func (m *recordMounter) Mount(ctx context.Context, opts fsmount.MountOpts) error {
	m.mountCalls++
	return m.mountErr
}

func (m *recordMounter) Unmount(ctx context.Context, target string, lazy bool) error {
	m.unmountCalls++
	return nil
}

func (m *recordMounter) IsMountPoint(path string) bool { return false }

// TestBackendUnsupportedError_IsSentinel verifies the typed error the
// non-Linux mount fast path returns matches ErrBackendUnsupported via
// errors.Is and carries backend/GOOS context in its message. This runs on
// Linux CI (it constructs the error directly rather than depending on the
// !linux build of sysMountOverlay).
func TestBackendUnsupportedError_IsSentinel(t *testing.T) {
	err := &fsmount.BackendUnsupportedError{Backend: "kernel-overlay", GOOS: "darwin"}
	if !errors.Is(err, fsmount.ErrBackendUnsupported) {
		t.Fatalf("errors.Is(err, ErrBackendUnsupported) = false, want true")
	}
	msg := err.Error()
	if !strings.Contains(msg, "kernel-overlay") || !strings.Contains(msg, "darwin") {
		t.Errorf("error message %q missing backend/GOOS context", msg)
	}
}
