package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// DecisionEventWebhookPath is the notification-hub ask.resolved webhook route.
const DecisionEventWebhookPath = "/api/v1/events/notification-hub"

// RunApprovalResolver applies a notification-hub ask resolution that answers a
// run-approval decision (needs_review), or an "ask, don't block" authorization
// decision (Phase 4). Each resolver reports handled=true when the resolution
// belongs to it, so the webhook stops routing; false falls through to the next.
// Satisfied by *orchestration.Orchestrator.
type RunApprovalResolver interface {
	ResolveRunApprovalFromHub(ctx context.Context, resolution supervision.DecisionResolution) (bool, error)
	ResolveAuthorizationFromHub(ctx context.Context, resolution supervision.DecisionResolution) (bool, error)
}

// DecisionResolvedWebhook receives notification-hub's
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook and
// applies it via the shared operatordecision.Webhook. The substrate verifies the
// HMAC signature, decodes the envelope, gates on the ask-resolved event type, and
// fans the resolution to three DomainEffects in the SAME order as before:
// run-approval → authorization → effort. The first effect that reports handled
// wins; an effect that does not own the resolution reports handled=false and the
// next one tries. Events about asks Agent Manager did not raise, and duplicate
// deliveries, are acknowledged without change (run-approval and authorization
// return a benign handled=true no-op for an unknown/already-decided ask of their
// type; a resolution that matches no effort decision falls through to the
// webhook's ignored response). extra effects (each claiming only its own
// source event type, such as the evidence-prune approval) run after the
// requester tie-back and before the run/authorization/effort appliers.
func DecisionResolvedWebhook(service *supervision.Service, resolver RunApprovalResolver, secret string, extra ...operatordecision.DomainEffect) http.Handler {
	effects := make([]operatordecision.DomainEffect, 0, 4+len(extra))
	// The requester tie-back is a pure observer that runs FIRST and never claims
	// the resolution (handled=false), so it fires for EVERY resolution carrying
	// an agent-await key regardless of which later effect applies the decision —
	// and never blocks run-approval / authorization / effort application.
	if waker, ok := resolver.(requesterWaker); ok {
		effects = append(effects, requesterWakeEffect{waker: waker})
	}
	effects = append(effects, extra...)
	effects = append(effects,
		runApprovalEffect{resolver: resolver},
		authorizationEffect{resolver: resolver},
		effortDecisionEffect{service: service},
	)
	return operatordecision.Webhook(secret, effects...)
}

// requesterWaker is the narrow orchestrator surface the requester tie-back uses:
// read the run that raised a decision and deliver the answer to it by its
// current state — wake it if parked, continue it if running/terminal. The
// concrete *orchestration.Orchestrator (passed as the webhook's resolver)
// satisfies it; a test/fake resolver that does not simply disables the effect.
type requesterWaker interface {
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	WakeRun(ctx context.Context, in orchestration.WakeRunInput) (*domain.Run, error)
	ContinueRun(ctx context.Context, req orchestration.ContinueRunRequest) (*domain.Run, error)
	// SendInteractivePrompt is the stable-session-id delivery lane: deliver to a
	// web-console session directly, without an agent-manager run (used when the
	// run id is absent/unusable — e.g. a long session whose token expired).
	SendInteractivePrompt(ctx context.Context, sessionID, text, source string) error
}

// requesterWakeEffect delivers an operator's answer back to the agent run that
// raised the decision, so the agent resumes automatically instead of polling or
// waiting to be told. It is the generalization of the effort-decision wake to
// ANY agent-raised ask: a producer opts in by stamping
// operatordecision.CorrelationRaisedByRunID (the agent's run id, obtained from
// the shared agent-provenance layer). Parking is OPTIONAL — delivery adapts to
// the run's state:
//   - parked on this operator decision → WakeRun (resumes the suspended run)
//   - running interactive / terminal / idle → ContinueRun (mid-turn paste into
//     the live session, or a continuation turn) — the same path an operator uses
//     chatting to a live agent
//   - running non-interactive mid-turn → cannot be interrupted (platform limit);
//     logged, left for the agent to re-check (such an agent should park)
//
// It is best-effort and NON-owning: it never returns handled=true (so it cannot
// block a later effect that actually applies the decision) and never returns an
// error (so a delivery failure cannot trigger a vrooli-events retry storm — the
// durable effect is the producer projecting its own grant, not this notice).
// Delivery runs off the webhook goroutine on a detached, bounded context because
// WakeRun/ContinueRun can block on a turn drain. Redelivery is deduped: WakeRun
// is idempotent, ContinueRun carries a stable IdempotencyKey, and a run already
// woken for this decision is not then also continued.
type requesterWakeEffect struct{ waker requesterWaker }

