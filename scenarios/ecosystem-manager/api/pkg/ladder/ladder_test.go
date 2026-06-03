package ladder

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/dimensions"
)

func dimd(s string) dimensions.Dimension { return dimensions.Dimension(s) }

// allClean is a signal set where every rung is satisfied.
func allClean() Signals {
	return Signals{
		ErrorPlusByDimension: map[dimensions.Dimension]int{},
		CountByDimension:     map[dimensions.Dimension]int{},
		BuildPassing:         true,
		OTPercentage:         100,
		OTTarget:             90,
		OTHasTargets:         true,
		OTKnown:              true,
	}
}

func TestLowest_R0WhenBuildFailing(t *testing.T) {
	sig := allClean()
	sig.BuildPassing = false
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR0 {
		t.Fatalf("build failing must select R0, got %v ok=%v", r.ID, ok)
	}
	if !r.HardGate {
		t.Error("R0 must be a hard gate")
	}
}

func TestLowest_R1WhenSecurityError(t *testing.T) {
	sig := allClean()
	sig.ErrorPlusByDimension[dimd("security")] = 1
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR1 {
		t.Fatalf("security error must select R1, got %v ok=%v", r.ID, ok)
	}
}

func TestLowest_R2WhenCyclesPresent(t *testing.T) {
	sig := allClean()
	sig.CountByDimension[dimd("cycles")] = 2
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR2 {
		t.Fatalf("cycles must select R2, got %v ok=%v", r.ID, ok)
	}
	if r.HardGate {
		t.Error("R2 must be a soft gate")
	}
}

func TestLowest_R3WhenCoverageError(t *testing.T) {
	sig := allClean()
	sig.ErrorPlusByDimension[dimd("coverage")] = 1
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR3 {
		t.Fatalf("coverage error must select R3, got %v ok=%v", r.ID, ok)
	}
}

func TestLowest_R4WhenOTBelowTarget(t *testing.T) {
	sig := allClean()
	sig.OTPercentage = 70
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR4 {
		t.Fatalf("OT below target must select R4, got %v ok=%v", r.ID, ok)
	}
}

func TestLowest_AllCleanNoConstraint(t *testing.T) {
	if _, ok := Lowest(allClean(), DefaultThresholds(), ""); ok {
		t.Error("a fully clean scenario must impose no rung constraint")
	}
	if !AllHold(allClean(), DefaultThresholds(), "") {
		t.Error("AllHold must be true for a clean scenario")
	}
}

func TestLowest_LowestWins(t *testing.T) {
	// Both an R1 (security) and an R3 (coverage) gap open: the LOWER rung wins.
	sig := allClean()
	sig.ErrorPlusByDimension[dimd("security")] = 1
	sig.ErrorPlusByDimension[dimd("coverage")] = 1
	r, _ := Lowest(sig, DefaultThresholds(), "")
	if r.ID != RungR1 {
		t.Fatalf("lowest unsatisfied rung must win, got %v", r.ID)
	}
}

func TestTopRungCap(t *testing.T) {
	// Only an R4 (OT) gap is open. With top_rung=R3 the ladder ignores R4.
	sig := allClean()
	sig.OTPercentage = 50
	if _, ok := Lowest(sig, DefaultThresholds(), RungR3); ok {
		t.Error("top_rung=R3 must ignore an R4 gap")
	}
	if !AllHold(sig, DefaultThresholds(), RungR3) {
		t.Error("AllHold up to R3 must be true when only R4 is open")
	}
}

func TestStandardsCountCap(t *testing.T) {
	// 5 warning-level standards findings, under the default density cap (10) ⇒ R1
	// still holds; a tighter standards-specific override (3) reopens it.
	sig := allClean()
	sig.CountByDimension[dimd("standards")] = 5
	th := DefaultThresholds()
	if _, ok := Lowest(sig, th, ""); ok {
		t.Error("5 standards warnings (< default cap 10) must still satisfy R1")
	}
	th.StandardsMaxCount = 3
	r, ok := Lowest(sig, th, "")
	if !ok || r.ID != RungR1 {
		t.Fatalf("standards count cap must reopen R1, got %v ok=%v", r.ID, ok)
	}
}

// TestWarningDensityHoldsR1 is the fix-#1 regression: a warning-only scenario
// with a finding backlog above the default density cap must NOT be waved through
// R1 as "safe". This is the accessibility-compliance-hub case (95 security / 21
// standards warnings, zero errors) that previously left the ladder inert.
func TestWarningDensityHoldsR1(t *testing.T) {
	sig := allClean()
	sig.CountByDimension[dimd("security")] = 95
	sig.CountByDimension[dimd("standards")] = 21
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR1 {
		t.Fatalf("warning-density backlog must hold R1, got %v ok=%v", r.ID, ok)
	}
	if !r.HardGate {
		t.Error("R1 must be a hard gate so selection is restricted to its dimensions")
	}
	// Once the backlog clears below the cap, R1 releases and the ladder advances.
	sig.CountByDimension[dimd("security")] = 4
	sig.CountByDimension[dimd("standards")] = 2
	if _, ok := Lowest(sig, DefaultThresholds(), ""); ok {
		t.Error("cleared backlog (< cap) must release R1")
	}
}

// TestR4UnknownNotVacuous is the fix-#2 regression: when the operational-targets
// metric was not collected (OTKnown == false — a best-effort failure), R4 must be
// treated as unsatisfied rather than silently "met". Previously the silent zero
// was indistinguishable from "no targets" and no-opped the only non-error rung.
func TestR4UnknownNotVacuous(t *testing.T) {
	sig := allClean() // every lower rung clean
	sig.OTKnown = false
	r, ok := Lowest(sig, DefaultThresholds(), "")
	if !ok || r.ID != RungR4 {
		t.Fatalf("unknown OT metric must hold R4, got %v ok=%v", r.ID, ok)
	}
	// Collected + genuinely no targets ⇒ R4 vacuously satisfied (the legitimate case).
	sig.OTKnown = true
	sig.OTHasTargets = false
	if _, ok := Lowest(sig, DefaultThresholds(), ""); ok {
		t.Error("collected-but-no-targets must satisfy R4 (no constraint)")
	}
}

func TestParseRung(t *testing.T) {
	for _, in := range []string{"R3", "r3", "R3 Features hardened"} {
		if got, ok := ParseRung(in); !ok || got != RungR3 {
			t.Errorf("ParseRung(%q) = %v ok=%v, want R3", in, got, ok)
		}
	}
	if _, ok := ParseRung("R9"); ok {
		t.Error("ParseRung(R9) must fail")
	}
	if _, ok := ParseRung(""); ok {
		t.Error("ParseRung(empty) must fail")
	}
}
