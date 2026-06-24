package autosteer

import (
	"reflect"
	"testing"

	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
)

func eligibilityResolver() *skillmap.Resolver {
	return skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "docs", Dimensions: []string{"docs"}},
		{ID: "refactor", Dimensions: []string{"standards", "structure", "cycles"}},
		{ID: "security", Dimensions: []string{"security"}},
		{ID: "test", Dimensions: []string{"tests", "coverage"}},
		{ID: "ux", Dimensions: []string{"ui", "visual"}},
	}}, func(string, ...any) {})
}

func TestRelevantDimensionsFromWeights(t *testing.T) {
	p := &AutoSteerProfile{
		Objective: Objective{DimensionWeights: map[string]float64{"ui": 1, "tests": 1}},
	}
	got := relevantDimensions(p)
	for _, want := range []dimensions.Dimension{"ui", "tests"} {
		if !containsDimension(got, want) {
			t.Fatalf("relevantDimensions missing %q from %v", want, got)
		}
	}
	if len(got) != 2 {
		t.Fatalf("relevantDimensions = %v, want exactly [tests ui]", got)
	}
}

func TestEffectiveAllowDerivesFromWeightedDimensions(t *testing.T) {
	p := &AutoSteerProfile{Objective: Objective{DimensionWeights: map[string]float64{"ui": 1, "standards": 1}}}
	got := effectiveAllow(p, eligibilityResolver())
	want := []string{"refactor", "ux"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effectiveAllow = %v, want %v", got, want)
	}
}

func TestEffectiveAllowAppliesMaskAndDeny(t *testing.T) {
	p := &AutoSteerProfile{
		Objective:     Objective{DimensionWeights: map[string]float64{"standards": 1, "tests": 1}},
		AllowedSkills: []string{"test", "refactor"},
		DeniedSkills:  []string{"refactor"},
	}
	got := effectiveAllow(p, eligibilityResolver())
	if !reflect.DeepEqual(got, []string{"test"}) {
		t.Fatalf("effectiveAllow = %v, want [test]", got)
	}
}

func TestReconcileProfileRejectsDeadMaskEntry(t *testing.T) {
	p := &AutoSteerProfile{
		Name:          "Docs Only",
		Objective:     Objective{DimensionWeights: map[string]float64{"docs": 1}},
		AllowedSkills: []string{"refactor"},
	}
	if err := ReconcileProfile(p, eligibilityResolver()); err == nil {
		t.Fatal("expected a dead allowed_skills entry to fail reconciliation")
	}
}

func TestReconcileProfileAcceptsDerivedMaskEntry(t *testing.T) {
	p := &AutoSteerProfile{
		Name:          "Docs Only",
		Objective:     Objective{DimensionWeights: map[string]float64{"docs": 1}},
		AllowedSkills: []string{" docs ", "docs"},
	}
	if err := ReconcileProfile(p, eligibilityResolver()); err != nil {
		t.Fatalf("unexpected reconciliation error: %v", err)
	}
	if !reflect.DeepEqual(p.AllowedSkills, []string{"docs"}) {
		t.Fatalf("allowed skills normalized to %v, want [docs]", p.AllowedSkills)
	}
}

func containsDimension(values []dimensions.Dimension, want dimensions.Dimension) bool {
	for _, got := range values {
		if got == want {
			return true
		}
	}
	return false
}
