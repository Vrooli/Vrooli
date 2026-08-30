package gates

import (
	"path/filepath"
	"testing"
)

func TestClassifySurfaceReportsAllOracleCombinations(t *testing.T) {
	tests := []struct {
		name                                     string
		staticApplied, captured, renderedMatches bool
		want                                     SurfaceVerdict
	}{
		{name: "both agree", staticApplied: true, captured: true, renderedMatches: true, want: SurfacePass},
		{name: "rendered wrong", staticApplied: true, captured: true, renderedMatches: false, want: SurfaceRenderedWrong},
		{name: "hard coded", staticApplied: false, captured: true, renderedMatches: true, want: SurfaceHardCoded},
		{name: "both mismatch", staticApplied: false, captured: true, renderedMatches: false, want: SurfaceBothMismatch},
		{name: "no capture", staticApplied: true, captured: false, renderedMatches: false, want: SurfaceUnmeasured},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := classifySurface(test.staticApplied, test.captured, test.renderedMatches); got != test.want {
				t.Fatalf("classifySurface() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSourceUsesSurfaceRampRequiresAnAppliedReference(t *testing.T) {
	if sourceUsesSurfaceRamp(`import { SURFACE_ELEVATIONS } from "./VisualRecipes"; const shadow = "shadow-[var(--elev-raised)]";`, "raised") {
		t.Fatal("unused ramp import was treated as applied")
	}
	if !sourceUsesSurfaceRamp(`import { SURFACE_ELEVATIONS } from "./VisualRecipes"; const className = SURFACE_ELEVATIONS.raised;`, "raised") {
		t.Fatal("applied ramp reference was not detected")
	}
}

func TestEquivalentBoxShadowsAcceptsComputedBrowserSerialization(t *testing.T) {
	if !equivalentBoxShadows(
		"rgba(9, 18, 22, 0.06) 0px 1px 2px 0px, rgba(9, 18, 22, 0.1) 0px 1px 3px 0px",
		"0 1px 2px rgba(9, 18, 22, 0.06), 0 1px 3px rgba(9, 18, 22, 0.1)",
	) {
		t.Fatal("computed and authored raised shadows should compare equal")
	}
	if equivalentBoxShadows(
		"rgba(9, 18, 22, 0.06) 0px 2px 2px 0px",
		"0 1px 2px rgba(9, 18, 22, 0.06)",
	) {
		t.Fatal("different elevation offsets should not compare equal")
	}
}

func TestValidateSurfaceDisciplinePublishesCorpusCounts(t *testing.T) {
	root, err := filepath.Abs("../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateSurfaceDiscipline(Scope{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspected == 0 || len(result.SurfaceCounts) == 0 {
		t.Fatalf("surface gate did not publish inspected counts: inspected=%d counts=%v", result.Inspected, result.SurfaceCounts)
	}
}
