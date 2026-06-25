package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func finding(id string, dim string, sev architecturev1.FindingSeverity) findings.Finding {
	return findings.Finding{ID: id, Dimension: dimensions.Dimension(dim), Severity: sev}
}

func testResolver() *skillmap.Resolver {
	return skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "refactor", Dimensions: []string{"standards", "structure"}},
		{ID: "test", Dimensions: []string{"tests"}},
		{ID: "security", Dimensions: []string{"security"}},
	}}, func(string, ...any) {})
}

func TestSelector_PicksHeaviestDimensionWithEligibleSkill(t *testing.T) { // [REQ:EM-CTRL-003]
	sel := NewSelector(testResolver())
	state := findings.BuildState([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING), // weight 2
		finding("b", "tests", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER),     // weight 8 (heaviest)
	})
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor", "test", "security"}}

	got := sel.SelectNextSkill(state, profile)
	if got.SkillID != "test" {
		t.Fatalf("expected 'test' for heaviest (tests) dimension, got %q (%s)", got.SkillID, got.Rationale)
	}
	if got.Dimension != dimensions.Dimension("tests") {
		t.Fatalf("expected dimension 'tests', got %q", got.Dimension)
	}
}

func TestSelector_RespectsDimensionWeights(t *testing.T) {
	sel := NewSelector(testResolver())
	state := findings.BuildState([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING), // raw 2
		finding("b", "tests", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),       // raw 4
	})
	// Weight standards 5x so its weighted score (10) beats tests (4).
	profile := &AutoSteerProfile{
		AllowedSkills: []string{"refactor", "test"},
		Objective:     Objective{DimensionWeights: map[string]float64{"standards": 5.0}},
	}

	got := sel.SelectNextSkill(state, profile)
	if got.SkillID != "refactor" {
		t.Fatalf("expected weighting to select 'refactor' for standards, got %q", got.SkillID)
	}
}

func TestSelector_FallsThroughWhenNoEligibleSkill(t *testing.T) {
	sel := NewSelector(testResolver())
	state := findings.BuildState([]findings.Finding{
		finding("a", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER), // heaviest, but...
		finding("b", "tests", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	// security skill not in allow-set → fall through to tests.
	profile := &AutoSteerProfile{AllowedSkills: []string{"test"}}

	got := sel.SelectNextSkill(state, profile)
	if got.SkillID != "test" {
		t.Fatalf("expected fall-through to 'test', got %q (%s)", got.SkillID, got.Rationale)
	}
}

func TestSelector_EmptyFindingsSelectsNothing(t *testing.T) {
	sel := NewSelector(testResolver())
	got := sel.SelectNextSkill(findings.BuildState(nil), &AutoSteerProfile{AllowedSkills: []string{"test"}})
	if got.SkillID != "" {
		t.Fatalf("expected empty selection for no findings, got %q", got.SkillID)
	}
}

func TestSelector_NoEligibleSkillForAnyDimension(t *testing.T) {
	sel := NewSelector(testResolver())
	state := findings.BuildState([]findings.Finding{
		finding("a", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER),
	})
	// Only 'test' allowed, which targets 'tests' (no open tests findings).
	got := sel.SelectNextSkill(state, &AutoSteerProfile{AllowedSkills: []string{"test"}})
	if got.SkillID != "" {
		t.Fatalf("expected no actionable skill, got %q", got.SkillID)
	}
}

func TestSelector_DeterministicTiebreak(t *testing.T) {
	sel := NewSelector(testResolver())
	// standards and structure both raw 4; refactor targets both. Equal weighted
	// scores resolve by alphabetical dimension order (standards < structure).
	state := findings.BuildState([]findings.Finding{
		finding("a", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("b", "structure", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor"}}
	first := sel.SelectNextSkill(state, profile)
	for i := 0; i < 5; i++ {
		if got := sel.SelectNextSkill(state, profile); got.Dimension != first.Dimension {
			t.Fatalf("tiebreak not stable: %q vs %q", got.Dimension, first.Dimension)
		}
	}
	if first.Dimension != dimensions.Dimension("standards") {
		t.Fatalf("expected alphabetical tiebreak to 'standards', got %q", first.Dimension)
	}
}
