package supplement

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestSchedulePersistenceAndRevisionUpdate(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	dose, _ := decimalx.Parse("2")
	r := NewSQLiteRepository(db, schedule.System())
	ctx := context.Background()
	s, err := r.Create(ctx, Schedule{WorkspaceID: "w", ProductRevisionID: "product:p:1", Dose: dose, DoseUnit: "capsule", Weekdays: []int{1, 3, 5}, StartDate: "2026-09-18", Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !AppliesOn(s, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)) || AppliesOn(s, time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("weekday applicability incorrect: %+v", s)
	}
	nextDose, _ := decimalx.Parse("1")
	updated, err := r.Update(ctx, UpdateInput{WorkspaceID: "w", ID: s.ID, ExpectedRevision: 1, Dose: nextDose, DoseUnit: "capsule", Weekdays: []int{1, 3, 5}, StartDate: "2026-09-18", Confirmed: true})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.Dose.String() != "1" {
		t.Fatalf("update not revisioned: %+v", updated)
	}
	if _, err = r.Update(ctx, UpdateInput{WorkspaceID: "w", ID: s.ID, ExpectedRevision: 1, Dose: dose, DoseUnit: "capsule", Weekdays: []int{1}, StartDate: "2026-09-18", Confirmed: true}); err == nil {
		t.Fatal("stale schedule update accepted")
	}
}

func TestScheduleRequiresConfirmationAndValidWeekday(t *testing.T) {
	dose, _ := decimalx.Parse("1")
	base := Schedule{WorkspaceID: "w", ProductRevisionID: "p", Dose: dose, DoseUnit: "g", Weekdays: []int{7}, StartDate: "2026-09-18"}
	if err := Validate(base); err == nil {
		t.Fatal("invalid weekday accepted")
	}
	base.Weekdays = []int{1}
	if AppliesOn(base, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("unconfirmed schedule applied")
	}
}
