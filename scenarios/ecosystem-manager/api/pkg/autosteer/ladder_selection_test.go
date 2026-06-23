package autosteer

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/completeness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
	"github.com/vrooli/maturity-go/ladder"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// ladderResolver maps one skill to several dimensions so any rung's work is
// actionable in these selection tests.
func ladderResolver() *skillmap.Resolver {
	return skillmap.NewResolverWithWarner(&skillmap.FakeCatalog{Declarations: []skillmap.SkillDeclaration{
		{ID: "refactor", Dimensions: []string{"standards", "structure", "security", "coverage", "tidiness", "cycles", "tests", "docs", "contracts"}},
	}}, func(string, ...any) {})
}

func ladderProfile(top string) *AutoSteerProfile {
	return &AutoSteerProfile{
		AllowedSkills: []string{"refactor"},
		Ladder:        &LadderObjective{Enabled: true, TopRung: top, BoostFactor: 8},
	}
}

func ladderSelector(score completeness.Score, profile *AutoSteerProfile) *Selector {
	return NewSelector(ladderResolver()).WithLadder(newLadderRuntime(profile, score))
}

// scoreAtRung is the completeness measurement EM now consumes (plan D4): the
// working rung is whatever completeness-scoring reported as the lowest
// unsatisfied rung; EM applies governance for it rather than re-deriving it from
// findings.
func scoreAtRung(rung ladder.RungID) completeness.Score {
	return completeness.Score{WorkingRung: string(rung), BuildPassing: true, OTKnown: true}
}

// TestLadder_HardGateRestrictsToRung proves a hard rung restricts selection to
// its own dimensions even when a higher-weighted dimension is open elsewhere.
func TestLadder_HardGateRestrictsToRung(t *testing.T) {
	// security ERROR (R1) + a heavier coverage WARNING (R3). completeness reports
	// R1 as the working rung, so selection must land on security.
	state := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := ladderProfile("R4")
	got := ladderSelector(scoreAtRung(ladder.RungR1), profile).SelectNextSkill(state, profile)

	if got.Dimension != dimensions.Dimension("security") {
		t.Fatalf("hard R1 must restrict to security, got %q", got.Dimension)
	}
	if got.CurrentRung != string(ladder.RungR1) {
		t.Fatalf("CurrentRung = %q, want R1", got.CurrentRung)
	}
}

