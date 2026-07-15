package opsbridge

import (
	"encoding/json"
	"testing"

	"swarm-manager/internal/operatingmode"
)

// completedReviewRound builds a completed round carrying a resolved review
// declared-output envelope (verdict + the enriched review handoff fields).
func completedReviewRound(resolved map[string]any) operatingmode.RoundEnvelope {
	round := operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusCompleted}
	view := operatingmode.MutableRoundPayload(&round)
	if resolved != nil {
		view.SetPhaseResult(operatingmode.PhaseResult{}, resolved)
	}
	return round
}

func TestReviewRoundDeliveryMapsVerdictToOutcome(t *testing.T) {
	cases := []struct {
		verdict    string
		outcome    string
		abstain    bool
		normalized string
	}{
		{"ready", ReviewOutcomeAccepted, false, "accepted"},
		{"ready_with_notes", ReviewOutcomeAccepted, false, "accepted"},
		{"needs_work", ReviewOutcomeChangesRequested, false, "changes-requested"},
		{"not_assessable", ReviewOutcomeNeedsAttention, true, ""},
		{"", ReviewOutcomeNeedsAttention, true, ""},
		{"garbage", ReviewOutcomeNeedsAttention, true, ""},
	}
	for _, tc := range cases {
		resolved := map[string]any{
			"verdict":          tc.verdict,
			"agent_assessment": "looked at it",
			"classification":   tc.verdict,
		}
		d, err := ReviewRoundDelivery(completedReviewRound(resolved))
		if err != nil {
			t.Fatalf("verdict %q: %v", tc.verdict, err)
		}
		if !d.Deliver {
			t.Fatalf("verdict %q: expected delivery", tc.verdict)
		}
		if d.Outcome != tc.outcome {
			t.Fatalf("verdict %q: want outcome %q, got %q", tc.verdict, tc.outcome, d.Outcome)
		}
		if d.Abstain != tc.abstain {
			t.Fatalf("verdict %q: want abstain=%v, got %v", tc.verdict, tc.abstain, d.Abstain)
		}
		if !tc.abstain {
			var payload struct {
				Verdict string         `json:"verdict"`
				Handoff map[string]any `json:"handoff"`
			}
			if err := json.Unmarshal(d.Result, &payload); err != nil {
				t.Fatalf("verdict %q: decode result: %v", tc.verdict, err)
			}
			if payload.Verdict != tc.normalized {
				t.Fatalf("verdict %q: want normalized %q, got %q", tc.verdict, tc.normalized, payload.Verdict)
			}
			if payload.Handoff["agent_assessment"] != "looked at it" {
				t.Fatalf("verdict %q: handoff lost: %+v", tc.verdict, payload.Handoff)
			}
		}
	}
}

// A parked/failed review round abstains but still forwards its gathered
// artifacts so the review surface can explain the round.
func TestReviewRoundDeliveryAbstainPreservesArtifacts(t *testing.T) {
	round := operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusNeedsAttention}
	view := operatingmode.MutableRoundPayload(&round)
	view.SetPhaseResult(operatingmode.PhaseResult{}, map[string]any{
		"verdict":          "not_assessable",
		"agent_assessment": "could not reach the UI",
		"evidence":         []any{map[string]any{"id": "e1", "type": "cli_output", "title": "logs"}},
	})
	d, err := ReviewRoundDelivery(round)
	if err != nil {
		t.Fatalf("ReviewRoundDelivery: %v", err)
	}
	if !d.Deliver || !d.Abstain || d.Outcome != ReviewOutcomeNeedsAttention {
		t.Fatalf("want needs-attention abstain delivery, got %+v", d)
	}
	if len(d.Result) == 0 {
		t.Fatalf("abstain must still forward the gathered artifacts")
	}
	var payload struct {
		Handoff map[string]any `json:"handoff"`
	}
	if err := json.Unmarshal(d.Result, &payload); err != nil {
		t.Fatalf("decode abstain result: %v", err)
	}
	if payload.Handoff["agent_assessment"] != "could not reach the UI" {
		t.Fatalf("abstain dropped artifacts: %+v", payload.Handoff)
	}
}

// A still-running round is not a terminal outcome the runner finalizes.
func TestReviewRoundDeliveryNonTerminalDoesNotDeliver(t *testing.T) {
	d, err := ReviewRoundDelivery(operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusAgentRunning})
	if err != nil {
		t.Fatalf("ReviewRoundDelivery: %v", err)
	}
	if d.Deliver {
		t.Fatalf("a running round must not deliver")
	}
}

func TestRoundDeliveryForSelectsReviewMapper(t *testing.T) {
	resolved := map[string]any{"verdict": "ready", "agent_assessment": "ok"}
	// review-round routes through the verdict mapper.
	d, err := roundDeliveryFor("review-round", completedReviewRound(resolved))
	if err != nil || d.Outcome != ReviewOutcomeAccepted {
		t.Fatalf("review-round should map verdict->accepted, got %+v err=%v", d, err)
	}
	// evidence-request stays on the handoff mapper (verdict is ignored there).
	hd, err := HandoffRoundDelivery(completedRound(operatingmode.ProgressComplete, map[string]any{"handoff": map[string]any{"summary": "x"}}))
	if err != nil {
		t.Fatalf("handoff mapper: %v", err)
	}
	ed, err := roundDeliveryFor("evidence-request", completedRound(operatingmode.ProgressComplete, map[string]any{"handoff": map[string]any{"summary": "x"}}))
	if err != nil {
		t.Fatalf("evidence-request delivery: %v", err)
	}
	if ed.Outcome != hd.Outcome || ed.Outcome != OutcomeCompleted {
		t.Fatalf("evidence-request should use handoff mapping (completed), got %+v", ed)
	}
}