func (requesterWakeEffect) Handles(string) bool { return true }

func (e requesterWakeEffect) Apply(_ context.Context, res operatordecision.DecisionResolution) (bool, error) {
	runIDStr := strings.TrimSpace(res.Correlation[operatordecision.CorrelationRaisedByRunID])
	wcSession := strings.TrimSpace(res.Correlation[operatordecision.CorrelationWebConsoleSessionID])
	if e.waker == nil || (runIDStr == "" && wcSession == "") {
		// Not a tie-back resolution (or no waker wired): pass through untouched so
		// the real appliers decide ownership.
		return false, nil
	}
	go e.deliverToRequester(runIDStr, wcSession,
		strings.TrimSpace(res.Correlation[operatordecision.CorrelationAgentAwaitKey]),
		res)
	return false, nil
}

// deliverToRequester delivers the resolved answer back to the requesting agent.
// It prefers the run-id lane (wake if parked, continue if running/terminal) and
// falls back to the stable web-console-session-id lane when there is no usable
// run (e.g. a long interactive session whose identity token expired, so no run
// id was captured). Runs detached from the webhook request.
func (e requesterWakeEffect) deliverToRequester(runIDStr, wcSession, awaitKey string, res operatordecision.DecisionResolution) {
	defer obs.RecoverToFailure("operator-decision requester delivery", nil)
	log := obs.Component("operator-decision-wake")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	message := "[operator decision resolved] " + formatDecisionWakeMessage(res)

	// Lane 1: by run id. A resolvable run is delivered to by its current state;
	// an unresolvable one falls through to the stable session-id lane.
	if runIDStr != "" {
		if runID, err := uuid.Parse(runIDStr); err != nil {
			log.Warn("tie-back run id is not a valid uuid; trying session lane", "raised_by_run_id", runIDStr, obs.KeyError, err.Error())
		} else if run, gErr := e.waker.GetRun(ctx, runID); gErr != nil {
			log.Info("tie-back run not found; trying session lane", obs.KeyRunID, runID.String(), obs.KeyError, gErr.Error())
		} else {
			e.deliverByRunState(ctx, run, awaitKey, message, res)
			return
		}
	}

	// Lane 2: by stable web-console session id — deliver straight to the live
	// session via agent-manager's web-console client, no run required.
	if wcSession != "" {
		if err := e.waker.SendInteractivePrompt(ctx, wcSession, message, "agent-manager:operator-decision"); err != nil {
			log.Warn("tie-back session delivery failed", "web_console_session_id", wcSession, obs.KeyError, err.Error())
		}
		return
	}
	log.Info("tie-back resolution had no deliverable target")
}

// deliverByRunState delivers to a known run adapting to its current state.
func (e requesterWakeEffect) deliverByRunState(ctx context.Context, run *domain.Run, awaitKey, message string, res operatordecision.DecisionResolution) {
	log := obs.Component("operator-decision-wake")
	runID := run.ID
	if run.Status == domain.RunStatusParked {
		handle := run.AwaitHandle
		if handle == nil || handle.Producer != orchestration.ProducerOperatorDecision || (awaitKey != "" && strings.TrimSpace(handle.Key) != awaitKey) {
			// Parked on unrelated work — waking it would resolve the wrong await.
			log.Info("tie-back run is parked on other work; not delivering", obs.KeyRunID, runID.String(), "await_key", awaitKey)
			return
		}
		if _, err := e.waker.WakeRun(ctx, orchestration.WakeRunInput{RunID: runID, Result: message}); err != nil {
			log.Warn("tie-back wake failed", obs.KeyRunID, runID.String(), obs.KeyError, err.Error())
		}
		return
	}

	// Dedupe a redelivery that arrives after the run was already woken for this
	// exact decision: WakeRun records LastAwaitKey, so a running run carrying it
	// has already been told — continuing would double-deliver.
	if awaitKey != "" && run.LastAwaitKey == orchestration.ProducerOperatorDecision+":"+awaitKey {
		log.Info("tie-back run already woken for this decision; not re-delivering", obs.KeyRunID, runID.String(), "await_key", awaitKey)
		return
	}

	// Running interactive, terminal, or idle: deliver via ContinueRun — a
	// mid-turn paste into a live interactive session, or a continuation turn.
	// A stable idempotency key dedupes a redelivered resolution.
	idemKey := "operator-decision:" + awaitKey + ":" + res.State + ":" + res.Answer
	if _, err := e.waker.ContinueRun(ctx, orchestration.ContinueRunRequest{RunID: runID, Message: message, IdempotencyKey: idemKey}); err != nil {
		// A running NON-interactive run refuses mid-turn text (platform limit);
		// not an error worth retrying — the agent should have parked.
		log.Info("tie-back continue did not deliver (agent may be mid-turn and non-interactive)", obs.KeyRunID, runID.String(), "status", string(run.Status), obs.KeyError, err.Error())
	}
}

