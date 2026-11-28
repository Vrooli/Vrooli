// Package units is the validated unit and dimension registry.
package units

import "fmt"

type Dimension string

const (
	Mass    Dimension = "mass"
	Volume  Dimension = "volume"
	Count   Dimension = "count"
	Serving Dimension = "serving"
	Package Dimension = "package"
)

type Unit struct {
	ID, Display string
	Dimension   Dimension
	ToBase      int64
	BaseScale   int64
}

var registry = map[string]Unit{
	"mg": {"mg", "mg", Mass, 1, 1}, "g": {"g", "g", Mass, 1_000, 1}, "kg": {"kg", "kg", Mass, 1_000_000, 1}, "ug": {"ug", "µg", Mass, 1, 1_000},
	"ml": {"ml", "mL", Volume, 1, 1}, "l": {"l", "L", Volume, 1_000, 1}, "count": {"count", "count", Count, 1, 1}, "serving": {"serving", "serving", Serving, 1, 1}, "package": {"package", "package", Package, 1, 1},
}
var aliases = map[string]string{"µg": "ug", "mcg": "ug", "milligram": "mg", "milligrams": "mg", "gram": "g", "grams": "g", "kilogram": "kg", "kilograms": "kg", "milliliter": "ml", "milliliters": "ml", "liter": "l", "liters": "l", "each": "count"}

func Lookup(id string) (Unit, bool) {
	if canonical, ok := aliases[id]; ok {
		id = canonical
	}
	u, ok := registry[id]
	return u, ok
}

func Convert(value int64, from, to string) (int64, error) {
	a, ok := Lookup(from)
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q", from)
	}
	b, ok := Lookup(to)
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q", to)
	}
	if a.Dimension != b.Dimension {
		return 0, fmt.Errorf("unit dimensions differ")
	}
	numerator := value * a.ToBase * b.BaseScale
	denominator := a.BaseScale * b.ToBase
	if numerator%denominator != 0 {
		return 0, fmt.Errorf("conversion requires fractional precision")
	}
	return numerator / denominator, nil
}
