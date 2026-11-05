package routine

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestRoutineRepositoryCreatesImmutableRevision(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	q, _ := decimalx.Parse("1")
	r := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	created, err := r.Create(ctx, Template{WorkspaceID: "w", SlotName: "dinner", RecipeID: "r1", Quantity: q, Weekdays: []int{5}, StartDate: "2026-09-18", Mode: "fixed", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := r.Update(ctx, UpdateInput{WorkspaceID: "w", ID: created.ID, ExpectedRevision: 1, SlotName: "dinner", RecipeID: "r2", Quantity: q, Weekdays: []int{5}, StartDate: "2026-09-18", Mode: "fixed", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 {
		t.Fatalf("revision = %d", updated.Revision)
	}
	old, err := r.Get(ctx, created.ID, "w", 1)
	if err != nil {
		t.Fatal(err)
	}
	if old.RecipeID != "r1" {
		t.Fatal("historical routine revision was rewritten")
	}
}