// formatDecisionWakeMessage composes the answer-agnostic summary injected into
// the woken requester's turn. It states what the operator answered and leaves
// the interpretation to the agent's own request semantics (this effect does not
// know which option key means "granted" for an arbitrary producer).
func formatDecisionWakeMessage(res operatordecision.DecisionResolution) string {
	answeredBy := strings.TrimSpace(res.AnsweredBy)
	if answeredBy == "" {
		answeredBy = "the operator"
	}
	label := strings.TrimSpace(res.AnswerLabel)
	if label == "" {
		label = strings.TrimSpace(res.Answer)
	}
	note := strings.TrimSpace(res.Note)
	noteLine := ""
	if note != "" {
		noteLine = fmt.Sprintf(" Operator note: %s.", note)
	}
	switch res.State {
	case "expired":
		reason := strings.TrimSpace(res.Reason)
		if reason != "" {
			reason = " (" + reason + ")"
		}
		return fmt.Sprintf("Your pending operator decision EXPIRED without an answer%s. The authority you requested was NOT granted. Decide whether to re-raise the request or proceed without it; do not assume approval.", reason)
	case "defaulted":
		return fmt.Sprintf("Your pending operator decision was resolved by its default answer: %q (%s).%s Check whether this grants the authority you requested before acting; do not re-raise if it is already resolved.", label, strings.TrimSpace(res.Answer), noteLine)
	default: // "answered" and any concrete operator answer
		return fmt.Sprintf("Your pending operator decision was answered by %s: %q (%s).%s Re-check the resource you requested — if your requested authority was granted it is now in effect. Proceed accordingly and do NOT re-raise the request.", answeredBy, label, strings.TrimSpace(res.Answer), noteLine)
	}
}

// runApprovalEffect wraps ResolveRunApprovalFromHub (needs_review → approve/reject)
// verbatim. Handles is permissive: the resolver itself gates on the source event
// type and reports handled=false for a foreign resolution, so the webhook falls
// through to the next effect — preserving the pre-substrate fall-through contract.
type runApprovalEffect struct{ resolver RunApprovalResolver }

func (runApprovalEffect) Handles(string) bool { return true }

func (e runApprovalEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if e.resolver == nil {
		return false, nil
	}
	return e.resolver.ResolveRunApprovalFromHub(ctx, res)
}

// authorizationEffect wraps ResolveAuthorizationFromHub ("ask, don't block":
// approve issues a bounded grant, reject records the refusal, a stale/unknown ask
// is a benign no-op) verbatim. Same permissive-Handles + resolver-gates contract.
type authorizationEffect struct{ resolver RunApprovalResolver }

func (authorizationEffect) Handles(string) bool { return true }

func (e authorizationEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if e.resolver == nil {
		return false, nil
	}
	return e.resolver.ResolveAuthorizationFromHub(ctx, res)
}

// effortDecisionEffect wraps the effort answer-sync resolver (ResolveDecisionFromHub
// → records the answer, signals the effort, wakes a parked orchestrator) verbatim.
// It is last in the fan-out and handles both the effort event type and the legacy
// empty-source-event-type resolution. A resolution that matches no effort decision
// reports handled=false so the webhook returns its ignored response (the
// pre-substrate "no matching effort decision" acknowledgement).
type effortDecisionEffect struct{ service *supervision.Service }

func (effortDecisionEffect) Handles(string) bool { return true }

func (e effortDecisionEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if e.service == nil || e.service.Efforts == nil {
		return false, nil
	}
	decision, err := e.service.Efforts.ResolveDecisionFromHub(ctx, res)
	if err != nil {
		return true, err
	}
	// A nil decision means no effort decision matched (ignored); a non-nil one
	// means the answer was applied (handled).
	return decision != nil, nil
}
