package execute

import (
	"reflect"
	"testing"
	"time"

	execTypes "test-genie/cli/internal/execute"
)

func TestPlannedPhaseNamesPreservesServerOrder(t *testing.T) {
	preview := execTypes.PlanPreview{
		Phases: []execTypes.PlanPhase{
			{Name: "structure"},
			{Name: "unit"},
			{Name: "integration"},
		},
	}

	got := plannedPhaseNames(preview)
	want := []string{"structure", "unit", "integration"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected ordered planned phases %v, got %v", want, got)
	}
}

func TestPhaseTimingTargetsBuildsEstimateAndTimeoutMaps(t *testing.T) {
	preview := execTypes.PlanPreview{
		Phases: []execTypes.PlanPhase{
			{Name: "unit", EstimatedDurationSeconds: 12, TimeoutSeconds: 90},
			{Name: "e2e", EstimatedDurationSeconds: 30, TimeoutSeconds: 120},
		},
	}

	estimates, timeouts := phaseTimingTargets(preview)
	if got := estimates["unit"]; got != 12*time.Second {
		t.Fatalf("expected unit estimate 12s, got %s", got)
	}
	if got := estimates["playbooks"]; got != 30*time.Second {
		t.Fatalf("expected e2e alias to map to playbooks, got %s", got)
	}
	if got := timeouts["playbooks"]; got != 120*time.Second {
		t.Fatalf("expected playbooks timeout 120s, got %s", got)
	}
}
