package findings

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func loadFixtureAudit(t *testing.T) *Audit {
	t.Helper()
	a, err := LoadAuditFile(filepath.Join("testdata", "audit.json"))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	return a
}

func TestToFindingsResolvesDimensions(t *testing.T) {
	found := ToFindings(loadFixtureAudit(t))
	if len(found) == 0 {
		t.Fatal("no findings parsed from fixture")
	}
	for _, f := range found {
		if !dimensions.IsValid(f.Dimension) {
			t.Errorf("finding %s resolved to invalid dimension %q", f.ID, f.Dimension)
		}
		if f.ID == "" {
			t.Errorf("finding has empty id (phase %q)", f.Phase)
		}
	}
}

func TestToFindingsSourceAndPhaseMapping(t *testing.T) {
	found := ToFindings(loadFixtureAudit(t))
	byID := map[string]Finding{}
	for _, f := range found {
		byID[f.ID] = f
	}

	// Structured finding from the architecture phase (source ARCHITECTURE=6) →
	// cycles, severity BLOCKER.
	if f, ok := byID["afid:0000000000000007"]; !ok {
		t.Error("expected architecture cycle finding")
	} else {
		if f.Dimension != dimensions.Dimension("cycles") {
			t.Errorf("architecture finding dimension = %q, want cycles", f.Dimension)
		}
		if f.Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER {
			t.Errorf("architecture finding severity = %v, want BLOCKER", f.Severity)
		}
	}

	// The unit phase failed with no structured findings → one synthetic finding
	// in the tests dimension.
	if f, ok := byID["phase:unit"]; !ok {
		t.Error("expected synthetic finding for failing unit phase")
	} else {
		if !f.Synthetic {
			t.Error("unit-phase finding should be synthetic")
		}
		if f.Dimension != dimensions.Dimension("tests") {
			t.Errorf("unit-phase dimension = %q, want tests", f.Dimension)
		}
	}

	// Passing phases (dependencies, integration, …) must not produce findings.
	if _, ok := byID["phase:dependencies"]; ok {
		t.Error("passing dependencies phase should not yield a finding")
	}
}

func TestBuildStateScoresAndFingerprint(t *testing.T) {
	found := ToFindings(loadFixtureAudit(t))
	st := BuildState(found)

	if st.TotalScore <= 0 {
		t.Fatalf("total score = %v, want positive", st.TotalScore)
	}
	if len(st.DimensionScore) == 0 {
		t.Fatal("no dimension scores")
	}
	// standards has two findings (ERROR=4 + WARNING=2) plus lint's GOFUMPT
	// (WARNING=2) → 8.
	if got := st.DimensionScore[dimensions.Dimension("standards")]; got != 8 {
		t.Errorf("standards score = %v, want 8", got)
	}
	if got := st.DimensionCount[dimensions.Dimension("standards")]; got != 3 {
		t.Errorf("standards count = %v, want 3", got)
	}

	// Heaviest ordering is deterministic and score-descending.
	heaviest := st.HeaviestDimensions()
	for i := 1; i < len(heaviest); i++ {
		if st.DimensionScore[heaviest[i-1]] < st.DimensionScore[heaviest[i]] {
			t.Errorf("heaviest not sorted descending at %d", i)
		}
	}

	// Fingerprint is stable and order-independent.
	if st.Fingerprint == "" {
		t.Error("empty fingerprint for non-empty state")
	}
	reordered := append([]Finding{found[len(found)-1]}, found[:len(found)-1]...)
	if BuildState(reordered).Fingerprint != st.Fingerprint {
		t.Error("fingerprint changed under reordering")
	}
}

func TestEmptyAuditYieldsEmptyState(t *testing.T) {
	st := BuildState(ToFindings(&Audit{ScenarioName: "x", Phases: nil}))
	if st.TotalScore != 0 || len(st.Findings) != 0 || st.Fingerprint != "" {
		t.Errorf("expected empty state, got %+v", st)
	}
}

func TestTestGenieRunnerArgs(t *testing.T) {
	r := &TestGenieRunner{}
	cases := []struct {
		name string
		req  AuditRequest
		want []string
	}{
		{"default preset", AuditRequest{Scenario: "foo"}, []string{"execute", "foo", "--json", "--preset", "comprehensive"}},
		{"explicit preset", AuditRequest{Scenario: "foo", Preset: "quick"}, []string{"execute", "foo", "--json", "--preset", "quick"}},
		{"scoped phases", AuditRequest{Scenario: "foo", Preset: "quick", Phases: []string{"standards", "lint"}}, []string{"execute", "foo", "--json", "--phases", "standards,lint"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := r.Args(tc.req)
			if len(got) != len(tc.want) {
				t.Fatalf("args = %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("args = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestFakeRunner(t *testing.T) {
	fx := loadFixtureAudit(t)
	r := &FakeRunner{Audits: map[string]*Audit{"fixture-scenario": fx}}
	got, err := r.Audit(context.Background(), AuditRequest{Scenario: "fixture-scenario", Preset: "comprehensive"})
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if got.ScenarioName != "fixture-scenario" {
		t.Errorf("scenario = %q", got.ScenarioName)
	}
	if len(r.Calls) != 1 || r.Calls[0].Preset != "comprehensive" {
		t.Errorf("calls not recorded: %+v", r.Calls)
	}
	if _, err := r.Audit(context.Background(), AuditRequest{Scenario: "missing"}); err == nil {
		t.Error("expected error for unconfigured scenario")
	}
}

func TestAudit_Conclusive(t *testing.T) {
	cases := []struct {
		name  string
		audit *Audit
		want  bool
	}{
		{"nil", nil, false},
		{"no phases", &Audit{}, false},
		{"all skipped", &Audit{Phases: []AuditPhase{
			{Name: "standards", Status: "skipped"},
			{Name: "docs", Status: "pending"},
			{Name: "smoke", Status: ""},
		}}, false},
		{"one executed pass", &Audit{Phases: []AuditPhase{
			{Name: "standards", Status: "skipped"},
			{Name: "docs", Status: "pass"},
		}}, true},
		{"executed fail", &Audit{Phases: []AuditPhase{
			{Name: "standards", Status: "fail"},
		}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.audit.Conclusive(); got != tc.want {
				t.Errorf("Conclusive() = %v, want %v", got, tc.want)
			}
		})
	}
}
