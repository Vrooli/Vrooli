package report

import (
	"bytes"
	"strings"
	"testing"

	execTypes "test-genie/cli/internal/execute"
)

func nonMaxStanding() *execTypes.MaturityStanding {
	return &execTypes.MaturityStanding{
		Phase:                   "contracts",
		CurrentLevel:            "L2",
		CurrentLevelLabel:       "Ready",
		NextLevel:               "L3",
		CeilingLevel:            "L4",
		NorthStar:               "Verified renderer-separated primitives.",
		NextMove:                "Prove each declared primitive with cli-core evidence.",
		PriorityCapabilityLabel: "Command Architecture",
		BlockingFindingCodes:    []string{"arch.primitive_unverified"},
		DocSearchTopics:         []string{"contracts arch.primitive_unverified canonical fix"},
	}
}

func renderStanding(t *testing.T, phase execTypes.Phase) string {
	t.Helper()
	var buf bytes.Buffer
	pr := New(&buf, "cli-health", "", nil, nil, false, nil, nil)
	pr.printPhaseResults([]execTypes.Phase{phase})
	return buf.String()
}

func TestScorecardShowsRungGapsNextMoveAndDocQuery(t *testing.T) {
	out := renderStanding(t, execTypes.Phase{Name: "contracts", Status: "passed", MaturityStanding: nonMaxStanding()})

	for _, want := range []string{
		"standing:",
		"L2 Ready → L3", "(ceiling L4)", // rung + distinct ceiling
		"North Star: Verified renderer-separated primitives.",
		"gaps: arch.primitive_unverified",
		"next: Prove each declared primitive with cli-core evidence.",
		"[→ Command Architecture]",
		`search-hub query "contracts arch.primitive_unverified canonical fix" --type doc`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("scorecard missing %q\n---\n%s", want, out)
		}
	}
}

func TestScorecardIsConcise(t *testing.T) {
	out := renderStanding(t, execTypes.Phase{Name: "contracts", Status: "passed", MaturityStanding: nonMaxStanding()})
	// The scorecard block (indented "     " lines) stays within a tight budget:
	// standing / North Star / gaps / next / docs = 5 lines.
	count := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "     ") {
			count++
		}
	}
	if count > 6 {
		t.Fatalf("scorecard exceeded the concise budget: %d indented lines\n%s", count, out)
	}
}

func TestScorecardSuppressedAtMaximum(t *testing.T) {
	st := nonMaxStanding()
	st.AtMaximum = true
	st.NextLevel = ""
	st.BlockingFindingCodes = nil
	st.DocSearchTopics = nil
	out := renderStanding(t, execTypes.Phase{Name: "contracts", Status: "passed", MaturityStanding: st})

	if !strings.Contains(out, "maximum maturity") {
		t.Errorf("expected a maximum-maturity marker\n%s", out)
	}
	if strings.Contains(out, "next:") {
		t.Errorf("next-move line must be suppressed at maximum maturity\n%s", out)
	}
	if strings.Contains(out, "search-hub query") {
		t.Errorf("doc line must be suppressed at maximum maturity\n%s", out)
	}
}

func TestScorecardAbsentWhenNoStanding(t *testing.T) {
	out := renderStanding(t, execTypes.Phase{Name: "unit", Status: "passed"})
	if strings.Contains(out, "standing:") {
		t.Errorf("no scorecard expected for a phase without a standing\n%s", out)
	}
}
