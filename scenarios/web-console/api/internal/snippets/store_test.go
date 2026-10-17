package snippets

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func newTestSQLStore(t *testing.T) *SQLStore {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "snippets.db")+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return NewSQLStore(db)
}

func eachStore(t *testing.T, run func(*testing.T, Store)) {
	t.Helper()
	t.Run("mem", func(t *testing.T) { run(t, NewMemStore()) })
	t.Run("sql", func(t *testing.T) { run(t, newTestSQLStore(t)) })
}

func TestListOrdersPinnedThenRecentThenCountThenID(t *testing.T) {
	eachStore(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		never, err := store.Upsert(ctx, UpsertRequest{ID: "never", Name: "Never used", Body: "n"})
		if err != nil {
			t.Fatal(err)
		}
		used, err := store.Upsert(ctx, UpsertRequest{ID: "used", Name: "Used", Body: "u"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Touch(ctx, used.ID, time.Date(2026, 1, 1, 0, 0, 0, 100_000_000, time.UTC)); err != nil {
			t.Fatal(err)
		}
		pinned, err := store.Upsert(ctx, UpsertRequest{ID: "pinned", Name: "Pinned", Pinned: true, HasPinned: true})
		if err != nil {
			t.Fatal(err)
		}
		listed, err := store.List(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{pinned.ID, used.ID, never.ID}
		for i, id := range want {
			if listed[i].ID != id {
				t.Fatalf("order[%d] = %q, want %q; all=%#v", i, listed[i].ID, id, listed)
			}
		}
		if listed[2].LastUsedAt != "" {
			t.Fatalf("never-used timestamp = %q, want empty", listed[2].LastUsedAt)
		}
	})
}

func TestTouchIncrementsOnceWithoutRewritingContent(t *testing.T) {
	eachStore(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		created, err := store.Upsert(ctx, UpsertRequest{Name: "Keep", Body: "body", Color: "#22d3ee"})
		if err != nil {
			t.Fatal(err)
		}
		touched, err := store.Touch(ctx, created.ID, time.Date(2026, 8, 28, 12, 0, 0, 123, time.UTC))
		if err != nil {
			t.Fatal(err)
		}
		if touched.UseCount != 1 {
			t.Fatalf("use_count = %d, want 1", touched.UseCount)
		}
		if touched.Name != created.Name || touched.Body != created.Body || touched.Color != created.Color {
			t.Fatalf("touch rewrote content: before=%#v after=%#v", created, touched)
		}
	})
}

func TestBlankNameRejectedAndBlankBodyAccepted(t *testing.T) {
	eachStore(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		if _, err := store.Upsert(ctx, UpsertRequest{Name: "   "}); err != ErrInvalidSnippet {
			t.Fatalf("blank name error = %v, want ErrInvalidSnippet", err)
		}
		if _, err := store.Upsert(ctx, UpsertRequest{Name: "Named first", Body: ""}); err != nil {
			t.Fatalf("blank body rejected: %v", err)
		}
	})
}

func TestDeleteIsIdempotentAndMissingTouchIsTyped(t *testing.T) {
	eachStore(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		created, err := store.Upsert(ctx, UpsertRequest{Name: "Disposable"})
		if err != nil {
			t.Fatal(err)
		}
		if deleted, err := store.Delete(ctx, created.ID); err != nil || !deleted {
			t.Fatalf("first delete = %v, %v", deleted, err)
		}
		if deleted, err := store.Delete(ctx, created.ID); err != nil || deleted {
			t.Fatalf("second delete = %v, %v", deleted, err)
		}
		if _, err := store.Touch(ctx, created.ID, time.Now()); err != ErrSnippetNotFound {
			t.Fatalf("missing touch = %v, want ErrSnippetNotFound", err)
		}
	})
}

func TestUpsertPreservesPinnedAndUsageWhenOmitted(t *testing.T) {
	eachStore(t, func(t *testing.T, store Store) {
		ctx := context.Background()
		created, err := store.Upsert(ctx, UpsertRequest{Name: "Pinned", Body: "old", Pinned: true, HasPinned: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Touch(ctx, created.ID, time.Now()); err != nil {
			t.Fatal(err)
		}
		edited, err := store.Upsert(ctx, UpsertRequest{ID: created.ID, Name: "Pinned", Body: "new"})
		if err != nil {
			t.Fatal(err)
		}
		if !edited.Pinned || edited.UseCount != 1 || edited.LastUsedAt == "" {
			t.Fatalf("edit lost metadata: %#v", edited)
		}
	})
}
