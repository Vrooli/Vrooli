package sync

import (
	"testing"

	"test-genie/internal/requirements/parsing"
	"test-genie/internal/requirements/types"
)

func TestOperationalTargetSummary(t *testing.T) {
	index := parsing.NewModuleIndex()
	module := &types.RequirementModule{
		FilePath: "/test/requirements/module.json",
		Requirements: []types.Requirement{
			// OT-P0-001: both requirements complete -> OT complete.
			{ID: "REQ-001", PRDRef: "OT-P0-001", Status: types.StatusComplete},
			{ID: "REQ-002", PRDRef: "OT-P0-001", Status: types.StatusComplete},
			// OT-P1-002: one incomplete -> OT not complete.
			{ID: "REQ-003", PRDRef: "OT-P1-002", Status: types.StatusComplete},
			{ID: "REQ-004", PRDRef: "OT-P1-002", Status: types.StatusInProgress},
			// OT-P2-003: incomplete.
			{ID: "REQ-005", PRDRef: "OT-P2-003", Status: types.StatusPending},
			// No PRDRef -> not counted as an OT.
			{ID: "REQ-006", PRDRef: "", Status: types.StatusComplete},
		},
	}
	mustAddModule(t, index, module)

	summary := OperationalTargetSummary(index)

	if summary.Total != 3 {
		t.Fatalf("expected 3 operational targets, got %d", summary.Total)
	}
	if summary.Complete != 1 {
		t.Fatalf("expected 1 complete operational target, got %d", summary.Complete)
	}
	if got := summary.ByPriority["P0"]; got.Complete != 1 || got.Total != 1 {
		t.Fatalf("P0 band = %+v, want {Complete:1 Total:1}", got)
	}
	if got := summary.ByPriority["P1"]; got.Complete != 0 || got.Total != 1 {
		t.Fatalf("P1 band = %+v, want {Complete:0 Total:1}", got)
	}
	if got := summary.ByPriority["P2"]; got.Complete != 0 || got.Total != 1 {
		t.Fatalf("P2 band = %+v, want {Complete:0 Total:1}", got)
	}
}

func TestOperationalTargetSummaryEmpty(t *testing.T) {
	summary := OperationalTargetSummary(parsing.NewModuleIndex())
	if summary.Total != 0 || summary.Complete != 0 {
		t.Fatalf("expected zeroed summary, got %+v", summary)
	}
}
