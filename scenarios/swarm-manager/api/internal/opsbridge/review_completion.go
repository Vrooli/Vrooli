package opsbridge

import (
	"encoding/json"
	"fmt"

	"swarm-manager/internal/operatingmode"
)

// Review-verdict operation outcome names. review-round and initiative-review use
// the reviewOutcomes() vocabulary (agentops.reviewResult / reviewOutcomes),
// distinct from the handoff vocabulary the other backlog operations share. A
// review round classifies its work with a raw verdict {ready, ready_with_notes,
// needs_work, not_assessable}; the domain normalizes that onto this closed set.
const (
	ReviewOutcomeAccepted         = "accepted"
	ReviewOutcomeChangesRequested = "changes-requested"
	ReviewOutcomeFailed           = "failed"
	// ReviewOutcomeNeedsAttention aliases the shared needs-attention abstain so a
	// review round that cannot be classified honestly abstains to operator
	// attention exactly as a handoff round does.
	ReviewOutcomeNeedsAttention = OutcomeNeedsAttention
)

// ReviewRoundDelivery maps a resolved review-style operating-mode round (backlog-
// review or initiative-review) onto a CommitResult delivery. It exists because
// review operations classify with a `verdict` field the handoff delivery mapper
// does not read: HandoffRoundDelivery derives its outcome from the round's
// classifier `progress.Decision`, which review modes never emit, so routing a
// review round through it would make EVERY review round an honest-but-wrong
// needs-attention abstain (which the backlog-item policy escalates to
// needs_followup, wrongly parking the item instead of surfacing it to the
// operator review gate).
//
// The mapping is fail-honest, mirroring the recommendation-not-mutation contract:
// the delivered outcome is the agent's RECOMMENDATION, not a terminal decision.
// The completion handler records the review artifacts + recommended verdict and
// opens the operator review gate; it never terminally mutates the item. A parked,
// failed, unresolved, or unclassifiable round abstains — its artifacts are still
// forwarded so the review surface explains the round rather than showing an empty
// set.
func ReviewRoundDelivery(round operatingmode.RoundEnvelope) (Delivery, error) {
	switch round.Status {
	case operatingmode.RoundStatusCompleted:
		return reviewCompletedDelivery(round)
	case operatingmode.RoundStatusNeedsAttention, operatingmode.RoundStatusFailed:
		// A parked or failed round abstains: the operator sees it, no verdict is
		// invented. Forward the resolved output when present so the review round's
		// gathered artifacts survive for the review surface + recovery.
		return reviewAbstain(round), nil
	default:
		// reserved, agent_running, pending_evidence, canceled: not a terminal
		// outcome the runner finalizes here.
		return Delivery{Deliver: false}, nil
	}
}

// reviewCompletedDelivery derives the review outcome from the round's raw verdict
// and forwards the review handoff. A completed round whose verdict is missing,
// unclassifiable, or not_assessable abstains rather than being coerced into a
// verdict the agent did not give.
func reviewCompletedDelivery(round operatingmode.RoundEnvelope) (Delivery, error) {
	view := operatingmode.RoundPayload(round.Payload)
	resolved, ok := view.ResolvedOutput()
	if !ok {
		// Completed but no resolved envelope is a contradiction the engine should
		// not produce; abstain rather than deliver an empty result.
		return reviewAbstain(round), nil
	}
	rawVerdict, _ := resolved[reviewVerdictField].(string)
	outcome, normalized, mapped := reviewOutcomeForVerdict(rawVerdict)
	if !mapped {
		// not_assessable / missing / invalid: honest abstain, artifacts preserved.
		return reviewAbstain(round), nil
	}
	result, err := reviewResult(normalized, resolved)
	if err != nil {
		return Delivery{}, err
	}
	return Delivery{Deliver: true, Outcome: outcome, Result: result}, nil
}

// reviewAbstain builds a needs-attention abstain delivery that still carries the
// review handoff when one resolved, so a completion handler can persist the
// gathered evidence even though no honest verdict was derived. CommitResult does
// not validate an abstaining result, so the partial handoff is accepted as-is.
func reviewAbstain(round operatingmode.RoundEnvelope) Delivery {
	view := operatingmode.RoundPayload(round.Payload)
	resolved, ok := view.ResolvedOutput()
	if !ok {
		return Delivery{Deliver: true, Outcome: ReviewOutcomeNeedsAttention, Abstain: true}
	}
	// The abstain result carries the resolved handoff under `handoff` with a
	// failed verdict marker, so the handler writes a failed review round with its
	// artifacts intact. Validation is skipped for abstains, so the non-enum
	// verdict is fine here.
	result, err := reviewResult(ReviewOutcomeFailed, resolved)
	if err != nil {
		return Delivery{Deliver: true, Outcome: ReviewOutcomeNeedsAttention, Abstain: true}
	}
	return Delivery{Deliver: true, Outcome: ReviewOutcomeNeedsAttention, Abstain: true, Result: result}
}

// reviewResult assembles the reviewResult contract payload: the normalized
// verdict plus the review handoff (the mode's full validated declared output,
// carrying the raw classification, assessment, evidence, and suggestions the
// completion handler materializes into the review round).
func reviewResult(verdict string, resolved map[string]any) (json.RawMessage, error) {
	raw, err := json.Marshal(map[string]any{
		"verdict": verdict,
		"handoff": resolved,
	})
	if err != nil {
		return nil, fmt.Errorf("opsbridge: marshal review result: %w", err)
	}
	return raw, nil
}

// reviewVerdictField is the review modes' routing declared-output field carrying
// the raw classification vocabulary.
const reviewVerdictField = "verdict"

// reviewOutcomeForVerdict maps the review mode's raw classification onto the
// review operation outcome vocabulary and the normalized verdict the contract
// declares. ready / ready_with_notes accept; needs_work requests changes;
// everything else (not_assessable, missing, or any unrecognized value) is not
// mapped so the caller abstains — an honest "the agent did not give an
// actionable verdict" rather than a coerced accept/fail.
func reviewOutcomeForVerdict(rawVerdict string) (outcome, normalized string, mapped bool) {
	switch rawVerdict {
	case "ready", "ready_with_notes":
		return ReviewOutcomeAccepted, ReviewOutcomeAccepted, true
	case "needs_work":
		return ReviewOutcomeChangesRequested, ReviewOutcomeChangesRequested, true
	default:
		return "", "", false
	}
}

// isReviewOperation reports whether an operation classifies with the review
// verdict vocabulary (and so must route through ReviewRoundDelivery rather than
// HandoffRoundDelivery). evidence-request and revision use the handoff
// vocabulary and are intentionally excluded.
func isReviewOperation(op string) bool {
	return op == "review-round" || op == "initiative-review"
}

// roundDeliveryFor selects the delivery mapper for an owning operation: review-
// verdict operations use ReviewRoundDelivery, everything else the shared handoff
// mapper. It is the single branch the completion router and refresh path share.
func roundDeliveryFor(operation string, round operatingmode.RoundEnvelope) (Delivery, error) {
	if isReviewOperation(operation) {
		return ReviewRoundDelivery(round)
	}
	return HandoffRoundDelivery(round)
}
