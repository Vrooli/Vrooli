package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// --- Layout handlers ---

func TestHandleGetLayout_Empty(t *testing.T) {
	srv := newFakeTestServer()
	req := httptest.NewRequest("GET", "/api/v1/workspace/layout", nil)
	rec := httptest.NewRecorder()

	srv.handleGetLayout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp LayoutResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Panes) != 0 {
		t.Fatalf("expected 0 panes, got %d", len(resp.Panes))
	}
	if len(resp.Groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(resp.Groups))
	}
}

func TestHandleGetLayout_WithPanes(t *testing.T) {
	srv := newFakeTestServer()

	// Insert panes via store
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "bash", SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "node", SortOrder: 0, IsActive: true})

	req := httptest.NewRequest("GET", "/api/v1/workspace/layout", nil)
	rec := httptest.NewRecorder()
	srv.handleGetLayout(rec, req)

	var resp LayoutResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Panes) != 2 {
		t.Fatalf("expected 2 panes, got %d", len(resp.Panes))
	}
	// Should be ordered by sort_order: s2 (0) then s1 (1)
	if resp.Panes[0].SessionID != "s2" {
		t.Errorf("expected first pane s2, got %s", resp.Panes[0].SessionID)
	}
	if resp.ActivePane != "s2" {
		t.Errorf("expected active_pane s2, got %s", resp.ActivePane)
	}
}

func TestHandleSaveLayout_Reorder(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{SessionID: "a", Name: "a", SortOrder: 0})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "b", Name: "b", SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "c", Name: "c", SortOrder: 2})

	body := `{"active_pane":"b","pane_order":["c","a","b"]}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/layout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.handleSaveLayout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify order was updated
	layout, _ := store.GetLayout()
	if layout.Panes[0].SessionID != "c" {
		t.Errorf("expected first pane c, got %s", layout.Panes[0].SessionID)
	}
	if layout.Panes[1].SessionID != "a" {
		t.Errorf("expected second pane a, got %s", layout.Panes[1].SessionID)
	}
	if layout.ActivePane != "b" {
		t.Errorf("expected active pane b, got %s", layout.ActivePane)
	}
}

// --- Pane handlers ---

func TestHandleUpdatePane_Create(t *testing.T) {
	srv := newFakeTestServer()
	body := `{"name":"custom","header_color":"#ff0000","theme_id":"dracula","font_size":16}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/panes/s1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"session_id": "s1"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdatePane(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var pane WorkspacePane
	if err := json.NewDecoder(rec.Body).Decode(&pane); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if pane.Name != "custom" {
		t.Errorf("expected name custom, got %s", pane.Name)
	}
	if pane.HeaderColor != "#ff0000" {
		t.Errorf("expected color #ff0000, got %s", pane.HeaderColor)
	}
	if pane.ThemeID != "dracula" {
		t.Errorf("expected theme dracula, got %s", pane.ThemeID)
	}
	if pane.FontSize != 16 {
		t.Errorf("expected font_size 16, got %d", pane.FontSize)
	}
}

