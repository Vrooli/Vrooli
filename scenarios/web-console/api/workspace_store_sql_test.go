package main

import (
	"testing"
	"web-console/internal/workspace"
)

func TestSQLStore_GetLayout_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	layout, err := store.GetLayout()
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(layout.Panes) != 0 {
		t.Errorf("expected 0 panes, got %d", len(layout.Panes))
	}
	if len(layout.Groups) != 0 {
		t.Errorf("expected 0 groups, got %d", len(layout.Groups))
	}
	if layout.ActivePane != "" {
		t.Errorf("expected empty active pane, got %q", layout.ActivePane)
	}
}

func TestSQLStore_UpsertAndGet(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	pane := workspace.Pane{
		SessionID:   "sess-1",
		Name:        "my-terminal",
		HeaderColor: "#ff0000",
		ThemeID:     "dracula",
		FontSize:    16,
		SortOrder:   0,
		IsActive:    true,
	}
	if err := store.UpsertPane(pane); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, err := store.GetLayout()
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(layout.Panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(layout.Panes))
	}
	p := layout.Panes[0]
	if p.SessionID != "sess-1" {
		t.Errorf("session_id: got %q", p.SessionID)
	}
	if p.Name != "my-terminal" {
		t.Errorf("name: got %q", p.Name)
	}
	if p.HeaderColor != "#ff0000" {
		t.Errorf("header_color: got %q", p.HeaderColor)
	}
	if p.FontSize != 16 {
		t.Errorf("font_size: got %d", p.FontSize)
	}
	if layout.ActivePane != "sess-1" {
		t.Errorf("active_pane: got %q", layout.ActivePane)
	}
}

func TestSQLStore_UpsertDefaults(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	if err := store.UpsertPane(workspace.Pane{SessionID: "sess-2"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, _ := store.GetLayout()
	p := layout.Panes[0]
	if p.Name != workspace.DefaultPaneName {
		t.Errorf("expected default name %q, got %q", workspace.DefaultPaneName, p.Name)
	}
	if p.FontSize != workspace.DefaultPaneFontSize {
		t.Errorf("expected default font size %d, got %d", workspace.DefaultPaneFontSize, p.FontSize)
	}
}

func TestSQLStore_UpsertIdempotent(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	pane := workspace.Pane{SessionID: "sess-3", Name: "first"}
	if err := store.UpsertPane(pane); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	pane.Name = "updated"
	if err := store.UpsertPane(pane); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	layout, _ := store.GetLayout()
	if len(layout.Panes) != 1 {
		t.Fatalf("expected 1 pane after idempotent upsert, got %d", len(layout.Panes))
	}
	if layout.Panes[0].Name != "updated" {
		t.Errorf("expected updated name, got %q", layout.Panes[0].Name)
	}
}

func TestSQLStore_DeletePane(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	if err := store.UpsertPane(workspace.Pane{SessionID: "sess-4"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}
	if err := store.DeletePane("sess-4"); err != nil {
		t.Fatalf("DeletePane: %v", err)
	}

	layout, _ := store.GetLayout()
	if len(layout.Panes) != 0 {
		t.Errorf("expected 0 panes after delete, got %d", len(layout.Panes))
	}

	if err := store.DeletePane("nonexistent"); err != nil {
		t.Errorf("delete nonexistent should not error: %v", err)
	}
}

func TestSQLStore_SavePaneOrder(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	for _, sid := range []string{"a", "b", "c"} {
		if err := store.UpsertPane(workspace.Pane{SessionID: sid, SortOrder: 0}); err != nil {
			t.Fatalf("UpsertPane %s: %v", sid, err)
		}
	}

	if err := store.SavePaneOrder("c", []string{"c", "b", "a"}); err != nil {
		t.Fatalf("SavePaneOrder: %v", err)
	}

	layout, _ := store.GetLayout()
	if layout.ActivePane != "c" {
		t.Errorf("active pane: got %q, want %q", layout.ActivePane, "c")
	}
	if len(layout.Panes) != 3 {
		t.Fatalf("expected 3 panes, got %d", len(layout.Panes))
	}
	if layout.Panes[0].SessionID != "c" {
		t.Errorf("first pane: got %q, want %q", layout.Panes[0].SessionID, "c")
	}
	if layout.Panes[2].SessionID != "a" {
		t.Errorf("last pane: got %q, want %q", layout.Panes[2].SessionID, "a")
	}
}

func TestSQLStore_TabGroups(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	g, err := store.CreateGroup("Dev", "#00ff00")
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Name != "Dev" {
		t.Errorf("name: got %q", g.Name)
	}
	if g.Color != "#00ff00" {
		t.Errorf("color: got %q", g.Color)
	}
	if g.ID == "" {
		t.Error("expected non-empty ID")
	}

	g2, _ := store.CreateGroup("", "")
	if g2.Name != "Group" {
		t.Errorf("default name: got %q", g2.Name)
	}
	if g2.SortOrder != 1 {
		t.Errorf("second group sort_order: got %d, want 1", g2.SortOrder)
	}

	newName := "Production"
	collapsed := true
	updated, err := store.UpdateGroup(g.ID, &newName, nil, &collapsed)
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if updated.Name != "Production" {
		t.Errorf("updated name: got %q", updated.Name)
	}
	if !updated.IsCollapsed {
		t.Error("expected collapsed=true")
	}
	if updated.Color != "#00ff00" {
		t.Errorf("color should be unchanged: got %q", updated.Color)
	}

	if _, err := store.UpdateGroup("nonexistent", &newName, nil, nil); err == nil {
		t.Error("expected error for non-existent group")
	}

	deleted, err := store.DeleteGroup(g.ID)
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	deleted, _ = store.DeleteGroup("nonexistent")
	if deleted {
		t.Error("expected deleted=false for non-existent group")
	}
}

func TestSQLStore_GroupPaneRelation(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	g, _ := store.CreateGroup("Test", "#0000ff")
	if err := store.UpsertPane(workspace.Pane{SessionID: "s1", GroupID: g.ID}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, _ := store.GetLayout()
	if layout.Panes[0].GroupID != g.ID {
		t.Errorf("pane group_id: got %q, want %q", layout.Panes[0].GroupID, g.ID)
	}

	if _, err := store.DeleteGroup(g.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	layout, _ = store.GetLayout()
	if layout.Panes[0].GroupID != "" {
		t.Errorf("expected empty group_id after group delete, got %q", layout.Panes[0].GroupID)
	}
}

func TestSQLStore_UnicodeNames(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	if err := store.UpsertPane(workspace.Pane{SessionID: "uni-1", Name: "터미널"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}
	layout, _ := store.GetLayout()
	if layout.Panes[0].Name != "터미널" {
		t.Errorf("unicode name: got %q", layout.Panes[0].Name)
	}

	g, _ := store.CreateGroup("グループ", "#123456")
	if g.Name != "グループ" {
		t.Errorf("unicode group name: got %q", g.Name)
	}
}
