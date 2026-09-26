package shopping

import (
	"context"

	planning "nutrition-planner/internal/planning"
	recipe "nutrition-planner/internal/recipe"
)

type Line struct {
	Key, Label, Need, Stock, Missing, PackageCount, Price string
	SourceRecipes                                         []string
	Checked                                               bool
}

type Repository interface {
	Checked(context.Context, string) (map[string]bool, error)
	SetChecked(context.Context, string, string, bool) error
}

func Derive(draft planning.Draft, recipes []recipe.Recipe, checked map[string]bool) []Line {
	byID := make(map[string]recipe.Recipe, len(recipes))
	for _, item := range recipes {
		byID[item.ID] = item
	}
	lines := make(map[string]*Line)
	for _, occurrence := range draft.Occurrences {
		item, ok := byID[occurrence.RecipeID]
		if !ok {
			key := "recipe:" + occurrence.RecipeID
			if lines[key] == nil {
				lines[key] = &Line{Key: key, Label: "Ingredients for " + occurrence.RecipeName, Need: "unknown", Stock: "unknown", Missing: "unknown", PackageCount: "unknown", Price: "unknown"}
			}
			lines[key].SourceRecipes = appendUnique(lines[key].SourceRecipes, occurrence.RecipeID)
			continue
		}
		inputs := make(map[string]bool)
		for _, method := range item.Methods {
			for _, step := range method.Steps {
				for _, input := range step.Inputs {
					if input != "" {
						inputs[input] = true
					}
				}
			}
		}
		if len(inputs) == 0 {
			inputs["recipe:"+item.ID] = true
		}
		for input := range inputs {
			key := "ingredient:" + input
			if lines[key] == nil {
				lines[key] = &Line{Key: key, Label: input, Need: "unknown", Stock: "unknown", Missing: "unknown", PackageCount: "unknown", Price: "unknown"}
			}
			lines[key].SourceRecipes = appendUnique(lines[key].SourceRecipes, item.ID)
		}
	}
	out := make([]Line, 0, len(lines))
	for _, line := range lines {
		line.Checked = checked[line.Key]
		out = append(out, *line)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Key < out[i].Key {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func appendUnique(values []string, value string) []string {
	for _, item := range values {
		if item == value {
			return values
		}
	}
	return append(values, value)
}
