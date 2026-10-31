package runs

import (
	"testing"

	"test-genie/internal/orchestrator"
	sharedruns "test-genie/internal/shared/runs"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestTargetKindMapsEveryDescriptorSchemaKind(t *testing.T) {
	for _, kind := range []string{
		"scenario", "asset", "resource", "tool", "safeguard", "team", "package", "control-plane", "docs", "project",
	} {
		if got := targetKind(kind); got == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED {
			t.Errorf("targetKind(%q) returned UNSPECIFIED", kind)
		}
	}
}

func TestTerminalProjectionRetainsOnlyKnownObservationCount(t *testing.T) {
	count := uint32(3)
	record := sharedruns.RunRecord{RunID: "run-count", Scenario: "demo", Status: sharedruns.StatusPassed, Phases: []sharedruns.PhaseRecord{{Name: "workflow", Status: "passed"}}}
	result := &orchestrator.SuiteExecutionResult{Phases: []orchestrator.PhaseExecutionResult{{Name: "workflow", Status: "passed", ObservationCount: &count}}}
	got := toTerminalRunInfo(record, result, nil).GetPhases()[0]
	if got.ObservationCount == nil || got.GetObservationCount() != 3 {
		t.Fatalf("terminal observation count = %v, want 3", got.ObservationCount)
	}

	legacy := toTerminalRunInfo(record, &orchestrator.SuiteExecutionResult{Phases: []orchestrator.PhaseExecutionResult{{Name: "workflow", Status: "passed"}}}, nil).GetPhases()[0]
	if legacy.ObservationCount != nil {
		t.Fatalf("legacy compact phase count must remain unknown, got %d", legacy.GetObservationCount())
	}
}
