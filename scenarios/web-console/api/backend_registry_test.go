package main

import (
	"testing"
)

func TestBackendRegistry_RegisterAndGet(t *testing.T) {
	reg := NewBackendRegistry()

	desc := BackendDescriptor{
		ID:          BackendStandard,
		DisplayName: "Standard",
		Available:   true,
	}
	reg.Register(desc, defaultPTYFactory)

	got, ok := reg.Get(BackendStandard)
	if !ok {
		t.Fatal("expected to find registered backend")
	}
	if got.DisplayName != "Standard" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Standard")
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("expected false for unregistered backend")
	}
}

func TestBackendRegistry_Factory(t *testing.T) {
	reg := NewBackendRegistry()

	called := false
	factory := func(spec SessionLaunchSpec) (PTY, error) {
		called = true
		return nil, nil
	}
	reg.Register(BackendDescriptor{ID: "test", Available: true}, factory)

	f, ok := reg.Factory("test")
	if !ok {
		t.Fatal("expected factory to exist")
	}
	_, _ = f(SessionLaunchSpec{})
	if !called {
		t.Error("factory was not called")
	}

	if _, ok := reg.Factory("missing"); ok {
		t.Error("expected false for missing backend factory")
	}
}

func TestBackendRegistry_Available_PreservesOrder(t *testing.T) {
	reg := NewBackendRegistry()

	reg.Register(BackendDescriptor{ID: "b", DisplayName: "B", Available: true}, nil)
	reg.Register(BackendDescriptor{ID: "a", DisplayName: "A", Available: true}, nil)
	reg.Register(BackendDescriptor{ID: "c", DisplayName: "C", Available: false}, nil)

	all := reg.Available()
	if len(all) != 3 {
		t.Fatalf("expected 3 backends, got %d", len(all))
	}
	if all[0].ID != "b" || all[1].ID != "a" || all[2].ID != "c" {
		t.Errorf("order = [%s, %s, %s], want [b, a, c]", all[0].ID, all[1].ID, all[2].ID)
	}
}

func TestBackendRegistry_IsAvailable(t *testing.T) {
	reg := NewBackendRegistry()

	reg.Register(BackendDescriptor{ID: "up", Available: true}, nil)
	reg.Register(BackendDescriptor{ID: "down", Available: false, Reason: "not installed"}, nil)

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

func TestBackendRegistry_Register_OverwritePreservesOrder(t *testing.T) {
	reg := NewBackendRegistry()

	reg.Register(BackendDescriptor{ID: "x", DisplayName: "First"}, nil)
	reg.Register(BackendDescriptor{ID: "y", DisplayName: "Second"}, nil)
	// Overwrite "x" with new descriptor
	reg.Register(BackendDescriptor{ID: "x", DisplayName: "Updated"}, nil)

	all := reg.Available()
	if len(all) != 2 {
		t.Fatalf("expected 2 backends (no duplicate), got %d", len(all))
	}
	if all[0].ID != "x" || all[0].DisplayName != "Updated" {
		t.Errorf("first = %s/%s, want x/Updated", all[0].ID, all[0].DisplayName)
	}
}
