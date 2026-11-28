package nutrition

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestTargetRepositoryPersistsBoundsAndProvenance(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	lower, _ := decimalx.Parse("100")
	upper, _ := decimalx.Parse("200")
	r := NewSQLiteTargetRepository(db, schedule.System())
	want := Target{WorkspaceID: "w", NutrientID: "protein", Lower: lower, Upper: upper, Period: "local_day", Scope: "planned_day", Enforcement: "required", Provenance: "user_assertion", EffectiveFrom: time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC)}
	created, err := r.Create(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := r.Get(context.Background(), created.ID, "w", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Lower.String() != "100" || got.Upper.String() != "200" || got.Provenance != "user_assertion" {
		t.Fatalf("target not preserved: %+v", got)
	}
}

func TestValidateTargetRejectsInvalidBoundsAndMissingEvidence(t *testing.T) {
	lower, _ := decimalx.Parse("3")
	upper, _ := decimalx.Parse("2")
	base := Target{WorkspaceID: "w", NutrientID: "energy", Lower: lower, Upper: upper, Period: "local_day", Scope: "planned_day", Enforcement: "required", EffectiveFrom: time.Now()}
	if err := ValidateTarget(base); err == nil {
		t.Fatal("accepted inverted bounds")
	}
	base.Lower, base.Upper = decimalx.Unknown, decimalx.Unknown
	if err := ValidateTarget(base); err == nil {
		t.Fatal("accepted target without a bound")
	}
}
