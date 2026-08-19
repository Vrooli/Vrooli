package main

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	shortcutsH "web-console/handlers/shortcuts"

	shortcutsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts"
)

// [REQ:P1-002a] Shortcut Profile Storage - store tests

func TestShortcutProfileStore_DefaultProfile(t *testing.T) {
	store := NewShortcutProfileStore()
	profiles := store.List(context.Background())
	if len(profiles) != 1 {
		t.Fatalf("expected 1 default profile, got %d", len(profiles))
	}
	if profiles[0].ID != "default" {
		t.Errorf("expected default profile ID, got %q", profiles[0].ID)
	}
	if profiles[0].Scope != "service" {
		t.Errorf("expected service scope, got %q", profiles[0].Scope)
	}
	if len(profiles[0].Shortcuts) != 8 {
		t.Errorf("expected 8 default shortcuts, got %d", len(profiles[0].Shortcuts))
	}
	if got := profiles[0].Shortcuts[0].Command; got != "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions" {
		t.Errorf("default Claude shortcut = %q, want governed project invocation", got)
	}
	if got := profiles[0].Shortcuts[4].Command; got != "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent claude -- --dangerously-skip-permissions; fi; exec vrooli agent launch --runner claude --arg=--dangerously-skip-permissions" {
		t.Errorf("attributed Claude shortcut = %q, want PATH preflight", got)
	}
}

func TestShortcutProfileStore_Upsert(t *testing.T) {
	store := NewShortcutProfileStore()

	shortcuts := []ShortcutEntry{{Label: "Test", Command: "echo test"}}
	profile := store.Upsert(context.Background(), "custom", "workspace", "Custom", shortcuts)

	if profile.ID != "custom" {
		t.Errorf("expected ID 'custom', got %q", profile.ID)
	}
	if profile.Scope != "workspace" {
		t.Errorf("expected scope 'workspace', got %q", profile.Scope)
	}
	if len(profile.Shortcuts) != 1 {
		t.Errorf("expected 1 shortcut, got %d", len(profile.Shortcuts))
	}

	// Update existing
	updated := store.Upsert(context.Background(), "custom", "parent", "Updated", []ShortcutEntry{
		{Label: "A", Command: "a"},
		{Label: "B", Command: "b"},
	})
	if updated.Scope != "parent" {
		t.Errorf("expected updated scope 'parent', got %q", updated.Scope)
	}
	if len(updated.Shortcuts) != 2 {
		t.Errorf("expected 2 shortcuts after update, got %d", len(updated.Shortcuts))
	}
}

func TestShortcutProfileStore_Effective(t *testing.T) {
	store := NewShortcutProfileStore()

	// Default service profile exists with 4 shortcuts
	eff := store.Effective(context.Background())
	if len(eff) != 4 {
		t.Fatalf("expected default service profile shortcuts, got %d", len(eff))
	}

	// Add workspace-scope profile — should win over service
	store.Upsert(context.Background(), "ws", "workspace", "WS", []ShortcutEntry{
		{Label: "WS1", Command: "ws1"},
	})
	eff = store.Effective(context.Background())
	if len(eff) != 1 || eff[0].Label != "WS1" {
		t.Errorf("expected workspace profile to win, got %v", eff)
	}

	// Add parent-scope profile — should win over workspace
	store.Upsert(context.Background(), "parent", "parent", "P", []ShortcutEntry{
		{Label: "Parent1", Command: "p1"},
		{Label: "Parent2", Command: "p2"},
	})
	eff = store.Effective(context.Background())
	if len(eff) != 2 || eff[0].Label != "Parent1" {
		t.Errorf("expected parent profile to win, got %v", eff)
	}
}

func TestShortcutProfileStore_EffectiveEmpty(t *testing.T) {
	store := NewShortcutProfileStore()
	store.Delete(context.Background(), "default")
	eff := store.Effective(context.Background())
	if len(eff) != 4 {
		t.Errorf("expected default shortcuts fallback when no profiles, got %d", len(eff))
	}
}

