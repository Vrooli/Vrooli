package opsbridge

import (
	"encoding/json"
	"testing"

	"swarm-manager/internal/operatingmode"
)

// completedRound builds a completed round carrying a resolved declared-output
// envelope (what the resolution ladder validated) plus the routing progress
// decision — the two things the completion bridge reads.
func completedRound(decision operatingmode.ProgressDecision, resolved map[string]any) operatingmode.RoundEnvelope {
	round := operatingmode.RoundEnvelope{Status: operatingmode.RoundStatusCompleted}
	view := operatingmode.MutableRoundPayload(&round)
	if resolved != nil {
		view.SetPhaseResult(operatingmode.PhaseResult{}, resolved)
	}
	if decision != "" {
		view.SetProgress(operatingmode.ProgressState{Decision: decision})
	}
	return round
}

func TestHandoffRoundDeliveryMapsProgressToOutcome(t *testing.T) {
	cases := []struct {
		decision operatingmode.ProgressDecision
		outcome  string
		abstain  bool
	}{
		{operatingmode.ProgressComplete, OutcomeCompleted, false},
		{operatingmode.ProgressContinue, OutcomeContinue, false},
		{operatingmode.ProgressBlocked, OutcomeBlocked, false},
		{operatingmode.ProgressReplan, OutcomeNeedsAttention, true}, // no honest backlog outcome for replan
	}
	for _, tc := range cases {
		d, err := HandoffRoundDelivery(completedRound(tc.decision, map[string]any{"handoff": map[string]any{"summary": "x"}}))
		if err != nil {
			t.Fatalf("decision %q: %v", tc.decision, err)
		}
		if !d.Deliver {
			t.Fatalf("decision %q: expected delivery", tc.decision)
		}
		if d.Outcome != tc.outcome {
			t.Fatalf("decision %q: want outcome %q, got %q", tc.decision, tc.outcome, d.Outcome)
		}
		if d.Abstain != tc.abstain {
			t.Fatalf("decision %q: want abstain=%v, got %v", tc.decision, tc.abstain, d.Abstain)
		}
	}
}

func TestHandoffRoundDeliveryForwardsResolvedOutputAndMergesProgress(t *testing.T) {
	// The bridge forwards whatever rich result the engine validated — here an
	// enriched workshop round (handoff + decisions + self-assessment) — verbatim,
	// with progress merged in. It never re-derives or drops fields.
	resolved := map[string]any{
		"handoff": map[string]any{"summary": "synthesized decisions", "next_step": "finalize the plan"},
		"decisions": []any{
			map[string]any{"id": "d1", "topic": "storage", "options": []any{map[string]any{"key": "A", "label": "shard"}}},
		},
		"self_assessment": map[string]any{"problem_clarity": 3, "scope_defined": 2},
	}
	d, err := HandoffRoundDelivery(completedRound(operatingmode.ProgressComplete, resolved))
	if err != nil {
		t.Fatalf("HandoffRoundDelivery: %v", err)
	}
	var payload struct {
		Handoff        map[string]any `json:"handoff"`
		Decisions      []any          `json:"decisions"`
		SelfAssessment map[string]any `json:"self_assessment"`
		Progress       string         `json:"progress"`
	}
	if err := json.Unmarshal(d.Result, &payload); err != nil {
		t.Fatalf("decode result: %v (%s)", err, d.Result)
	}
	if payload.Progress != "complete" {
		t.Fatalf("want progress complete, got %q", payload.Progress)
	}
	if payload.Handoff["summary"] != "synthesized decisions" {
		t.Fatalf("handoff lost: %+v", payload.Handoff)
	}
	if len(payload.Decisions) != 1 {
		t.Fatalf("rich decisions dropped: %+v", payload.Decisions)
	}
	if payload.SelfAssessment["problem_clarity"] != float64(3) {
		t.Fatalf("self_assessment dropped: %+v", payload.SelfAssessment)
	}
}

func TestHandoffRoundDeliveryDoesNotOverwriteDeclaredProgress(t *testing.T) {
	// If the declared output already carries progress, the bridge keeps it rather
	// than overwriting from the routing decision.
	resolved := map[string]any{"handoff": map[string]any{"summary": "x"}, "progress": "complete"}
	d, err := HandoffRoundDelivery(completedRound(operatingmode.ProgressComplete, resolved))
	if err != nil {
		t.Fatalf("HandoffRoundDelivery: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(d.Result, &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["progress"] != "complete" {
		t.Fatalf("declared progress not preserved: %v", payload["progress"])
	}
}

// [REQ:REQ-P0-011-OUTCOME-CLASSIFIERS]
func TestHandoffRoundDeliveryCompletedWithoutProgressAbstains(t *testing.T) {
	d, err := HandoffRoundDelivery(completedRound("", map[string]any{"handoff": map[string]any{}}))
	if err != nil {
		t.Fatalf("HandoffRoundDelivery: %v", err)
	}
	if !d.Deliver || !d.Abstain || d.Outcome != OutcomeNeedsAttention {
		t.Fatalf("completed round without progress must abstain to needs-attention, got %+v", d)
	}
	if d.Result != nil {
		t.Fatalf("abstain must carry no result, got %s", d.Result)
	}
}

func TestHandoffRoundDeliveryTerminalNonCompletedAbstains(t *testing.T) {
	for _, status := range []operatingmode.RoundStatus{operatingmode.RoundStatusNeedsAttention, operatingmode.RoundStatusFailed} {
		d, err := HandoffRoundDelivery(operatingmode.RoundEnvelope{Status: status})
		if err != nil {
			t.Fatalf("status %q: %v", status, err)
		}
		if !d.Deliver || !d.Abstain || d.Outcome != OutcomeNeedsAttention {
			t.Fatalf("status %q must abstain-deliver, got %+v", status, d)
		}
	}
}

func TestHandoffRoundDeliveryNonTerminalDoesNotDeliver(t *testing.T) {
	for _, status := range []operatingmode.RoundStatus{
		operatingmode.RoundStatusReserved,
		operatingmode.RoundStatusAgentRunning,
		operatingmode.RoundStatusPendingEvidence,
		operatingmode.RoundStatusCanceled,
	} {
		d, err := HandoffRoundDelivery(operatingmode.RoundEnvelope{Status: status})
		if err != nil {
			t.Fatalf("status %q: %v", status, err)
		}
		if d.Deliver {
			t.Fatalf("status %q must not deliver to CommitResult, got %+v", status, d)
		}
	}
}
