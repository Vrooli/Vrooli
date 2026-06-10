package autosteer

import (
	"strings"
	"testing"

	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// coverageResolver maps the only steer skill to "standards" — leaving the
// known-unactionable dimensions (business/contracts/dependencies) with no
// eligible skill, exactly as the live catalog does (EM-P7).
func coverageResolver() *skillmap.Resolver {
	return skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "refactor", Dimensions: []string{"standards"}},
	}}, func(string, ...any) {})
}

// TestSelector_UnactionableHeaviestFallsThrough proves the controller degrades
// gracefully when the heaviest open dimension has no targeting skill: it falls
// through to the next actionable dimension and records the skip in the trace
// rationale, rather than stalling.
func TestSelector_UnactionableHeaviestFallsThrough(t *testing.T) {
	// business is the heaviest cluster (a BLOCKER, weight 8) but no skill targets
	// it; standards (a WARNING, weight 2) is actionable by refactor.
	state := findings.BuildState([]findings.Finding{
		finding("biz", "business", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER),
		finding("std", "standards", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor"}}

	got := NewSelector(coverageResolver()).SelectNextSkill(state, profile)

	if got.SkillID != "refactor" {
		t.Fatalf("expected fall-through to the actionable 'standards' skill, got %q (%s)", got.SkillID, got.Rationale)
	}
	if got.Dimension != dimensions.Dimension("standards") {
		t.Fatalf("expected dimension 'standards', got %q", got.Dimension)
	}
	if !strings.Contains(got.Rationale, "skipped") || !strings.Contains(got.Rationale, "business") {
		t.Fatalf("rationale must record the skipped unactionable dimension; got %q", got.Rationale)
	}
}

// TestSelector_AllUnactionableStalls confirms the honest terminal case: when
// EVERY open dimension is unactionable, selection returns no skill (the caller
// halts with nothing_actionable) — not a panic or a wrong pick.
func TestSelector_AllUnactionableStalls(t *testing.T) {
	state := findings.BuildState([]findings.Finding{
		finding("biz", "business", architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER),
		finding("dep", "dependencies", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor"}}

	got := NewSelector(coverageResolver()).SelectNextSkill(state, profile)
	if got.SkillID != "" {
		t.Fatalf("expected no actionable skill, got %q (%s)", got.SkillID, got.Rationale)
	}
	if !strings.Contains(got.Rationale, "no eligible skill") {
		t.Fatalf("rationale should explain nothing is actionable; got %q", got.Rationale)
	}
}
