package operatingmode

import (
	"encoding/json"
	"testing"
)

// TestHandoffToProto_CarriesFrontier proves the elastic-slice frontier travels
// over the Connect wire: the declared true-frontier the execute round emits is
// projected onto the OperatingModeHandoff message, so consumers (UI, CLI) can
// read where the next round should continue.
func TestHandoffToProto_CarriesFrontier(t *testing.T) {
	h := &Handoff{
		Summary:  "Landed the durable_run primitive.",
		NextStep: "classify_progress",
		Frontier: "Migrate test-genie execute onto durable run handles.",
	}
	got := handoffToProto(h)
	if got == nil {
		t.Fatal("handoffToProto returned nil")
	}
	if got.Frontier != h.Frontier {
		t.Fatalf("proto frontier = %q, want %q", got.Frontier, h.Frontier)
	}
	if got.NextStep != h.NextStep {
		t.Fatalf("proto next_step = %q, want %q", got.NextStep, h.NextStep)
	}
	if got.Summary != h.Summary {
		t.Fatalf("proto summary = %q, want %q", got.Summary, h.Summary)
	}
}

// TestHandoff_FrontierRoundTripsThroughJSON proves the frontier field parses
// from the agent's operating_mode_result envelope (the inbound direction) via
// the json tag, so a handoff an agent emits is captured, not dropped.
func TestHandoff_FrontierRoundTripsThroughJSON(t *testing.T) {
	const raw = `{"summary":"x","next_step":"review","frontier":"the exact remainder"}`
	var h Handoff
	if err := json.Unmarshal([]byte(raw), &h); err != nil {
		t.Fatalf("unmarshal handoff: %v", err)
	}
	if h.Frontier != "the exact remainder" {
		t.Fatalf("frontier = %q, want %q", h.Frontier, "the exact remainder")
	}
}
