package backend

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"web-console/internal/pty"
)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := New()

	desc := Descriptor{
		ID:          Standard,
		DisplayName: "Standard",
		Available:   true,
	}
	reg.Register(desc, nil)

	got, ok := reg.Get(Standard)
	if !ok {
		t.Fatal("expected to find registered backend")
	}
	if got.DisplayName != "Standard" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Standard")
	}

	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("expected false for unregistered backend")
	}
}

func TestResolveAutoAndTmuxProbeHelpers(t *testing.T) {
	reg := New()
	reg.Register(Descriptor{ID: Standard, Available: true}, nil)
	if got := reg.ResolveAuto(); got != Standard {
		t.Fatalf("ResolveAuto without persistent = %q", got)
	}
	reg.Register(Descriptor{ID: Persistent, Available: true}, nil)
	if got := reg.ResolveAuto(); got != Persistent {
		t.Fatalf("ResolveAuto with persistent = %q", got)
	}

	t.Setenv("WC_TMUX_SOCKET", filepath.Join(t.TempDir(), "nested", "socket"))
	path := tmuxProbeSocketPath()
	if path == "" {
		t.Fatal("tmux probe path is empty")
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("probe directory was not created: %v", err)
	}
	old := tmuxProbeCommand
	t.Cleanup(func() { tmuxProbeCommand = old })
	tmuxProbeCommand = func(string, ...string) error { return nil }
	if reason := runTmuxProbeCommand("tmux", "-V"); reason != "" {
		t.Fatalf("successful probe command returned %q", reason)
	}
	if available, reason := cacheTmuxProbe("test-version", true, ""); !available || reason != "" {
		t.Fatalf("cache success = %v/%q", available, reason)
	}
	if available, reason := cacheTmuxProbe("test-version-fail", false, "failed"); available || reason != "failed" {
		t.Fatalf("cache failure = %v/%q", available, reason)
	}
	tmuxProbeCommand = func(string, ...string) error { return errors.New("no") }
	if reason := runTmuxProbeCommand("tmux", "send-keys"); reason == "" {
		t.Fatal("failed probe command returned no reason")
	}
}

func TestRegistry_Factory(t *testing.T) {
	reg := New()

	called := false
	factory := pty.Factory(func(pty.LaunchSpec) (pty.PTY, error) {
		called = true
		return nil, nil
	})
	reg.Register(Descriptor{ID: "test", Available: true}, factory)

	f, ok := reg.Factory("test")
	if !ok {
		t.Fatal("expected factory to exist")
	}
	_, _ = f(pty.LaunchSpec{})
	if !called {
		t.Error("factory was not called")
	}

	if _, ok := reg.Factory("missing"); ok {
		t.Error("expected false for missing backend factory")
	}
}

func TestRegistry_Available_PreservesOrder(t *testing.T) {
	reg := New()

	reg.Register(Descriptor{ID: "b", DisplayName: "B", Available: true}, nil)
	reg.Register(Descriptor{ID: "a", DisplayName: "A", Available: true}, nil)
	reg.Register(Descriptor{ID: "c", DisplayName: "C", Available: false}, nil)

	all := reg.Available()
	if len(all) != 3 {
		t.Fatalf("expected 3 backends, got %d", len(all))
	}
	if all[0].ID != "b" || all[1].ID != "a" || all[2].ID != "c" {
		t.Errorf("order = [%s, %s, %s], want [b, a, c]", all[0].ID, all[1].ID, all[2].ID)
	}
}

func TestRegistry_IsAvailable(t *testing.T) {
	reg := New()

	reg.Register(Descriptor{ID: "up", Available: true}, nil)
	reg.Register(Descriptor{ID: "down", Available: false, Reason: "not installed"}, nil)

	if !reg.IsAvailable("up") {
		t.Error("expected 'up' to be available")
	}
	if reg.IsAvailable("down") {
		t.Error("expected 'down' to be unavailable")
	}
	if reg.IsAvailable("missing") {
		t.Error("expected unregistered backend to be unavailable")
	}
}

func TestRegistry_Register_OverwritePreservesOrder(t *testing.T) {
	reg := New()

	reg.Register(Descriptor{ID: "x", DisplayName: "First"}, nil)
	reg.Register(Descriptor{ID: "y", DisplayName: "Second"}, nil)
	reg.Register(Descriptor{ID: "x", DisplayName: "Updated"}, nil)

	all := reg.Available()
	if len(all) != 2 {
		t.Fatalf("expected 2 backends (no duplicate), got %d", len(all))
	}
	if all[0].ID != "x" || all[0].DisplayName != "Updated" {
		t.Errorf("first = %s/%s, want x/Updated", all[0].ID, all[0].DisplayName)
	}
}
