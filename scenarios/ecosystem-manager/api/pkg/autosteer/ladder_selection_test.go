package autosteer

import (
	"testing"

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

func ladderSelector(metrics MetricsSnapshot, profile *AutoSteerProfile) *Selector {
	return NewSelector(ladderResolver()).WithLadder(newLadderRuntime(profile, metrics))
}

// passingMetrics is a green build with operational targets met.
func passingMetrics() MetricsSnapshot {
	return MetricsSnapshot{BuildStatus: 1, OperationalTargetsTotal: 1, OperationalTargetsPassing: 1, OperationalTargetsPercentage: 100}
}

// TestLadder_HardGateRestrictsToRung proves a hard rung restricts selection to
// its own dimensions even when a higher-weighted dimension is open elsewhere.
func TestLadder_HardGateRestrictsToRung(t *testing.T) {
	// security ERROR (R1, weight whatever) + a heavier coverage WARNING (R3).
	// R1 is the lowest unsatisfied HARD rung, so selection must land on security.
	state := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := ladderProfile("R4")
	got := ladderSelector(passingMetrics(), profile).SelectNextSkill(state, profile)

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
	// No hard-rung gaps (build green, no security/standards/structure errors). A
	// coverage ERROR (R3 soft) and a docs ERROR (R2 soft) are both open. R2 is the
	// lower unsatisfied soft rung, so its docs dimension is boosted and chosen even
	// though coverage has the same base weight.
	state := findings.BuildState([]findings.Finding{
		finding("doc", "docs", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	profile := ladderProfile("R4")
	got := ladderSelector(passingMetrics(), profile).SelectNextSkill(state, profile)

	if got.Dimension != dimensions.Dimension("docs") {
		t.Fatalf("soft R2 boost must select docs, got %q", got.Dimension)
	}
	if got.CurrentRung != string(ladder.RungR2) {
		t.Fatalf("CurrentRung = %q, want R2", got.CurrentRung)
	}
}

// TestLadder_Oscillation proves that closing a low-rung gap lets a higher rung
// take over, and re-opening it drops selection back — the core ladder behavior.
func TestLadder_Oscillation(t *testing.T) {
	profile := ladderProfile("R4")

	// Round 1: a security ERROR (R1) and a coverage ERROR (R3) are both open.
	round1 := findings.BuildState([]findings.Finding{
		finding("sec", "security", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	got1 := ladderSelector(passingMetrics(), profile).SelectNextSkill(round1, profile)
	if got1.CurrentRung != string(ladder.RungR1) {
		t.Fatalf("round1 must work R1, got %q", got1.CurrentRung)
	}

	// Round 2: security closed; only the coverage ERROR remains → R3.
	round2 := findings.BuildState([]findings.Finding{
		finding("cov", "coverage", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR),
	})
	got2 := ladderSelector(passingMetrics(), profile).SelectNextSkill(round2, profile)
	if got2.CurrentRung != string(ladder.RungR3) {
		t.Fatalf("round2 must climb to R3, got %q", got2.CurrentRung)
	}

	// Round 3: R3 work re-introduced a security ERROR → drops back to R1.
	got3 := ladderSelector(passingMetrics(), profile).SelectNextSkill(round1, profile)
	if got3.CurrentRung != string(ladder.RungR1) {
		t.Fatalf("round3 must drop back to R1, got %q", got3.CurrentRung)
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
	got := NewSelector(ladderResolver()).WithLadder(newLadderRuntime(profile, passingMetrics())).SelectNextSkill(state, profile)
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
	// Build failing → R0 unsatisfied, but the only open finding is in docs (not an
	// R0 dimension). Selection must not stall.
	state := findings.BuildState([]findings.Finding{
		finding("doc", "docs", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING),
	})
	profile := ladderProfile("R4")
	metrics := passingMetrics()
	metrics.BuildStatus = 0 // failing → R0 hard rung unsatisfied
	got := ladderSelector(metrics, profile).SelectNextSkill(state, profile)
	if got.SkillID == "" {
		t.Fatal("must not stall when the hard rung has no open finding to act on")
	}
	if got.CurrentRung != string(ladder.RungR0) {
		t.Errorf("CurrentRung should still report R0, got %q", got.CurrentRung)
	}
}
