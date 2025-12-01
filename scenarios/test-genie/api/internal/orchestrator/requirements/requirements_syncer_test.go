package requirements

import (
	"testing"

	"test-genie/internal/orchestrator/phases"
)

func TestBuildPhaseStatusPayload(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Structure},
		{Name: phases.Unit, Optional: true},
	}
	results := []phases.ExecutionResult{
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