func TestHandleUpdatePane_PartialUpdate(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)
	_ = store.UpsertPane(&WorkspacePane{
		SessionID: "s1", Name: "original", HeaderColor: "#00ff00",
		ThemeID: "monokai", FontSize: 14,
	})

	// Only update color
	body := `{"header_color":"#ff0000"}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/panes/s1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"session_id": "s1"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdatePane(rec, req)

	var pane WorkspacePane
	_ = json.NewDecoder(rec.Body).Decode(&pane)
	if pane.Name != "original" {
		t.Errorf("name should be preserved, got %s", pane.Name)
	}
	if pane.HeaderColor != "#ff0000" {
		t.Errorf("color should be updated, got %s", pane.HeaderColor)
	}
	if pane.ThemeID != "monokai" {
		t.Errorf("theme should be preserved, got %s", pane.ThemeID)
	}
}

func TestHandleDeletePane_Idempotent(t *testing.T) {
	srv := newFakeTestServer()

	req := httptest.NewRequest("DELETE", "/api/v1/workspace/panes/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"session_id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDeletePane(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

// --- Group handlers ---

func TestHandleCreateGroup(t *testing.T) {
	srv := newFakeTestServer()
	body := `{"name":"Dev servers","color":"#e11d48"}`
	req := httptest.NewRequest("POST", "/api/v1/workspace/groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateGroup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var group TabGroup
	if err := json.NewDecoder(rec.Body).Decode(&group); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if group.Name != "Dev servers" {
		t.Errorf("expected name 'Dev servers', got %s", group.Name)
	}
	if group.Color != "#e11d48" {
		t.Errorf("expected color #e11d48, got %s", group.Color)
	}
	if group.ID == "" {
		t.Error("expected generated ID")
	}
}

func TestHandleCreateGroup_Defaults(t *testing.T) {
	srv := newFakeTestServer()
	body := `{}`
	req := httptest.NewRequest("POST", "/api/v1/workspace/groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateGroup(rec, req)

	var group TabGroup
	_ = json.NewDecoder(rec.Body).Decode(&group)
	if group.Name != "Group" {
		t.Errorf("expected default name 'Group', got %s", group.Name)
	}
	if group.Color != "#3b82f6" {
		t.Errorf("expected default color #3b82f6, got %s", group.Color)
	}
}

func TestHandleUpdateGroup(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Old name", "#000000")

	body := `{"name":"New name","is_collapsed":true}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/groups/"+group.ID, strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": group.ID})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateGroup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated TabGroup
	_ = json.NewDecoder(rec.Body).Decode(&updated)
	if updated.Name != "New name" {
		t.Errorf("expected name 'New name', got %s", updated.Name)
	}
	if !updated.IsCollapsed {
		t.Error("expected is_collapsed to be true")
	}
	if updated.Color != "#000000" {
		t.Errorf("color should be preserved, got %s", updated.Color)
	}
}

func TestHandleUpdateGroup_NotFound(t *testing.T) {
	srv := newFakeTestServer()
	body := `{"name":"x"}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/groups/nonexistent", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdateGroup(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleDeleteGroup_Idempotent(t *testing.T) {
	srv := newFakeTestServer()
	req := httptest.NewRequest("DELETE", "/api/v1/workspace/groups/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleDeleteGroup(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestHandleDeleteGroup_ClearsGroupOnPanes(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)

	group, _ := store.CreateGroup("Test", "#ff0000")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "a", GroupID: group.ID})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "b", GroupID: group.ID})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s3", Name: "c"}) // not in group

	req := httptest.NewRequest("DELETE", "/api/v1/workspace/groups/"+group.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": group.ID})
	rec := httptest.NewRecorder()
	srv.handleDeleteGroup(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Verify panes had their group_id cleared
	layout, _ := store.GetLayout()
	for _, p := range layout.Panes {
		if p.GroupID != "" {
			t.Errorf("pane %s still has group_id %s", p.SessionID, p.GroupID)
		}
	}
}

func TestHandleUpdatePane_SetGroup(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Dev", "#3b82f6")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "bash"})

	body := `{"group_id":"` + group.ID + `"}`
	req := httptest.NewRequest("PUT", "/api/v1/workspace/panes/s1", strings.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"session_id": "s1"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleUpdatePane(rec, req)

	var pane WorkspacePane
	_ = json.NewDecoder(rec.Body).Decode(&pane)
	if pane.GroupID != group.ID {
		t.Errorf("expected group_id %s, got %s", group.ID, pane.GroupID)
	}
}

func TestHandleGetLayout_WithGroups(t *testing.T) {
	srv := newFakeTestServer()
	store := srv.workspace.(*MemWorkspaceStore)
	group, _ := store.CreateGroup("Servers", "#e11d48")
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s1", Name: "web", GroupID: group.ID, SortOrder: 0})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s2", Name: "db", GroupID: group.ID, SortOrder: 1})
	_ = store.UpsertPane(&WorkspacePane{SessionID: "s3", Name: "logs", SortOrder: 2})

	req := httptest.NewRequest("GET", "/api/v1/workspace/layout", nil)
	rec := httptest.NewRecorder()
	srv.handleGetLayout(rec, req)

	var resp LayoutResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.Panes) != 3 {
		t.Fatalf("expected 3 panes, got %d", len(resp.Panes))
	}
	if len(resp.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(resp.Groups))
	}
	if resp.Groups[0].Name != "Servers" {
		t.Errorf("expected group name 'Servers', got %s", resp.Groups[0].Name)
	}

	// Check pane-group association
	groupedCount := 0
	for _, p := range resp.Panes {
		if p.GroupID == group.ID {
			groupedCount++
		}
	}
	if groupedCount != 2 {
		t.Errorf("expected 2 panes in group, got %d", groupedCount)
	}
}
