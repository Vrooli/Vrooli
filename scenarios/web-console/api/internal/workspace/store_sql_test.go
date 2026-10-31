package workspace

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestSQLStore opens a store over the REAL schema.sql with foreign keys
// on, not a hand-written subset. A subset cannot exercise ON DELETE CASCADE
// or the partial unique index, which are exactly the two behaviours roles
// depend on — and it silently drifts from production the moment the schema
// changes.
func newTestSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	dsn := "file:" + filepath.Join(t.TempDir(), "workspace.db") +
		"?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	schema, err := os.ReadFile(filepath.Join("..", "sessions", "schema.sql"))
	if err != nil {
		t.Fatalf("read schema.sql: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		t.Fatalf("exec schema.sql: %v", err)
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
