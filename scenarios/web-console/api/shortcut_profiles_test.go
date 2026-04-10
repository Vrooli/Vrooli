package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:P1-002a] Shortcut Profile Storage - store tests

func TestShortcutProfileStore_DefaultProfile(t *testing.T) {
	store := NewShortcutProfileStore()
	profiles := store.List()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 default profile, got %d", len(profiles))
	}
	if profiles[0].ID != "default" {
		t.Errorf("expected default profile ID, got %q", profiles[0].ID)
	}
	if profiles[0].Scope != "service" {
		t.Errorf("expected service scope, got %q", profiles[0].Scope)
	}
	if len(profiles[0].Shortcuts) != 2 {
		t.Errorf("expected 2 default shortcuts, got %d", len(profiles[0].Shortcuts))
	}
}

func TestShortcutProfileStore_Upsert(t *testing.T) {
	store := NewShortcutProfileStore()

	shortcuts := []ShortcutEntry{{Label: "Test", Command: "echo test"}}
	profile := store.Upsert("custom", "workspace", "Custom", shortcuts)

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
	updated := store.Upsert("custom", "parent", "Updated", []ShortcutEntry{
		{Label: "A", Command: "a"},
		{Label: "B", Command: "b"},
	})
	if updated.Scope != "parent" {
		t.Errorf("expected scope 'parent', got %q", updated.Scope)
	}
	if len(updated.Shortcuts) != 2 {
		t.Errorf("expected 2 shortcuts after update, got %d", len(updated.Shortcuts))
	}
}

func TestShortcutProfileStore_Delete(t *testing.T) {
	store := NewShortcutProfileStore()

	if !store.Delete("default") {
		t.Error("expected delete of default to succeed")
	}
	if store.Delete("nonexistent") {
		t.Error("expected delete of nonexistent to fail")
	}
	if len(store.List()) != 0 {
		t.Errorf("expected 0 profiles after delete, got %d", len(store.List()))
	}
}

func TestShortcutProfileStore_Effective(t *testing.T) {
	store := NewShortcutProfileStore()

	// Default (service scope) should be effective
	eff := store.Effective()
	if len(eff) != 2 {
		t.Fatalf("expected 2 effective shortcuts from default, got %d", len(eff))
	}

	// Add a workspace-scoped profile — should win over service
	store.Upsert("ws", "workspace", "Workspace", []ShortcutEntry{
		{Label: "WS Only", Command: "ws-cmd"},
	})
	eff = store.Effective()
	if len(eff) != 1 || eff[0].Label != "WS Only" {
		t.Errorf("expected workspace profile to win, got %v", eff)
	}

	// Add a parent-scoped profile — should win over workspace
	store.Upsert("par", "parent", "Parent", []ShortcutEntry{
		{Label: "Parent1", Command: "p1"},
		{Label: "Parent2", Command: "p2"},
	})
	eff = store.Effective()
	if len(eff) != 2 || eff[0].Label != "Parent1" {
		t.Errorf("expected parent profile to win, got %v", eff)
	}
}

func TestShortcutProfileStore_EffectiveEmpty(t *testing.T) {
	store := NewShortcutProfileStore()
	store.Delete("default")
	eff := store.Effective()
	if len(eff) != 2 {
		t.Errorf("expected default shortcuts fallback when no profiles, got %d", len(eff))
	}
}

func TestValidateShortcutProfile(t *testing.T) {
	tests := []struct {
		name    string
		req     ShortcutProfileRequest
		wantErr bool
	}{
		{"valid", ShortcutProfileRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []ShortcutEntry{{Label: "A", Command: "a"}}}, false},
		{"empty id", ShortcutProfileRequest{Scope: "service", Name: "X"}, true},
		{"bad scope", ShortcutProfileRequest{ID: "x", Scope: "invalid", Name: "X"}, true},
		{"empty name", ShortcutProfileRequest{ID: "x", Scope: "service"}, true},
		{"empty label", ShortcutProfileRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []ShortcutEntry{{Command: "a"}}}, true},
		{"empty command", ShortcutProfileRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: []ShortcutEntry{{Label: "A"}}}, true},
		{"empty shortcuts ok", ShortcutProfileRequest{ID: "x", Scope: "service", Name: "X", Shortcuts: nil}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateShortcutProfile(tt.req)
			if tt.wantErr && msg == "" {
				t.Error("expected validation error, got none")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("unexpected validation error: %s", msg)
			}
		})
	}
}

// [REQ:P1-002a] Shortcut Profile Storage - handler tests

