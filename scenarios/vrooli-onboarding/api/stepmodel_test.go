package main

import (
	"testing"

	"github.com/vrooli/vrooli/internal/operatorstate"
)

func TestOnboardingStepModelIsOrderedAndEmptyStateIsUnsatisfied(t *testing.T) {
	if len(onboardingSteps) != 9 {
		t.Fatalf("step count = %d, want 9", len(onboardingSteps))
	}
	empty := OperatorState{}
	for index, step := range onboardingSteps {
		if step.Ordinal != index || step.ID == "" || step.Route == "" {
			t.Fatalf("invalid step %d: %#v", index, step)
		}
		if step.Satisfied(empty) {
			t.Errorf("empty state satisfies step %s", step.ID)
		}
	}
	if got := firstUnsatisfiedStep(empty); got != 0 {
		t.Fatalf("first empty step = %d, want 0", got)
	}
	if got := firstUnsatisfiedStep(OperatorState{Version: "1", Session: &operatorstate.Session{Step: 1}, Scenarios: map[string]ScenarioChoice{"demo": {Enabled: boolPtr(true)}}, Resources: map[string]EnabledChoice{}, HostTools: map[string]OptInChoice{}}); got != 7 {
		t.Fatalf("first configured step = %d, want apply step 7", got)
	}
}

func boolPtr(value bool) *bool { return &value }
