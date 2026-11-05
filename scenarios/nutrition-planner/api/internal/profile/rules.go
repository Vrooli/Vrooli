package profile

import "strings"

const CurrentPresetVersion int64 = 1

var presetRules = map[string][]string{
	"everything": {}, "vegan": {"animal_products", "milk", "eggs", "meat", "fish", "shellfish"},
	"vegetarian": {"meat", "fish", "shellfish"}, "pescatarian": {"meat"},
	"plant-forward": {"red_meat"}, "my-own-way": {},
}

func ExpandPreset(preset string) []string {
	preset = strings.ToLower(strings.TrimSpace(preset))
	base, ok := presetRules[preset]
	if !ok {
		preset = "everything"
		base = presetRules[preset]
	}
	out := append([]string(nil), base...)
	return out
}