// --- adapter validation (replaces the old REST validateShortcutProfile) ---

func TestShortcutsAdapter_UpsertValidation(t *testing.T) {
	srv := newFakeTestServer()
	a := newShortcutsAdapter(srv)

	tests := []struct {
		name    string
		req     shortcutsH.UpsertRequest
		wantErr bool
	}{
		{"valid", shortcutsH.UpsertRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []shortcutsH.Shortcut{{Label: "A", Command: "a"}}}, false},
		{"empty id", shortcutsH.UpsertRequest{Scope: "service", Name: "X"}, true},
		{"bad scope", shortcutsH.UpsertRequest{ID: "x", Scope: "invalid", Name: "X"}, true},
		{"empty name", shortcutsH.UpsertRequest{ID: "x", Scope: "service"}, true},
		{"empty label", shortcutsH.UpsertRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []shortcutsH.Shortcut{{Command: "a"}}}, true},
		{"empty command", shortcutsH.UpsertRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []shortcutsH.Shortcut{{Label: "A"}}}, true},
		{"empty shortcuts ok", shortcutsH.UpsertRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: nil}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := a.Upsert(context.Background(), tt.req)
			if tt.wantErr && err == nil {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, shortcutsH.ErrInvalidArgument) {
				t.Errorf("expected ErrInvalidArgument, got %v", err)
			}
		})
	}
}

// --- Connect handler tests (replace the old REST handler tests) ---

func newShortcutsConnectHandler(t *testing.T) (*shortcutsConnectTestHarness, *Server) {
	srv := newFakeTestServer()
	h := shortcutsH.NewConnectHandler(shortcutsH.Deps{Service: newShortcutsAdapter(srv)})
	return &shortcutsConnectTestHarness{h: h}, srv
}

type shortcutsConnectTestHarness struct {
	h interface {
		GetEffective(context.Context, *connect.Request[shortcutsv1.GetEffectiveRequest]) (*connect.Response[shortcutsv1.GetEffectiveResponse], error)
		ListProfiles(context.Context, *connect.Request[shortcutsv1.ListProfilesRequest]) (*connect.Response[shortcutsv1.ListProfilesResponse], error)
		UpsertProfile(context.Context, *connect.Request[shortcutsv1.UpsertProfileRequest]) (*connect.Response[shortcutsv1.UpsertProfileResponse], error)
		DeleteProfile(context.Context, *connect.Request[shortcutsv1.DeleteProfileRequest]) (*connect.Response[shortcutsv1.DeleteProfileResponse], error)
	}
}

func TestConnect_GetEffective(t *testing.T) {
	harness, _ := newShortcutsConnectHandler(t)
	resp, err := harness.h.GetEffective(context.Background(), connect.NewRequest(&shortcutsv1.GetEffectiveRequest{}))
	if err != nil {
		t.Fatalf("GetEffective: %v", err)
	}
	if len(resp.Msg.GetShortcuts()) != 8 {
		t.Errorf("expected 8 shortcuts, got %d", len(resp.Msg.GetShortcuts()))
	}
	if got := resp.Msg.GetShortcuts()[0].GetCommand(); got != "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions" {
		t.Errorf("effective Claude shortcut = %q, want governed project invocation", got)
	}
}

func TestConnect_UpsertProfile(t *testing.T) {
	harness, _ := newShortcutsConnectHandler(t)
	resp, err := harness.h.UpsertProfile(context.Background(), connect.NewRequest(&shortcutsv1.UpsertProfileRequest{
		Id:    "ws",
		Scope: "workspace",
		Name:  "Workspace",
		Shortcuts: []*shortcutsv1.Shortcut{
			{Label: "Test", Command: "echo test"},
		},
	}))
	if err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}
	if resp.Msg.GetProfile().GetId() != "ws" {
		t.Errorf("expected ID 'ws', got %q", resp.Msg.GetProfile().GetId())
	}
}

