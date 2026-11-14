package report

import (
	"bytes"
	"strings"
	"testing"
	"time"

	execTypes "test-genie/cli/internal/execute"
)

// baseResp returns a minimal passing response we can attach a Requirements
// summary to for rendering assertions.
func baseResp() execTypes.Response {
	return execTypes.Response{
		Success:     true,
		Verdict:     "PASS",
		StartedAt:   time.Now().UTC().Format(time.RFC3339),
		CompletedAt: time.Now().Add(5 * time.Second).UTC().Format(time.RFC3339),
		PhaseSummary: execTypes.PhaseSummary{
			Total:           1,
			Passed:          1,
			DurationSeconds: 5,
		},
		Phases: []execTypes.Phase{
			{Name: "unit", Status: "passed", DurationSeconds: 5},
		},
	}
}

func renderRequirements(t *testing.T, r *execTypes.RequirementsSummary) string {
	t.Helper()
	resp := baseResp()
	resp.Requirements = r
	var buf bytes.Buffer
	// No-color writer: a plain *bytes.Buffer is not a TTY, so Color is disabled
	// and assertions can match raw substrings.
	pr := New(&buf, "demo-scenario", "", nil, nil, false, nil, nil)
	pr.Print(resp)
	return buf.String()
}

func TestRequirementsSummaryAbsentWhenNil(t *testing.T) {
	out := renderRequirements(t, nil)
	if strings.Contains(out, "OPERATIONAL TARGETS") {
		t.Fatalf("did not expect requirements block when summary is nil:\n%s", out)
	}
}

func TestRequirementsSummaryStaleBranch(t *testing.T) {
	r := &execTypes.RequirementsSummary{
		Synced:       false,
		SkipReason:   "required phases skipped: integration",
		LastSyncedAt: "2026-05-29T10:00:00Z",
		OTComplete:   3,
		OTTotal:      7,
		OTByPriority: map[string]execTypes.OTCount{
			"P0": {Complete: 2, Total: 2},
			"P1": {Complete: 1, Total: 3},
			"P2": {Complete: 0, Total: 2},
		},
		ReqComplete: 18,
		ReqTotal:    25,
		ReqByStatus: map[string]int{"complete": 18, "in_progress": 4, "planned": 2, "pending": 1},
	}
	out := renderRequirements(t, r)

	for _, token := range []string{
		"REQUIREMENTS & OPERATIONAL TARGETS:",
		"Operational targets: 3/7 complete",
		"P0 2/2 · P1 1/3 · P2 0/2",
		"Requirements: 18/25 complete",
		"in_progress 4 · planned 2 · pending 1",
		"Not updated this run",
		"required phases skipped: integration",
		"last full sync (2026-05-29)",
		"To refresh:",
		"test-genie execute demo-scenario",
		"STATUS_MODEL.md",
		"IMPROVING_COVERAGE.md",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("stale branch missing %q\n----\n%s\n----", token, out)
		}
	}
}

func TestRequirementsSummaryFreshWithPromotions(t *testing.T) {
	r := &execTypes.RequirementsSummary{
		Synced:      true,
		OTComplete:  4,
		OTTotal:     7,
		ReqComplete: 20,
		ReqTotal:    25,
		Changes: []execTypes.RequirementChange{
			{ID: "REQ-AB-001", PRDRef: "OT-P1-002", From: "in_progress", To: "complete", Kind: execTypes.ChangeKindPromotion},
		},
	}
	out := renderRequirements(t, r)

	for _, token := range []string{
		"REQUIREMENTS & OPERATIONAL TARGETS:",
		"now complete:",
		"REQ-AB-001",
		"OT-P1-002",
		"in_progress → complete",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("fresh/promotion branch missing %q\n----\n%s\n----", token, out)
		}
	}
	if strings.Contains(out, "Not updated this run") {
		t.Fatalf("fresh run should not show stale notice:\n%s", out)
	}
	if strings.Contains(out, "regressed") {
		t.Fatalf("no regressions expected:\n%s", out)
	}
}

func TestRequirementsSummaryRegressionAlert(t *testing.T) {
	r := &execTypes.RequirementsSummary{
		Synced:      true,
		OTComplete:  3,
		OTTotal:     7,
		ReqComplete: 19,
		ReqTotal:    25,
		Changes: []execTypes.RequirementChange{
			{ID: "REQ-AB-007", PRDRef: "OT-P0-001", From: "complete", To: "in_progress", Kind: execTypes.ChangeKindRegression},
			{ID: "REQ-AB-002", From: "planned", To: "complete", Kind: execTypes.ChangeKindPromotion},
		},
	}
	out := renderRequirements(t, r)

	for _, token := range []string{
		// Loud, non-zero regression signal in the header.
		"1 regressed this run",
		"requirement(s) regressed:",
		"REQ-AB-007",
		"OT-P0-001",
		"complete → in_progress",
		// Promotions still listed.
		"REQ-AB-002",
	} {
		if !strings.Contains(out, token) {
			t.Fatalf("regression branch missing %q\n----\n%s\n----", token, out)
		}
	}
}

func TestRequirementsSummaryNoChangesOnFreshRun(t *testing.T) {
	r := &execTypes.RequirementsSummary{
		Synced:      true,
		OTComplete:  7,
		OTTotal:     7,
		ReqComplete: 25,
		ReqTotal:    25,
	}
	out := renderRequirements(t, r)
	if !strings.Contains(out, "No requirement status changes this run") {
		t.Fatalf("expected no-changes line on a fresh run with no transitions:\n%s", out)
	}
}
