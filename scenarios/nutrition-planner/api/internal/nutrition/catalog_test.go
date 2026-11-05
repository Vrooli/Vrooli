package nutrition

import (
	"testing"

	"nutrition-planner/internal/catalog"
	"nutrition-planner/internal/decimalx"
)

func TestCatalogContributionsPreservePinnedEvidenceAndScale(t *testing.T) {
	serving, _ := decimalx.Parse("1")
	protein, _ := decimalx.Parse("41.5")
	quantity, _ := decimalx.Parse("2")
	revision := catalog.Revision{ID: "food-1", Revision: 3, WorkspaceID: "w1", ConceptID: "beans", Name: "Beans", ServingQuantity: serving, ServingUnit: "serving", Nutrients: []catalog.NutrientValue{{NutrientID: "protein", Amount: protein, Unit: "g", Basis: serving, BasisUnit: "serving", Evidence: catalog.EvidenceSource, SourceRef: "fdc:123"}}}
	contributions, err := ContributionsFromCatalog(revision, quantity, "serving")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Aggregate("protein", contributions)
	if err != nil || result.Known.String() != "83" || !result.Complete || len(result.EvidenceSources) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}
