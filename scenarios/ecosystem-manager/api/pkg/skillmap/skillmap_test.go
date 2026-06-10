package skillmap

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vrooli/maturity-go/dimensions"
)

func sampleCatalog() *FakeCatalog {
	return &FakeCatalog{Declarations: []SkillDeclaration{
		{ID: "ux", Dimensions: []string{"accessibility", "visual", "ui"}},
		{ID: "test", Dimensions: []string{"tests", "coverage"}},
		{ID: "refactor", Dimensions: []string{"structure", "standards", "cycles", "tidiness"}},
		{ID: "polish", Dimensions: []string{"visual", "tidiness", "standards"}},
	}}
}

func newSilentResolver(src CatalogSource) *Resolver {
	return NewResolverWithWarner(src, func(string, ...any) {})
}

func TestSkillsForDimensionSorted(t *testing.T) {
	r := newSilentResolver(sampleCatalog())
	got := r.SkillsForDimension(dimensions.Dimension("standards"))
	want := []string{"polish", "refactor"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("standards skills = %v, want %v", got, want)
	}
	if got := r.SkillsForDimension(dimensions.Dimension("security")); len(got) != 0 {
		t.Errorf("security has no skills, got %v", got)
	}
}

func TestEligibleSkillsIntersectsAllowSetInOrder(t *testing.T) {
	r := newSilentResolver(sampleCatalog())
	// visual is targeted by polish and ux; allow-set order is honored.
	got := r.EligibleSkills(dimensions.Dimension("visual"), []string{"ux", "polish"})
	if !reflect.DeepEqual(got, []string{"ux", "polish"}) {
		t.Errorf("got %v, want [ux polish]", got)
	}
	got = r.EligibleSkills(dimensions.Dimension("visual"), []string{"polish"})
	if !reflect.DeepEqual(got, []string{"polish"}) {
		t.Errorf("got %v, want [polish]", got)
	}
	// A skill not targeting the dimension is filtered out.
	got = r.EligibleSkills(dimensions.Dimension("tests"), []string{"ux", "test"})
	if !reflect.DeepEqual(got, []string{"test"}) {
		t.Errorf("got %v, want [test]", got)
	}
}

func TestEmptyAllowSetMeansNoRestriction(t *testing.T) {
	r := newSilentResolver(sampleCatalog())
	got := r.EligibleSkills(dimensions.Dimension("visual"), nil)
	if !reflect.DeepEqual(got, []string{"polish", "ux"}) {
		t.Errorf("got %v, want [polish ux]", got)
	}
}

func TestDropsOutOfVocabularyDimension(t *testing.T) {
	var warnings []string
	src := &FakeCatalog{Declarations: []SkillDeclaration{
		{ID: "test", Dimensions: []string{"tests", "bogus-dimension"}},
	}}
	r := NewResolverWithWarner(src, func(f string, a ...any) {
		warnings = append(warnings, strings.ToLower(f))
	})
	if dims := r.DimensionsForSkill("test"); !reflect.DeepEqual(dims, []dimensions.Dimension{"tests"}) {
		t.Errorf("dims = %v, want [tests]", dims)
	}
	if len(warnings) == 0 {
		t.Error("expected a warning about the dropped dimension")
	}
}

func TestExcludesSkillWithNoValidDimensions(t *testing.T) {
	var warnings []string
	src := &FakeCatalog{Declarations: []SkillDeclaration{
		{ID: "empty", Dimensions: nil},
		{ID: "all-bogus", Dimensions: []string{"nope", "also-nope"}},
		{ID: "good", Dimensions: []string{"docs"}},
	}}
	r := NewResolverWithWarner(src, func(f string, a ...any) { warnings = append(warnings, f) })

	excluded := r.Excluded()
	if !reflect.DeepEqual(excluded, []string{"all-bogus", "empty"}) {
		t.Errorf("excluded = %v, want [all-bogus empty]", excluded)
	}
	if got := r.SkillsForDimension(dimensions.Dimension("docs")); !reflect.DeepEqual(got, []string{"good"}) {
		t.Errorf("docs skills = %v, want [good]", got)
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for excluded skills")
	}
}

func TestDeduplicatesDeclaredDimensions(t *testing.T) {
	src := &FakeCatalog{Declarations: []SkillDeclaration{
		{ID: "dupe", Dimensions: []string{"tests", "tests", "coverage"}},
	}}
	r := newSilentResolver(src)
	if dims := r.DimensionsForSkill("dupe"); !reflect.DeepEqual(dims, []dimensions.Dimension{"tests", "coverage"}) {
		t.Errorf("dims = %v, want [tests coverage]", dims)
	}
}
