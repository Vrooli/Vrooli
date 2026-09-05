package requirements

import (
	"testing"
	"time"

	reqsvc "test-genie/internal/requirements"
	"test-genie/internal/requirements/enrichment"
	syncpkg "test-genie/internal/requirements/sync"
	reqtypes "test-genie/internal/requirements/types"
)

func TestClassifyChange(t *testing.T) {
	cases := []struct {
		from, to, want string
	}{
		{"planned", "complete", ChangeKindPromotion},
		{"in_progress", "complete", ChangeKindPromotion},
		{"complete", "in_progress", ChangeKindRegression},
		{"complete", "failing", ChangeKindRegression},
		{"planned", "in_progress", ChangeKindOther},
		{"pending", "planned", ChangeKindOther},
	}
	for _, c := range cases {
		if got := classifyChange(c.from, c.to); got != c.want {
			t.Errorf("classifyChange(%q,%q) = %q, want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestNewOutcomeFromReportNil(t *testing.T) {
	if out := newOutcomeFromReport(nil, "reason"); out != nil {
		t.Fatalf("expected nil outcome for nil report, got %+v", out)
	}
}

func TestNewOutcomeFromReportFlattensAndClassifies(t *testing.T) {
	ts := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	report := &reqsvc.SyncReport{
		Synced:       true,
		LastSyncedAt: &ts,
		Summary: enrichment.Summary{
			Total: 25,
			ByDeclaredStatus: map[reqtypes.DeclaredStatus]int{
				reqtypes.StatusComplete:   20,
				reqtypes.StatusInProgress: 3,
				reqtypes.StatusPlanned:    2,
			},
		},
		OT: syncpkg.OTSummary{
			Complete: 4,
			Total:    7,
			ByPriority: map[string]syncpkg.OTPriorityCount{
				"P0": {Complete: 2, Total: 2},
				"P1": {Complete: 2, Total: 5},
			},
		},
		Changes: []reqsvc.StatusChange{
			{ID: "REQ-001", PRDRef: "OT-P1-002", From: "in_progress", To: "complete"},
			{ID: "REQ-007", PRDRef: "OT-P0-001", From: "complete", To: "in_progress"},
		},
	}

	out := newOutcomeFromReport(report, "")
	if out == nil {
		t.Fatal("expected non-nil outcome")
	}
	if !out.Synced {
		t.Error("expected Synced=true")
	}
	if out.ReqComplete != 20 || out.ReqTotal != 25 {
		t.Errorf("req counts = %d/%d, want 20/25", out.ReqComplete, out.ReqTotal)
	}
	if out.ReqByStatus["in_progress"] != 3 {
		t.Errorf("ReqByStatus[in_progress] = %d, want 3", out.ReqByStatus["in_progress"])
	}
	if out.OTComplete != 4 || out.OTTotal != 7 {
		t.Errorf("OT counts = %d/%d, want 4/7", out.OTComplete, out.OTTotal)
	}
	if out.OTByPriority["P1"].Total != 5 {
		t.Errorf("OTByPriority[P1].Total = %d, want 5", out.OTByPriority["P1"].Total)
	}
	if out.RegressionCount() != 1 {
		t.Errorf("RegressionCount() = %d, want 1", out.RegressionCount())
	}
	// Verify each change was classified.
	kinds := map[string]string{}
	for _, c := range out.Changes {
		kinds[c.ID] = c.Kind
	}
	if kinds["REQ-001"] != ChangeKindPromotion {
		t.Errorf("REQ-001 kind = %q, want promotion", kinds["REQ-001"])
	}
	if kinds["REQ-007"] != ChangeKindRegression {
		t.Errorf("REQ-007 kind = %q, want regression", kinds["REQ-007"])
	}
}

func TestNewOutcomeFromReportSkipReasonOnCached(t *testing.T) {
	report := &reqsvc.SyncReport{
		Synced:  false,
		Summary: enrichment.Summary{Total: 3},
	}
	out := newOutcomeFromReport(report, "required phases skipped: unit")
	if out.Synced {
		t.Error("expected Synced=false")
	}
	if out.SkipReason != "required phases skipped: unit" {
		t.Errorf("SkipReason = %q", out.SkipReason)
	}
}
