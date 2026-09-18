package nutrition

import (
	"fmt"

	"nutrition-planner/internal/catalog"
	"nutrition-planner/internal/decimalx"
)

// ContributionsFromCatalog turns one pinned catalog revision into explicit
// nutrient contributions for a requested quantity. It carries the source and
// basis through unchanged; it does not infer preparation conversions or mix
// generic and branded evidence.
func ContributionsFromCatalog(revision catalog.Revision, quantity decimalx.Decimal, quantityUnit string) ([]Contribution, error) {
	if revision.ID == "" || quantity.IsUnknown() || quantity.IsZero() || quantityUnit == "" {
		return nil, fmt.Errorf("catalog contribution requires a pinned revision and known quantity")
	}
	if err := catalog.Validate(revision); err != nil {
		return nil, err
	}
	if quantityUnit != revision.ServingUnit {
		return nil, fmt.Errorf("catalog quantity unit %q does not match serving unit %q", quantityUnit, revision.ServingUnit)
	}
	out := make([]Contribution, 0, len(revision.Nutrients))
	for _, nutrient := range revision.Nutrients {
		out = append(out, Contribution{NutrientID: nutrient.NutrientID, Amount: nutrient.Amount, Basis: nutrient.Basis, Quantity: quantity, Unit: quantityUnit, BasisUnit: nutrient.BasisUnit, Source: fmt.Sprintf("catalog:%s:%d", revision.ID, revision.Revision)})
	}
	return out, nil
}
