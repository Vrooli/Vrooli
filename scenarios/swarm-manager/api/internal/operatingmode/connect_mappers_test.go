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

func TestRoundEnvelopeToProtoCarriesResolution(t *testing.T) {
	round := RoundEnvelope{
		Round:  1,
		Status: RoundStatusNeedsAttention,
		Payload: map[string]any{
			payloadResolution: PhaseResolutionRecord{
				Outcome:         ResolutionAbstained,
				Layer:           ResolutionLayerClassifier,
				MessagesScanned: 2,
				Missing:         []string{"verdict"},
				Violations:      []string{"confidence: below minimum"},
				Notes:           []string{"classifier abstained on verdict"},
			},
		},
	}
	got := roundEnvelopeToProto(round)
	if got.GetResolution() == nil {
		t.Fatal("resolution projection is nil")
	}
	if got.GetStatus() != string(RoundStatusNeedsAttention) {
		t.Fatalf("status = %q, want needs_attention", got.GetStatus())
	}
	if got.GetResolution().GetOutcome() != string(ResolutionAbstained) || got.GetResolution().GetLayer() != string(ResolutionLayerClassifier) {
		t.Fatalf("resolution = %+v, want abstained classifier record", got.GetResolution())
	}
	if len(got.GetResolution().GetMissing()) != 1 || got.GetResolution().GetMissing()[0] != "verdict" {
		t.Fatalf("missing = %v, want verdict", got.GetResolution().GetMissing())
	}
}
