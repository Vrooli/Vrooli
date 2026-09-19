package planning

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestApplyRejectsStaleDraft(t *testing.T) {
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
	rev, err := r.Apply(ctx, "w1", 0, "first")
	if err != nil || rev != 1 {
		t.Fatalf("rev=%d err=%v", rev, err)
	}
	if _, err = r.Apply(ctx, "w1", 0, "stale"); err == nil {
		t.Fatal("stale draft accepted")
	}
}
