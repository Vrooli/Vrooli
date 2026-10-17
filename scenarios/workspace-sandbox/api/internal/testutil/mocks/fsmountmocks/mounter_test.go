package fsmountmocks_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/testutil/mocks/fsmountmocks"
)

func opts(target string) fsmount.MountOpts {
	return fsmount.MountOpts{
		Backend: fsmount.BackendKernelOverlay,
		Lower:   "/lower",
		Upper:   "/upper",
		Work:    "/work",
		Merged:  target,
	}
}

func TestFakeMounter_MountTracksTarget(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	if m.IsMounted("/x") {
		t.Error("expected unmounted by default")
	}
	if err := m.Mount(context.Background(), opts("/x")); err != nil {
		t.Fatalf("Mount: %v", err)
	}
	if !m.IsMounted("/x") {
		t.Error("expected mounted after Mount")
	}
}

func TestFakeMounter_UnmountClearsTarget(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	_ = m.Mount(context.Background(), opts("/x"))
	if err := m.Unmount(context.Background(), "/x", false); err != nil {
		t.Fatalf("Unmount: %v", err)
	}
	if m.IsMounted("/x") {
		t.Error("expected unmounted after Unmount")
	}
}

func TestFakeMounter_RecordsCallsInOrder(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	_ = m.Mount(context.Background(), opts("/a"))
	_ = m.Mount(context.Background(), opts("/b"))
	_ = m.Unmount(context.Background(), "/a", true)
	if len(m.MountCalls) != 2 {
		t.Errorf("MountCalls: got %d, want 2", len(m.MountCalls))
	}
	if len(m.UnmountCalls) != 1 {
		t.Errorf("UnmountCalls: got %d, want 1", len(m.UnmountCalls))
	}
	if m.UnmountCalls[0].Target != "/a" || !m.UnmountCalls[0].Lazy {
		t.Errorf("UnmountCall: %+v", m.UnmountCalls[0])
	}
}

func TestFakeMounter_RejectsUnsetBackend(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	err := m.Mount(context.Background(), fsmount.MountOpts{
		Lower: "/l", Upper: "/u", Work: "/w", Merged: "/m",
	})
	if err == nil || !strings.Contains(err.Error(), "Backend") {
		t.Errorf("expected Backend error, got %v", err)
	}
}

func TestFakeMounter_PerTargetMountErr(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	m.SetMountErrFor("/bad", errors.New("EPERM"))
	if err := m.Mount(context.Background(), opts("/good")); err != nil {
		t.Errorf("good mount: %v", err)
	}
	if err := m.Mount(context.Background(), opts("/bad")); err == nil || !strings.Contains(err.Error(), "EPERM") {
		t.Errorf("bad mount err=%v, want EPERM", err)
	}
}

func TestFakeMounter_GlobalMountErr(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	m.SetMountErr(errors.New("disk full"))
	if err := m.Mount(context.Background(), opts("/x")); err == nil {
		t.Errorf("expected disk full error, got nil")
	}
	m.SetMountErr(nil)
	if err := m.Mount(context.Background(), opts("/x")); err != nil {
		t.Errorf("after clearing: %v", err)
	}
}

func TestFakeMounter_AddMountPoint_VisibleViaIsMountPoint(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	if m.IsMountPoint("/leftover") {
		t.Error("unexpected default mount")
	}
	m.AddMountPoint("/leftover")
	if !m.IsMountPoint("/leftover") {
		t.Error("expected mounted after AddMountPoint")
	}
	if err := m.Unmount(context.Background(), "/leftover", false); err != nil {
		t.Errorf("Unmount: %v", err)
	}
	if m.IsMountPoint("/leftover") {
		t.Error("expected unmounted after Unmount")
	}
}

func TestFakeMounter_PerTargetUnmountErr(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	_ = m.Mount(context.Background(), opts("/x"))
	m.SetUnmountErrFor("/x", errors.New("busy"))
	if err := m.Unmount(context.Background(), "/x", false); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Errorf("err=%v, want busy", err)
	}
}

// TestFakeMounter_SilentMount models the "fuse-overlayfs forks-and-dies"
// regression: Mount returns nil but no kernel/userspace mount actually
// attached. The fake's silent-mount knob keeps mountErr clear yet leaves
// the merged target outside the mounted set so verifyMounted (in driver
// helpers) catches the discrepancy.
func TestFakeMounter_SilentMount(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	m.SetSilentMountFor("/silent")
	if err := m.Mount(context.Background(), opts("/silent")); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if m.IsMountPoint("/silent") {
		t.Error("silent mount should NOT register as a mount point")
	}
	// A non-silent target on the same fake must still register normally.
	if err := m.Mount(context.Background(), opts("/normal")); err != nil {
		t.Fatalf("normal mount: %v", err)
	}
	if !m.IsMountPoint("/normal") {
		t.Error("normal mount should register as a mount point")
	}
	// Clearing the silent set restores normal behavior.
	m.SetSilentMountFor("")
	if err := m.Mount(context.Background(), opts("/silent")); err != nil {
		t.Fatalf("after clear: %v", err)
	}
	if !m.IsMountPoint("/silent") {
		t.Error("after SetSilentMountFor(\"\") the silent target should mount normally")
	}
}

func TestFakeMounter_Reset_KeepsErrConfig(t *testing.T) {
	m := fsmountmocks.NewFakeMounter()
	m.SetMountErr(errors.New("boom"))
	_ = m.Mount(context.Background(), opts("/x"))
	m.Reset()
	if len(m.MountCalls) != 0 {
		t.Error("Reset should clear calls")
	}
	if err := m.Mount(context.Background(), opts("/x")); err == nil {
		t.Error("Reset should keep mount error config")
	}
}
