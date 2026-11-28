// Package nutrients defines stable nutrient identities and provider mappings.
package nutrients

type Nutrient struct {
	ID, Name, CanonicalUnit string
	Forms                   []string
	ProviderCodes           map[string]string
}

var all = map[string]Nutrient{
	"energy_kcal":        {"energy_kcal", "Energy", "kcal", nil, map[string]string{"fdc": "1008"}},
	"protein":            {"protein", "Protein", "g", nil, map[string]string{"fdc": "1003"}},
	"carbohydrate":       {"carbohydrate", "Carbohydrate", "g", nil, map[string]string{"fdc": "1005"}},
	"fat":                {"fat", "Total fat", "g", nil, map[string]string{"fdc": "1004"}},
	"fiber":              {"fiber", "Fiber", "g", nil, map[string]string{"fdc": "1079"}},
	"folate":             {"folate", "Folate", "ug", []string{"folate"}, map[string]string{"fdc": "1176"}},
	"folate_equivalents": {"folate_equivalents", "Folate equivalents", "ug", []string{"dfe"}, map[string]string{"fdc": "1190"}},
	"vitamin_a":          {"vitamin_a", "Vitamin A", "ug", []string{"retinol", "rae"}, map[string]string{"fdc": "1106"}},
	"niacin":             {"niacin", "Niacin", "mg", []string{"niacin", "ne"}, map[string]string{"fdc": "1167"}},
	"omega3_ala":         {"omega3_ala", "Omega-3 ALA", "g", nil, map[string]string{"fdc": "1278"}},
	"omega3_epa":         {"omega3_epa", "Omega-3 EPA", "g", nil, map[string]string{"fdc": "1278"}},
	"omega3_dha":         {"omega3_dha", "Omega-3 DHA", "g", nil, map[string]string{"fdc": "1272"}},
}

func Lookup(id string) (Nutrient, bool) { n, ok := all[id]; return n, ok }
func All() []Nutrient {
	out := make([]Nutrient, 0, len(all))
	for _, n := range all {
		out = append(out, n)
	}
	return out
}
