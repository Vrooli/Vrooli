package phases

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// TestNormalizeFindingSeverity is the R3 anti-drift guard: every severity
// vocabulary any producer emits must map to a defined ladder rung. Bare
// and SEVERITY_-prefixed forms normalize identically.
func TestNormalizeFindingSeverity(t *testing.T) {
	cases := map[string]architecturev1.FindingSeverity{
		"blocker":          architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER,
		"SEVERITY_BLOCKER": architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER,
		"error":            architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"SEVERITY_ERROR":   architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"failure":          architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"critical":         architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"high":             architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"warn":             architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		"warning":          architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		"SEVERITY_WARNING": architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		"medium":           architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		"info":             architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		"notice":           architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		"low":              architecturev1.FindingSeverity_FINDING_SEVERITY_INFO,
		"  Error  ":        architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		"":                 architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED,
		"bogus":            architecturev1.FindingSeverity_FINDING_SEVERITY_UNSPECIFIED,
	}
	for in, want := range cases {
		if got := normalizeFindingSeverity(in); got != want {
			t.Errorf("normalizeFindingSeverity(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestUIHealthArchFindings(t *testing.T) {
	rep := &commonv1.MaturityAssessment{
		Scenario: "demo",
		Findings: []*commonv1.AssessmentFinding{
			{Severity: "SEVERITY_ERROR", Code: "slot.violation", Location: "ui/src/features/x", Message: "unknown slot"},
		},
	}
	got := assessment.AssessmentToArchitectureFindings("demo", rep, architecturev1.FindingSource_FINDING_SOURCE_UI)
	if len(got) != 1 || got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_UI {
		t.Fatalf("want 1 UI finding, got %+v", got)
	}
	if got[0].StableId == "" {
		t.Errorf("missing stable id")
	}
}

// TestExecutionResultJSONCarriesFindings asserts the suite `--json` shape
// gains a `findings` array while leaving `observations` intact. This is the
// machine seam `campaign create --from-audit` reads.
func TestExecutionResultJSONCarriesFindings(t *testing.T) {
	res := ExecutionResult{
		Name:   "contracts",
		Status: "passed",
		Observations: []Observation{
			{Prefix: "INFO", Text: "ok"},
		},
		Findings: []*architecturev1.ArchitectureFinding{
			newFinding("demo", architecturev1.FindingSource_FINDING_SOURCE_CLI, "c", "error", "m", "", []string{"a"}, nil),
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"findings"`) {
		t.Errorf("json missing findings array: %s", s)
	}
	if !strings.Contains(s, `"observations"`) {
		t.Errorf("json missing observations (regression): %s", s)
	}
	if !strings.Contains(s, "afid:") {
		t.Errorf("json findings missing stable id: %s", s)
	}

	// Round-trips back into the shared proto type (what the cartographer
	// ingest will do).
	var back ExecutionResult
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(back.Findings) != 1 || back.Findings[0].StableId != res.Findings[0].StableId {
		t.Errorf("round-trip lost findings: %+v", back.Findings)
	}
}

// TestNewFindingPopulatesPerSourceEffort pins the documented per-source
// effort heuristic. Effort is advisory ranking input (the campaign tracker's
// FAST/LONG_TERM profiles) and must be populated by every emitter; each
// phase maps to one source, so the per-source default is the per-phase
// default.
func TestNewFindingPopulatesPerSourceEffort(t *testing.T) {
	cases := []struct {
		source architecturev1.FindingSource
		want   architecturev1.EffortHint
	}{
		{architecturev1.FindingSource_FINDING_SOURCE_DOCS, architecturev1.EffortHint_EFFORT_HINT_TRIVIAL},
		{architecturev1.FindingSource_FINDING_SOURCE_CLI, architecturev1.EffortHint_EFFORT_HINT_SMALL},
		{architecturev1.FindingSource_FINDING_SOURCE_UI, architecturev1.EffortHint_EFFORT_HINT_SMALL},
		{architecturev1.FindingSource_FINDING_SOURCE_STANDARDS, architecturev1.EffortHint_EFFORT_HINT_SMALL},
		{architecturev1.FindingSource_FINDING_SOURCE_DEPENDENCY, architecturev1.EffortHint_EFFORT_HINT_SMALL},
		{architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE, architecturev1.EffortHint_EFFORT_HINT_LARGE},
		{architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE, architecturev1.EffortHint_EFFORT_HINT_LARGE},
	}
	for _, tc := range cases {
		f := newFinding("demo", tc.source, "code", "error", "m", "", []string{"a"}, nil)
		if f.GetEffort() != tc.want {
			t.Errorf("source %v effort = %v, want %v", tc.source, f.GetEffort(), tc.want)
		}
	}

	// Effort is EXCLUDED from the stable-id hash: two findings differing only
	// in effort must collapse to the same afid.
	base := newFinding("demo", architecturev1.FindingSource_FINDING_SOURCE_CLI, "code", "error", "m", "", []string{"a"}, nil)
	override := newFindingWithEffort("demo", architecturev1.FindingSource_FINDING_SOURCE_CLI, "code", "error", "m", "", []string{"a"}, nil,
		architecturev1.EffortHint_EFFORT_HINT_LARGE)
	if base.GetStableId() != override.GetStableId() {
		t.Errorf("effort must not change the stable id: %q vs %q", base.GetStableId(), override.GetStableId())
	}
}
