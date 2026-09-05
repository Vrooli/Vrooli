package main

import (
	"context"
	"testing"

	"web-console/internal/workspace"
)

func TestSQLStore_GetLayout_Empty(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	layout, err := store.GetLayout(context.Background())
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
	if err := store.UpsertPane(context.Background(), pane); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, err := store.GetLayout(context.Background())
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

	if err := store.UpsertPane(context.Background(), workspace.Pane{SessionID: "sess-2"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, _ := store.GetLayout(context.Background())
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
	if err := store.UpsertPane(context.Background(), pane); err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	pane.Name = "updated"
	if err := store.UpsertPane(context.Background(), pane); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	layout, _ := store.GetLayout(context.Background())
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

	if err := store.UpsertPane(context.Background(), workspace.Pane{SessionID: "sess-4"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}
	if err := store.DeletePane(context.Background(), "sess-4"); err != nil {
		t.Fatalf("DeletePane: %v", err)
	}

	layout, _ := store.GetLayout(context.Background())
	if len(layout.Panes) != 0 {
		t.Errorf("expected 0 panes after delete, got %d", len(layout.Panes))
	}

	if err := store.DeletePane(context.Background(), "nonexistent"); err != nil {
		t.Errorf("delete nonexistent should not error: %v", err)
	}
}

func TestSQLStore_SavePaneOrder(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	for _, sid := range []string{"a", "b", "c"} {
		if err := store.UpsertPane(context.Background(), workspace.Pane{SessionID: sid, SortOrder: 0}); err != nil {
			t.Fatalf("UpsertPane %s: %v", sid, err)
		}
	}

	if err := store.SavePaneOrder(context.Background(), "c", []string{"c", "b", "a"}); err != nil {
		t.Fatalf("SavePaneOrder: %v", err)
	}

	layout, _ := store.GetLayout(context.Background())
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

	g, err := store.CreateGroup(context.Background(), "Dev", "#00ff00")
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

	g2, _ := store.CreateGroup(context.Background(), "", "")
	if g2.Name != "Group" {
		t.Errorf("default name: got %q", g2.Name)
	}
	if g2.SortOrder != 1 {
		t.Errorf("second group sort_order: got %d, want 1", g2.SortOrder)
	}

	newName := "Production"
	collapsed := true
	updated, err := store.UpdateGroup(context.Background(), g.ID, &newName, nil, &collapsed)
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

	if _, err := store.UpdateGroup(context.Background(), "nonexistent", &newName, nil, nil); err == nil {
		t.Error("expected error for non-existent group")
	}

	deleted, err := store.DeleteGroup(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if !deleted {
		t.Error("expected deleted=true")
	}

	deleted, _ = store.DeleteGroup(context.Background(), "nonexistent")
	if deleted {
		t.Error("expected deleted=false for non-existent group")
	}
}

func TestSQLStore_GroupPaneRelation(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	g, _ := store.CreateGroup(context.Background(), "Test", "#0000ff")
	if err := store.UpsertPane(context.Background(), workspace.Pane{SessionID: "s1", GroupID: g.ID}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, _ := store.GetLayout(context.Background())
	if layout.Panes[0].GroupID != g.ID {
		t.Errorf("pane group_id: got %q, want %q", layout.Panes[0].GroupID, g.ID)
	}

	if _, err := store.DeleteGroup(context.Background(), g.ID); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	layout, _ = store.GetLayout(context.Background())
	if layout.Panes[0].GroupID != "" {
		t.Errorf("expected empty group_id after group delete, got %q", layout.Panes[0].GroupID)
	}
}

func TestSQLStore_UnicodeNames(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)

	if err := store.UpsertPane(context.Background(), workspace.Pane{SessionID: "uni-1", Name: "터미널"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}
	layout, _ := store.GetLayout(context.Background())
	if layout.Panes[0].Name != "터미널" {
		t.Errorf("unicode name: got %q", layout.Panes[0].Name)
	}

	g, _ := store.CreateGroup(context.Background(), "グループ", "#123456")
	if g.Name != "グループ" {
		t.Errorf("unicode group name: got %q", g.Name)
	}
}

// TestSQLStore_ManuallyUnread_RoundTrips covers the "mark as unread" flag,
// which is stored on the pane rather than derived from the conversation read
// cursor. The cursor cannot express it: it only moves forward, and it exists
// only for message-capable sessions, while this flag applies to any pane —
// including a plain terminal that has no conversation at all.
func TestSQLStore_ManuallyUnread_RoundTrips(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)
	ctx := context.Background()

	pane := workspace.Pane{SessionID: "sess-1", Name: "shell", ManuallyUnread: true}
	if err := store.UpsertPane(ctx, pane); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}

	layout, err := store.GetLayout(ctx)
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(layout.Panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(layout.Panes))
	}
	if !layout.Panes[0].ManuallyUnread {
		t.Error("manually_unread did not survive the round trip")
	}
	if layout.Panes[0].SupportsMessagesView {
		t.Error("supports_messages_view leaked true; the flag must be independent of it")
	}

	pane.ManuallyUnread = false
	if err := store.UpsertPane(ctx, pane); err != nil {
		t.Fatalf("UpsertPane (clear): %v", err)
	}
	layout, err = store.GetLayout(ctx)
	if err != nil {
		t.Fatalf("GetLayout after clear: %v", err)
	}
	if layout.Panes[0].ManuallyUnread {
		t.Error("manually_unread stayed set after being cleared")
	}
}

// TestSQLStore_DefaultsUnreadToFalse pins the migration's backfill: every pane
// that existed before the column did must read as read, not as a wall of
// unexplained badges on first launch after the upgrade.
func TestSQLStore_DefaultsUnreadToFalse(t *testing.T) {
	db := setupTestDB(t)
	store := workspace.NewSQLStore(db)
	ctx := context.Background()

	if err := store.UpsertPane(ctx, workspace.Pane{SessionID: "sess-1", Name: "shell"}); err != nil {
		t.Fatalf("UpsertPane: %v", err)
	}
	layout, err := store.GetLayout(ctx)
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if layout.Panes[0].ManuallyUnread {
		t.Error("a pane created without the flag reads as manually unread")
	}
}
