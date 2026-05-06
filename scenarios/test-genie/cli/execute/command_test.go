package execute

import (
	"path/filepath"
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

func TestParseArgsAcceptsAbsoluteScenarioPath(t *testing.T) {
	scenarioPath := filepath.Join(t.TempDir(), "scenarios", "demo")
	parsed, err := ParseArgs([]string{"demo", "--scenario-path", scenarioPath, "--preset", "comprehensive"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if parsed.ScenarioPath != scenarioPath {
		t.Fatalf("ScenarioPath = %q, want %q", parsed.ScenarioPath, scenarioPath)
	}
	if parsed.Preset != "comprehensive" {
		t.Fatalf("Preset = %q", parsed.Preset)
	}
}

func TestParseArgsRejectsRelativeScenarioPath(t *testing.T) {
	if _, err := ParseArgs([]string{"demo", "--scenario-path", "scenarios/demo"}); err == nil {
		t.Fatal("expected relative --scenario-path to fail")
	}
}

func TestExecutionResultErrorFailsUnsuccessfulJSONResult(t *testing.T) {
	if err := executionResultError(Response{Success: true}); err != nil {
		t.Fatalf("executionResultError(success) error = %v", err)
	}
	if err := executionResultError(Response{Success: false}); err == nil {
		t.Fatal("expected unsuccessful execution result to fail")
	}
}
