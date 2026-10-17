package handlers

import (
	"context"
	"net/http"

	"agent-manager/internal/supervision"

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
// webhook's ignored response).
func DecisionResolvedWebhook(service *supervision.Service, resolver RunApprovalResolver, secret string) http.Handler {
	return operatordecision.Webhook(secret,
		runApprovalEffect{resolver: resolver},
		authorizationEffect{resolver: resolver},
		effortDecisionEffect{service: service},
	)
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