// TestLadder_SoftBoostAmplifiesButKeepsOthers proves a soft rung amplifies its
// dimensions yet leaves others eligible (no hard restriction).
func TestLadder_SoftBoostAmplifiesButKeepsOthers(t *testing.T) {
	// A coverage ERROR (R3 soft) and a docs ERROR (R2 soft) are both open.
	// completeness reports R2 as the working rung, so its docs dimension is boosted
	// and chosen even though coverage has the same base weight.
	state := findings.BuildState([]findings.Finding{
		finding("doc", "docs", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	profile := ladderProfile("R4")
	got := ladderSelector(scoreAtRung(ladder.RungR2), profile).SelectNextSkill(state, profile)

	if got.Dimension != dimensions.Dimension("docs") {
		t.Fatalf("soft R2 boost must select docs, got %q", got.Dimension)
	}
	if got.CurrentRung != string(ladder.RungR2) {
		t.Fatalf("CurrentRung = %q, want R2", got.CurrentRung)
	}
}

// TestLadder_WorkingRungDrivesSelection proves the consumed rung — not EM's own
// re-derivation — governs which dimension the controller works (plan D4): the
// same findings select different dimensions depending on the rung completeness
// reported.
func TestLadder_WorkingRungDrivesSelection(t *testing.T) {
	profile := ladderProfile("R4")

	// Round 1: completeness reports R1 with a security ERROR (R1) and a coverage
	// ERROR (R3) both open → hard R1 restricts to security.
	round1 := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	got1 := ladderSelector(scoreAtRung(ladder.RungR1), profile).SelectNextSkill(round1, profile)
	if got1.CurrentRung != string(ladder.RungR1) || got1.Dimension != dimensions.Dimension("security") {
		t.Fatalf("round1 must work R1/security, got %q/%q", got1.CurrentRung, got1.Dimension)
	}

	// Round 2: security closed; completeness now reports R3 with only the coverage
	// ERROR open → soft R3 boosts coverage.
	round2 := findings.BuildState([]findings.Finding{
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	got2 := ladderSelector(scoreAtRung(ladder.RungR3), profile).SelectNextSkill(round2, profile)
	if got2.CurrentRung != string(ladder.RungR3) || got2.Dimension != dimensions.Dimension("coverage") {
		t.Fatalf("round2 must climb to R3/coverage, got %q/%q", got2.CurrentRung, got2.Dimension)
	}

	// Round 3: completeness drops back to R1 (R3 work re-opened a security gap).
	got3 := ladderSelector(scoreAtRung(ladder.RungR1), profile).SelectNextSkill(round1, profile)
	if got3.CurrentRung != string(ladder.RungR1) {
		t.Fatalf("round3 must drop back to R1, got %q", got3.CurrentRung)
	}
}

// TestLadder_CleanToTopImposesNoConstraint proves that when completeness reports
// the ladder clean to the profile's top rung, selection is pure weighted-greedy
// (no rung label).
func TestLadder_CleanToTopImposesNoConstraint(t *testing.T) {
	state := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := ladderProfile("R4")
	clean := completeness.Score{LadderClean: true, BuildPassing: true, OTKnown: true}
	got := ladderSelector(clean, profile).SelectNextSkill(state, profile)
	if got.CurrentRung != "" {
		t.Errorf("ladder-clean score must yield empty CurrentRung, got %q", got.CurrentRung)
	}
	if got.Dimension != dimensions.Dimension("security") {
		t.Errorf("greedy must pick the heaviest dimension security, got %q", got.Dimension)
	}
}

// TestLadder_WorkingRungAboveTopRungImposesNoConstraint proves a profile that
// only pursues a low rung treats an unsatisfied higher rung as "clean to top".
func TestLadder_WorkingRungAboveTopRungImposesNoConstraint(t *testing.T) {
	state := findings.BuildState([]findings.Finding{
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	// Profile caps at R1; completeness reports the lowest unsatisfied rung as R3
	// (above the ceiling) → no constraint, pure greedy.
	profile := ladderProfile("R1")
	got := ladderSelector(scoreAtRung(ladder.RungR3), profile).SelectNextSkill(state, profile)
	if got.CurrentRung != "" {
		t.Errorf("working rung above top rung must yield empty CurrentRung, got %q", got.CurrentRung)
	}
}

// TestLadder_DisabledIsUnchanged confirms a profile without a ladder selects
// exactly as the weighted-greedy selector (no rung label).
func TestLadder_DisabledIsUnchanged(t *testing.T) {
	state := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := &AutoSteerProfile{AllowedSkills: []string{"refactor"}}
	got := NewSelector(ladderResolver()).WithLadder(newLadderRuntime(profile, completeness.Score{})).SelectNextSkill(state, profile)
	if got.CurrentRung != "" {
		t.Errorf("no-ladder profile must have empty CurrentRung, got %q", got.CurrentRung)
	}
	// security (ERROR, weight 4) outranks coverage (WARNING, weight 2).
	if got.Dimension != dimensions.Dimension("security") {
		t.Errorf("greedy must pick the heaviest dimension security, got %q", got.Dimension)
	}
}

// TestLadder_HardGateNoOpenFindingDoesNotStall proves that when a hard rung is
// unsatisfied only because the build is failing (no actionable finding in the
// rung's dimensions), selection still proceeds on the full ranking.
func TestLadder_HardGateNoOpenFindingDoesNotStall(t *testing.T) {
	// completeness reports R0 (build failing), but the only open finding is in docs
	// (not an R0 dimension). Selection must not stall.
	state := findings.BuildState([]findings.Finding{
		finding("doc", "docs", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := ladderProfile("R4")
	score := completeness.Score{WorkingRung: string(ladder.RungR0), BuildPassing: false, OTKnown: true}
	got := ladderSelector(score, profile).SelectNextSkill(state, profile)
	if got.SkillID == "" {
		t.Fatal("must not stall when the hard rung has no open finding to act on")
	}
	if got.CurrentRung != string(ladder.RungR0) {
		t.Errorf("CurrentRung should still report R0, got %q", got.CurrentRung)
	}
}
