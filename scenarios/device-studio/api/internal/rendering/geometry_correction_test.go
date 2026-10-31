package rendering

import (
	"encoding/json"
	"strings"
	"testing"

	"device-studio/internal/geometry"
)

func TestGeometryCorrectionPromptSeparatesAcceptedVisualRefinementFromSemanticRepair(t *testing.T) {
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{{ID: "housing", Kind: "rounded_box", Role: "housing", Size: [3]float64{78, 148, 10}, Bevel: 1, Material: "matte_polymer"}, {ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"}}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	recipeJSON, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	base := GeometryCorrectionPrompt{Mode: "visual_refinement", SourceJobID: "12345678-1234-1234-1234-123456789abc", SourceInputFingerprint: workerDigest([]byte("input")), SourceBriefSHA256: workerDigest([]byte("brief")), RecipeSHA256: workerDigest(recipeJSON), RecipeJSON: string(recipeJSON), Feedback: "Make the bezel thinner while preserving the locked dimensions."}
	if !ValidateGeometryCorrectionPrompt(&base) {
		t.Fatal("valid accepted recipe with bounded visual feedback should be refinable without a semantic issue")
	}
	withIssue := base
	withIssue.Issue = &geometry.SemanticIssue{Code: "part_outside_envelope", PartID: "housing", Axis: "y", Message: "failed semantic recipe"}
	if ValidateGeometryCorrectionPrompt(&withIssue) {
		t.Fatal("visual refinement must not disguise a semantic failure as an accepted recipe")
	}
	badRecipe := base
	badRecipe.RecipeJSON = `{"version":"device_geometry_recipe.v1","operation":"invalid"}`
	badRecipe.RecipeSHA256 = workerDigest([]byte(badRecipe.RecipeJSON))
	if ValidateGeometryCorrectionPrompt(&badRecipe) {
		t.Fatal("visual refinement must remain bound to a canonical construct recipe")
	}
}

func TestThinRoundedBoxBevelFailureIsSourceBoundAndCorrectableWithStrictNumericRule(t *testing.T) {
	recipe := geometry.Recipe{Version: "device_geometry_recipe.v1", Profile: "detailed_device_v1", Operation: "construct", Envelope: geometry.Dimensions{80, 150, 12}, Parts: []geometry.Part{
		{ID: "thin_bezel", Kind: "rounded_box", Role: "bezel", Size: [3]float64{64, 124, 2}, Bevel: 1, Material: "gloss_polymer"},
		{ID: "display", Kind: "box", Role: "display", Center: [3]float64{0, 0, 5.2}, Size: [3]float64{64, 124, 1}, Material: "glass"},
	}, Screen: geometry.Screen{PartID: "display", Face: "front:+z", Center: [2]float64{.5, .5}, Size: [2]float64{.9, .9}}}
	encoded, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = geometry.Decode(encoded)
	issue, ok := geometry.IssueFromError(err)
	if !ok || issue.Code != "bevel_exceeds_part_dimensions" {
		t.Fatalf("thin-panel bevel must remain rejected as typed correction issue, got issue=%+v err=%v", issue, err)
	}
	correction := GeometryCorrectionPrompt{Mode: "semantic_repair", SourceJobID: "12345678-1234-1234-1234-123456789abc", SourceInputFingerprint: workerDigest([]byte("input")), SourceBriefSHA256: workerDigest([]byte("brief")), RecipeSHA256: workerDigest(encoded), RecipeJSON: string(encoded), Issue: &issue, Feedback: "Reduce only the bezel bevel."}
	if !ValidateGeometryCorrectionPrompt(&correction) {
		t.Fatal("exact source-bound thin-panel issue should be eligible for semantic repair")
	}
	context := geometryCorrectionContext(&correction)
	for _, want := range []string{"part thin_bezel", "bevel_mm 1 mm", "smallest size_mm dimension is 2 mm", "0 < bevel_mm < 1 mm", "equality is invalid"} {
		if !strings.Contains(context, want) {
			t.Fatalf("correction context must include %q, got %s", want, context)
		}
	}
	correction.Issue.MaxBevelExclusiveMM = 1.01
	if ValidateGeometryCorrectionPrompt(&correction) {
		t.Fatal("a correction issue with an altered exclusive limit must not retain source binding")
	}
}