func TestConnect_UpsertProfile_Invalid(t *testing.T) {
	harness, _ := newShortcutsConnectHandler(t)
	_, err := harness.h.UpsertProfile(context.Background(), connect.NewRequest(&shortcutsv1.UpsertProfileRequest{
		Id:    "",
		Scope: "workspace",
		Name:  "Workspace",
	}))
	if err == nil {
		t.Fatal("expected error")
	}
	var connErr *connect.Error
	if !errors.As(err, &connErr) || connErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", err)
	}
}

func TestConnect_DeleteProfile_Idempotent(t *testing.T) {
	harness, _ := newShortcutsConnectHandler(t)
	// Delete existing default
	_, err := harness.h.DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: "default"}))
	if err != nil {
		t.Fatalf("DeleteProfile (existing): %v", err)
	}
	// Delete non-existent — must succeed
	_, err = harness.h.DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: "nonexistent"}))
	if err != nil {
		t.Fatalf("DeleteProfile (idempotent): %v", err)
	}
}

func TestConnect_ListProfiles(t *testing.T) {
	harness, _ := newShortcutsConnectHandler(t)
	resp, err := harness.h.ListProfiles(context.Background(), connect.NewRequest(&shortcutsv1.ListProfilesRequest{}))
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(resp.Msg.GetProfiles()) != 1 {
		t.Errorf("expected 1 profile, got %d", len(resp.Msg.GetProfiles()))
	}
}

// --- Replay / Idempotency Tests ---

// Upsert replay: identical content does not bump UpdatedAt.
func TestProfileUpsert_Replay_TimestampStable(t *testing.T) {
	store := NewShortcutProfileStore()

	shortcuts := []ShortcutEntry{{Label: "Go", Command: "go run"}}
	p1 := store.Upsert(context.Background(), "replay-test", "workspace", "Test", shortcuts)
	firstUpdated := p1.UpdatedAt

	// Replay with identical content — UpdatedAt should not change
	p2 := store.Upsert(context.Background(), "replay-test", "workspace", "Test", shortcuts)
	if p2.UpdatedAt != firstUpdated {
		t.Errorf("replay with same content should preserve UpdatedAt: got %s, want %s", p2.UpdatedAt, firstUpdated)
	}

	// Change content — UpdatedAt should change
	p3 := store.Upsert(context.Background(), "replay-test", "workspace", "Test", []ShortcutEntry{{Label: "Go", Command: "go test"}})
	if p3.UpdatedAt == firstUpdated {
		t.Error("changed content should update UpdatedAt")
	}
}

// Delete replay: deleting the same profile three times succeeds each time.
func TestProfileDelete_Replay_Idempotent(t *testing.T) {
	harness, srv := newShortcutsConnectHandler(t)
	srv.shortcuts.Upsert(context.Background(), "del-test", "workspace", "Del", []ShortcutEntry{{Label: "A", Command: "a"}})

	for i := 0; i < 3; i++ {
		_, err := harness.h.DeleteProfile(context.Background(), connect.NewRequest(&shortcutsv1.DeleteProfileRequest{Id: "del-test"}))
		if err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
}

// [REQ:P1-002a] Performance test - config resolution <50ms
func BenchmarkEffectiveShortcuts(b *testing.B) {
	store := NewShortcutProfileStore()
	store.Upsert(context.Background(), "ws", "workspace", "WS", []ShortcutEntry{
		{Label: "A", Command: "a"},
		{Label: "B", Command: "b"},
		{Label: "C", Command: "c"},
	})
	store.Upsert(context.Background(), "parent", "parent", "P", []ShortcutEntry{
		{Label: "X", Command: "x"},
		{Label: "Y", Command: "y"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Effective(context.Background())
	}
}
