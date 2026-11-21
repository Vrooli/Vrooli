package feedback

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestFeedbackRecordAndUndoAreExplicit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	r := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	if _, err = r.Record(ctx, Feedback{WorkspaceID: "w", Date: "2026-09-20", RecipeID: "r"}); err != nil {
		t.Fatal(err)
	}
	if _, err = r.Get(ctx, "w", "2026-09-20"); err != nil {
		t.Fatal(err)
	}
	if err = r.Undo(ctx, "w", "2026-09-20"); err != nil {
		t.Fatal(err)
	}
	if err = r.Undo(ctx, "w", "2026-09-20"); err == nil {
		t.Fatal("missing feedback undo accepted")
	}
}
