package workspace

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newTestSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/workspace.db")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE tab_groups (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, color TEXT NOT NULL,
			sort_order INTEGER NOT NULL, is_collapsed INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		);
		CREATE TABLE workspace_panes (
			session_id TEXT PRIMARY KEY, name TEXT NOT NULL, header_color TEXT NOT NULL,
			theme_id TEXT NOT NULL, font_size INTEGER NOT NULL, sort_order INTEGER NOT NULL,
			group_id TEXT, is_active INTEGER NOT NULL DEFAULT 0,
			supports_messages_view INTEGER NOT NULL DEFAULT 0,
			manually_unread INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		);`)
	if err != nil {
		t.Fatal(err)
	}
	return NewSQLStore(db)
}

func TestSQLStoreLifecycleAndReassignment(t *testing.T) {
	ctx := context.Background()
	store := newTestSQLStore(t)

	group, err := store.CreateGroup(ctx, "", "")
	if err != nil || group.Name != "Group" || group.Color != "#3b82f6" {
		t.Fatalf("CreateGroup defaults = %+v, err=%v", group, err)
	}
	if _, err := store.CreateGroup(ctx, "Second", "#abc"); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertPane(ctx, Pane{SessionID: "old", GroupID: group.ID, IsActive: true, SupportsMessagesView: true, ManuallyUnread: true, SortOrder: 2}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertPane(ctx, Pane{SessionID: "new", SortOrder: 3}); err != nil {
		t.Fatal(err)
	}
	if err := store.UpsertPane(ctx, Pane{SessionID: "old", Name: "renamed", HeaderColor: "red", ThemeID: "dark", FontSize: 18, GroupID: group.ID, SortOrder: 1}); err != nil {
		t.Fatal(err)
	}

	if err := store.SavePaneOrder(ctx, "old", []string{"old", "new"}); err != nil {
		t.Fatal(err)
	}
	layout, err := store.GetLayout(ctx)
	if err != nil || layout.ActivePane != "old" || len(layout.Panes) != 2 || len(layout.Groups) != 2 {
		t.Fatalf("layout = %+v, err=%v", layout, err)
	}

	if err := store.ReassignPane(ctx, "missing", "replacement"); err != nil {
		t.Fatal(err)
	}
	if err := store.ReassignPane(ctx, "old", "new"); err != nil {
		t.Fatal(err)
	}
	layout, err = store.GetLayout(ctx)
	if err != nil || len(layout.Panes) != 1 || layout.Panes[0].SessionID != "new" || layout.Panes[0].Name != "renamed" {
		t.Fatalf("reassigned layout = %+v, err=%v", layout, err)
	}

	name, color := "Production", "#00ff00"
	collapsed := true
	updated, err := store.UpdateGroup(ctx, group.ID, &name, &color, &collapsed)
	if err != nil || updated.Name != name || updated.Color != color || !updated.IsCollapsed {
		t.Fatalf("updated group = %+v, err=%v", updated, err)
	}
	if _, err := store.UpdateGroup(ctx, group.ID, nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateGroup(ctx, "missing", &name, nil, nil); err != ErrGroupNotFound {
		t.Fatalf("missing group error = %v", err)
	}
	if ok, err := store.DeleteGroup(ctx, group.ID); err != nil || !ok {
		t.Fatalf("DeleteGroup = %v, err=%v", ok, err)
	}
	if ok, err := store.DeleteGroup(ctx, "missing"); err != nil || ok {
		t.Fatalf("DeleteGroup missing = %v, err=%v", ok, err)
	}
	if err := store.DeletePane(ctx, "new"); err != nil {
		t.Fatal(err)
	}
}
