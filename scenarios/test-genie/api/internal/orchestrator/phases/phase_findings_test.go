package phases

import (
	"encoding/json"
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	kov1 "github.com/vrooli/vrooli/packages/proto/gen/go/knowledge-observatory/v1"

	"test-genie/internal/eligibility"
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

func TestContractsArchFindings(t *testing.T) {
	rep := &cliHealthReport{
		Scenario: "demo",
		Findings: []cliHealthFinding{
			{Severity: "SEVERITY_ERROR", Code: "proto.orphan_method", Location: "cli/manifest.json", Message: "method not bound", Suggestion: "bind it"},
			{Severity: "SEVERITY_WARNING", Code: "manifest.note", Location: "", Message: "advisory"},
		},
	}
	got := contractsArchFindings("demo", rep)
	if len(got) != 2 {
		t.Fatalf("want 2 findings, got %d", len(got))
	}
	if got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_CLI {
		t.Errorf("source = %v, want CLI", got[0].Source)
	}
	if got[0].Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Errorf("severity = %v, want ERROR", got[0].Severity)
	}
	if got[0].Code != "proto.orphan_method" || got[0].Locations[0] != "cli/manifest.json" {
		t.Errorf("unexpected mapping: %+v", got[0])
	}
	if !strings.HasPrefix(got[0].StableId, "afid:") {
		t.Errorf("missing stable id: %q", got[0].StableId)
	}
	if len(got[1].Locations) != 0 {
		t.Errorf("blank location should be dropped, got %v", got[1].Locations)
	}
}

func TestUIHealthArchFindings(t *testing.T) {
	rep := &uiHealthReport{
		Scenario: "demo",
		Findings: []uiHealthFinding{
			{Severity: "SEVERITY_ERROR", Code: "slot.violation", Location: "ui/src/features/x", Message: "unknown slot"},
		},
	}
	got := uiHealthArchFindings("demo", rep)
	if len(got) != 1 || got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_UI {
		t.Fatalf("want 1 UI finding, got %+v", got)
	}
	if got[0].StableId == "" {
		t.Errorf("missing stable id")
	}
}

func TestDocsArchFindings(t *testing.T) {
	refPath := "docs/y.md"
	refLine := int32(12)
	resp := &kov1.DocHealthResponse{
		ScenarioName: "demo",
		MisplacedDocs: []*kov1.DocHealthMisplacedDoc{
			{ActualPath: "docs/x.md", ExpectedPath: "docs/concepts/x.md", Severity: kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_WARNING},
		},
		MissingDocs: []*kov1.DocHealthMissingDoc{
			{DocType: "ARCHITECTURE", Path: "docs/concepts/ARCHITECTURE.md", Severity: kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE},
		},
		ReferenceFindings: []*kov1.DocHealthFinding{
			{Code: "broken_link", Message: "dead link", Path: &refPath, Line: &refLine, Severity: kov1.DocHealthSeverity_DOC_HEALTH_SEVERITY_FAILURE},
		},
	}
	got := docsArchFindings("demo", resp)
	if len(got) != 3 {
		t.Fatalf("want 3 docs findings, got %d", len(got))
	}
	for _, f := range got {
		if f.Source != architecturev1.FindingSource_FINDING_SOURCE_DOCS {
			t.Errorf("source = %v, want DOCS", f.Source)
		}
		if f.StableId == "" {
			t.Errorf("missing stable id on %+v", f)
		}
	}
	// reference finding carries the family-prefixed code + path:line location.
	var ref *architecturev1.ArchitectureFinding
	for _, f := range got {
		if f.Code == "reference/broken_link" {
			ref = f
		}
	}
	if ref == nil {
		t.Fatalf("reference finding code not family-prefixed: %+v", got)
	}
	if ref.Locations[0] != "docs/y.md:12" {
		t.Errorf("location = %q, want docs/y.md:12", ref.Locations[0])
	}
	if ref.Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Errorf("FAILURE should map to ERROR, got %v", ref.Severity)
	}
}

func TestStandardsArchFindings(t *testing.T) {
	summary := &eligibility.ViolationSummary{
		Total: 3,
		TopViolations: []eligibility.ViolationExcerpt{
			{Severity: "high", RuleID: "type-safety", Title: "as-cast", FilePath: "api/x.ts", LineNumber: 4},
			{Severity: "medium", RuleID: "lint", Title: "", FilePath: "api/y.ts"},
		},
	}
	got := standardsArchFindings("demo", summary)
	if len(got) != 2 {
		t.Fatalf("want 2 standards findings, got %d", len(got))
	}
	if got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_STANDARDS {
		t.Errorf("source = %v, want STANDARDS", got[0].Source)
	}
	if got[0].Code != "type-safety" || got[0].Locations[0] != "api/x.ts:4" {
		t.Errorf("unexpected mapping: %+v", got[0])
	}
	if got[0].Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR { // high → ERROR
		t.Errorf("high should map to ERROR, got %v", got[0].Severity)
	}
	// blank title falls back to rule id as the message.
	if got[1].Message != "lint" {
		t.Errorf("blank title should fall back to rule id, got %q", got[1].Message)
	}
}

func TestStructureArchFindings(t *testing.T) {
	obs := []Observation{
		{Prefix: "ERROR", Text: "missing required dir: api/"},
		{Prefix: "WARNING", Text: "schema drift in service.json"},
		{Prefix: "SUCCESS", Text: "all dirs present"},
		{Section: "Structure", Icon: "✅"},
		{Prefix: "INFO", Text: "12 files checked"},
	}
	got := structureArchFindings("demo", obs)
	if len(got) != 2 {
		t.Fatalf("want 2 structure findings (error+warning only), got %d", len(got))
	}
	if got[0].Source != architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE {
		t.Errorf("source = %v, want STRUCTURE", got[0].Source)
	}
	// message is the identity (code == message until coded findings exist).
	if got[0].Code != got[0].Message {
		t.Errorf("structure code should equal message, got code=%q msg=%q", got[0].Code, got[0].Message)
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
