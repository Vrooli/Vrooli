package catalog

import (
	"context"
	"database/sql"
	"testing"

	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
	"nutrition-planner/internal/decimalx"
)

func TestSQLiteRevisionPreservesEvidenceAndWorkspaceIsolation(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	clock := schedule.System()
	repo := NewSQLiteRepository(db, clock)
	quantity, _ := decimalx.Parse("100")
	amount, _ := decimalx.Parse("20")
	v, err := repo.Create(context.Background(), Revision{WorkspaceID: "w1", ConceptID: "food:tofu", Name: "Tofu", ProductName: "Plain tofu", ServingQuantity: quantity, ServingUnit: "g", SourceType: "label", SourceRef: "label-1", AllergenEvidence: map[string]string{"soy": "declared-present"}, Nutrients: []NutrientValue{{NutrientID: "protein", Amount: amount, Unit: "g", Basis: quantity, BasisUnit: "g", Evidence: EvidenceLabel, SourceRef: "label-1"}}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), v.ID, "w1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nutrients[0].Amount.String() != "20" || got.AllergenEvidence["soy"] != "declared-present" {
		t.Fatalf("evidence lost: %+v", got)
	}
	if _, err = repo.Get(context.Background(), v.ID, "w2", 1); err == nil {
		t.Fatal("foreign workspace was able to read catalog revision")
	}
}

func TestValidateRejectsIncompleteKnownNutrient(t *testing.T) {
	q, _ := decimalx.Parse("1")
	if err := Validate(Revision{WorkspaceID: "w", ConceptID: "c", Name: "x", ServingQuantity: q, ServingUnit: "serving", Nutrients: []NutrientValue{{NutrientID: "protein", Amount: decimalx.KnownInt(1), Basis: decimalx.Unknown}}}); err == nil {
		t.Fatal("expected amount/basis validation error")
	}
}
