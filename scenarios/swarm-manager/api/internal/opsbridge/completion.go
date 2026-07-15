// Package opsbridge connects the operating-mode engine's round lifecycle to the
// declarative operation runner (opsrunner). It is the one place that imports both
// engines, keeping the dependency edge one-way: the operating-mode engine never
// imports the runner, and the runner never imports the engine's round types. The
// bridge translates a resolved operating-mode round into the typed operation
// result the runner's CommitResult finalizes, and (in the wiring layer) routes a
// runner-owned round's completion into that call.
package opsbridge

import (
	"encoding/json"
	"fmt"
	"strings"

	"swarm-manager/internal/operatingmode"
)

// Handoff-style operation outcome names. Every Phase-5 backlog operation
// (research-refine, workshop-round, workshop-finalize, clarification-start,
// clarification-continue) shares this closed outcome vocabulary, derived from the
// round's classifier `progress` field (or an abstain when the round could not
// resolve one honestly).
const (
	OutcomeCompleted      = "completed"
	OutcomeContinue       = "continue"
	OutcomeBlocked        = "blocked"
	OutcomeNeedsAttention = "needs-attention"
)

// Delivery is the bridge's decision about a round: whether its completion should
// be delivered to CommitResult now, and if so, the operation outcome name and the
// contract-shaped result payload.
type Delivery struct {
	// Deliver is false when the round is not in a terminal state the runner should
	// finalize (still reserved/running, or eventually-consistent pending-evidence,
	// or a cancellation that a separate abort path owns). The caller leaves the
	// operation running and re-checks on a later refresh.
	Deliver bool
	// Outcome is the operation-contract outcome name to commit. Empty when Deliver
	// is false.
	Outcome string
	// Result is the round's validated resolved declared output (with the routing
	// progress field merged in), which the runner re-validates against the
	// operation contract's result schema. Nil for an abstaining (needs-attention)
	// outcome, which the runner accepts as a partial/absent result on purpose so
	// the round's domain artifacts survive for recovery.
	Result json.RawMessage
	// Abstain is true when the outcome is a needs-attention abstain.
	Abstain bool
}

// HandoffRoundDelivery maps a resolved handoff-style operating-mode round onto a
// CommitResult delivery. It never fabricates a routing decision: a round the
// engine parked in needs_attention/failed, or a completed round whose classifier
// progress cannot map onto the handoff outcome vocabulary, becomes an honest
// needs-attention abstain rather than a guessed continue/complete.
func HandoffRoundDelivery(round operatingmode.RoundEnvelope) (Delivery, error) {
	switch round.Status {
	case operatingmode.RoundStatusCompleted:
		return completedDelivery(round)
	case operatingmode.RoundStatusNeedsAttention, operatingmode.RoundStatusFailed:
		// A parked or failed round abstains: the operator sees it, no state auto-
		// progresses on absent/again data.
		return Delivery{Deliver: true, Outcome: OutcomeNeedsAttention, Abstain: true}, nil
	default:
		// reserved, agent_running, pending_evidence, canceled: not a terminal
		// outcome the runner finalizes here.
		return Delivery{Deliver: false}, nil
	}
}

func completedDelivery(round operatingmode.RoundEnvelope) (Delivery, error) {
	view := operatingmode.RoundPayload(round.Payload)
	progress, ok := view.Progress()
	if !ok {
		return Delivery{Deliver: true, Outcome: OutcomeNeedsAttention, Abstain: true}, nil
	}
	outcome, mapped := handoffOutcomeForProgress(progress.Decision)
	if !mapped {
		return Delivery{Deliver: true, Outcome: OutcomeNeedsAttention, Abstain: true}, nil
	}
	// The delivered result is the round's VALIDATED resolved declared output (the
	// engine's typed L1/L2 resolution), forwarded as-is with the routing `progress`
	// merged in. This is what makes the operation contract honest: whatever rich
	// result the mode declared and the engine validated — a review handoff, or the
	// enriched workshop round (decisions, lettered options, self-assessment) — is
	// exactly the result the runner records and the action handler consumes. The
	// bridge never re-derives or hand-builds the payload.
	resolved, ok := view.ResolvedOutput()
	if !ok {
		// Completed but no resolved envelope is a contradiction the engine should
		// not produce; abstain rather than deliver an empty result.
		return Delivery{Deliver: true, Outcome: OutcomeNeedsAttention, Abstain: true}, nil
	}
	result, err := mergedResult(resolved, progress.Decision)
	if err != nil {
		return Delivery{}, err
	}
	return Delivery{Deliver: true, Outcome: outcome, Result: result}, nil
}

// mergedResult forwards the resolved declared-output envelope with the routing
// progress field set, so a result whose declared output omits progress (it is a
// derived routing field, not a declared output field) still satisfies an
// operation contract that requires progress.
func mergedResult(resolved map[string]any, decision operatingmode.ProgressDecision) (json.RawMessage, error) {
	out := make(map[string]any, len(resolved)+1)
	for k, v := range resolved {
		out[k] = v
	}
	if _, ok := out["progress"]; !ok {
		out["progress"] = strings.TrimSpace(string(decision))
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("opsbridge: marshal resolved result: %w", err)
	}
	return raw, nil
}

// handoffOutcomeForProgress maps the mode classifier's progress decision onto the
// operation outcome vocabulary. A replan decision has no honest backlog outcome
// (the handoff contract declares no replan), so it abstains rather than being
// coerced into continue.
func handoffOutcomeForProgress(d operatingmode.ProgressDecision) (string, bool) {
	switch d {
	case operatingmode.ProgressComplete:
		return OutcomeCompleted, true
	case operatingmode.ProgressContinue:
		return OutcomeContinue, true
	case operatingmode.ProgressBlocked:
		return OutcomeBlocked, true
	default:
		return "", false
	}
}
