package main

import (
	"testing"
)

func TestBuildPhaseStatusPayload(t *testing.T) {
	defs := []phaseDefinition{
		{Name: "structure"},
		{Name: "unit", Optional: true},
	}
	results := []PhaseExecutionResult{
		{Name: "Structure", Status: "PASSED"},
		{Name: "unit", Status: "failed"},
	}

	payload := buildPhaseStatusPayload(defs, results)
	if len(payload) != 2 {
		t.Fatalf("expected two payload entries, got %d", len(payload))
	}
	if payload[0].Phase != "structure" || payload[0].Status != "passed" || !payload[0].Recorded {
		t.Fatalf("unexpected first payload entry: %#v", payload[0])
	}
	if payload[1].Phase != "unit" || payload[1].Status != "failed" || !payload[1].Recorded || !payload[1].Optional {
		t.Fatalf("unexpected second payload entry: %#v", payload[1])
	}
}

func TestShouldSyncRequirements(t *testing.T) {
	defs := []phaseDefinition{{Name: "structure"}, {Name: "unit"}}
	selected := append([]phaseDefinition(nil), defs...)
	phases := []PhaseExecutionResult{
		{Name: "structure", Status: "passed"},
		{Name: "unit", Status: "passed"},
	}

	req := SuiteExecutionRequest{ScenarioName: "demo"}
	if !shouldSyncRequirements(req, defs, selected, phases, true) {
		t.Fatalf("expected sync eligibility for full pass")
	}

	req.Preset = "quick"
	if shouldSyncRequirements(req, defs, selected, phases, true) {
		t.Fatalf("expected preset run to skip sync")
	}
	req.Preset = ""
	if shouldSyncRequirements(req, defs, selected[:1], phases, true) {
		t.Fatalf("expected partial phase selection to skip sync")
	}
	if shouldSyncRequirements(SuiteExecutionRequest{ScenarioName: "demo"}, defs, selected, phases[:1], true) {
		t.Fatalf("expected when not all phases recorded to skip sync")
	}
	if shouldSyncRequirements(SuiteExecutionRequest{ScenarioName: "demo"}, defs, selected, phases, false) {
		t.Fatalf("expected failed suite to skip sync")
	}
}

func TestBuildCommandHistory(t *testing.T) {
	req := SuiteExecutionRequest{
		ScenarioName: "demo",
		Phases:       []string{"structure"},
		Skip:         []string{"unit"},
		FailFast:     true,
	}
	selected := []phaseDefinition{{Name: "structure"}, {Name: "dependencies"}}

	history := buildCommandHistory(req, "quick", selected)
	if len(history) != 2 {
		t.Fatalf("expected two history entries, got %d", len(history))
	}
	if history[0] == "" || history[1] == "" {
		t.Fatalf("history entries should not be empty: %#v", history)
	}
}
