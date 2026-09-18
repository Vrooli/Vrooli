package recipe

import (
	"fmt"
	"strings"

	"nutrition-planner/internal/decimalx"
)

type ScalableQuantity struct {
	ComponentID string
	Amount      decimalx.Decimal
	Unit        string
	Discrete    bool
}

// ScaleQuantities derives display-only quantities from a canonical yield. It
// never mutates the stored recipe and deliberately leaves time, temperature,
// and appliance-capacity values outside this operation.
func ScaleQuantities(quantities []ScalableQuantity, canonicalYield, targetYield decimalx.Decimal) ([]ScalableQuantity, error) {
	ratio, err := decimalx.Div(targetYield, canonicalYield)
	if err != nil {
		return nil, fmt.Errorf("yield scale: %w", err)
	}
	if ratio.IsUnknown() {
		return nil, fmt.Errorf("yield scale: unknown yield")
	}
	out := make([]ScalableQuantity, len(quantities))
	for i, quantity := range quantities {
		amount, err := decimalx.Mul(quantity.Amount, ratio)
		if err != nil {
			return nil, err
		}
		if quantity.Discrete && strings.Contains(amount.String(), ".") {
			return nil, fmt.Errorf("quantity %q would become fractional", quantity.ComponentID)
		}
		out[i] = quantity
		out[i].Amount = amount
	}
	return out, nil
}
