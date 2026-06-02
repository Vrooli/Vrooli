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
	// No error-level standards findings, but 5 warnings; the count cap tightens R1.
	sig := allClean()
	sig.CountByDimension[dimd("standards")] = 5
	th := DefaultThresholds()
	if _, ok := Lowest(sig, th, ""); ok {
		t.Error("without a count cap, warning-only standards must satisfy R1")
	}
	th.StandardsMaxCount = 3
	r, ok := Lowest(sig, th, "")
	if !ok || r.ID != RungR1 {
		t.Fatalf("count cap must reopen R1, got %v ok=%v", r.ID, ok)
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
