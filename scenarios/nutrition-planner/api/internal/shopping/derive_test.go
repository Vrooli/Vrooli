package shopping

import (
	"testing"

	planning "nutrition-planner/internal/planning"
	recipe "nutrition-planner/internal/recipe"
)

func TestDerivePreservesUnknownsAndChecklistOnlyState(t *testing.T) {
	lines := Derive(planning.Draft{Occurrences: []planning.Occurrence{{RecipeID: "r1", RecipeName: "Bowl"}}}, []recipe.Recipe{{ID: "r1", Methods: []recipe.Method{{Steps: []recipe.MethodStep{{Inputs: []string{"rice"}}}}}}}, map[string]bool{"ingredient:rice": true})
	if len(lines) != 1 || !lines[0].Checked || lines[0].Need != "unknown" || lines[0].Price != "unknown" {
		t.Fatalf("lines=%+v", lines)
	}
}