func TestHandleGetEffectiveShortcuts(t *testing.T) {
	srv := newFakeTestServer()
	req := httptest.NewRequest("GET", "/api/v1/shortcuts", nil)
	rec := httptest.NewRecorder()

	srv.handleGetEffectiveShortcuts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var shortcuts []ShortcutEntry
	if err := json.Unmarshal(rec.Body.Bytes(), &shortcuts); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(shortcuts) != 2 {
		t.Errorf("expected 2 shortcuts, got %d", len(shortcuts))
	}
}

func TestHandleUpsertShortcutProfile(t *testing.T) {
	srv := newFakeTestServer()

	body := `{"id":"ws","scope":"workspace","name":"Workspace","shortcuts":[{"label":"Test","command":"echo test"}]}`
	req := httptest.NewRequest("PUT", "/api/v1/shortcuts/profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpsertShortcutProfile(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var profile ShortcutProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if profile.ID != "ws" {
		t.Errorf("expected ID 'ws', got %q", profile.ID)
	}
}

func TestHandleUpsertShortcutProfile_Invalid(t *testing.T) {
	srv := newFakeTestServer()

	body := `{"id":"","scope":"workspace","name":"Workspace"}`
	req := httptest.NewRequest("PUT", "/api/v1/shortcuts/profiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpsertShortcutProfile(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleDeleteShortcutProfile(t *testing.T) {
	srv := newFakeTestServer()

	req := httptest.NewRequest("DELETE", "/api/v1/shortcuts/profiles/default", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "default"})
	rec := httptest.NewRecorder()

	srv.handleDeleteShortcutProfile(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// DELETE is idempotent: deleting a non-existent profile returns 204 (not 404),
// so that retries and replays are safe.
func TestHandleDeleteShortcutProfile_NotFound_Idempotent(t *testing.T) {
	srv := newFakeTestServer()

	req := httptest.NewRequest("DELETE", "/api/v1/shortcuts/profiles/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDeleteShortcutProfile(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 (idempotent delete), got %d", rec.Code)
	}
}

func TestHandleListShortcutProfiles(t *testing.T) {
	srv := newFakeTestServer()
	req := httptest.NewRequest("GET", "/api/v1/shortcuts/profiles", nil)
	rec := httptest.NewRecorder()

	srv.handleListShortcutProfiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var profiles []*ShortcutProfile
	if err := json.Unmarshal(rec.Body.Bytes(), &profiles); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}
}

// --- Replay / Idempotency Tests ---

// Upsert replay: identical content does not bump UpdatedAt.
func TestProfileUpsert_Replay_TimestampStable(t *testing.T) {
	store := NewShortcutProfileStore()

	shortcuts := []ShortcutEntry{{Label: "Go", Command: "go run"}}
	p1 := store.Upsert("replay-test", "workspace", "Test", shortcuts)
	firstUpdated := p1.UpdatedAt

	// Replay with identical content — UpdatedAt should not change
	p2 := store.Upsert("replay-test", "workspace", "Test", shortcuts)
	if p2.UpdatedAt != firstUpdated {
		t.Errorf("replay with same content should preserve UpdatedAt: got %s, want %s", p2.UpdatedAt, firstUpdated)
	}

	// Change content — UpdatedAt should change
	p3 := store.Upsert("replay-test", "workspace", "Test", []ShortcutEntry{{Label: "Go", Command: "go test"}})
	if p3.UpdatedAt == firstUpdated {
		t.Error("changed content should update UpdatedAt")
	}
}

// Delete replay: deleting the same profile twice returns 204 both times.
func TestProfileDelete_Replay_Idempotent(t *testing.T) {
	srv := newFakeTestServer()

	// Create a profile
	srv.shortcuts.Upsert("del-test", "workspace", "Del", []ShortcutEntry{{Label: "A", Command: "a"}})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("DELETE", "/api/v1/shortcuts/profiles/del-test", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "del-test"})
		rec := httptest.NewRecorder()
		srv.handleDeleteShortcutProfile(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("attempt %d: expected 204, got %d", i+1, rec.Code)
		}
	}
}

// [REQ:P1-002a] Performance test - config resolution <50ms
func BenchmarkEffectiveShortcuts(b *testing.B) {
	store := NewShortcutProfileStore()
	// Add profiles at each scope level
	store.Upsert("ws", "workspace", "WS", []ShortcutEntry{{Label: "A", Command: "a"}})
	store.Upsert("par", "parent", "PAR", []ShortcutEntry{{Label: "B", Command: "b"}})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Effective()
	}
}
