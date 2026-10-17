package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
)

// DecisionEventWebhookPath is the notification-hub ask.resolved webhook route
// autoheal mounts to receive operator answers to its remediation decisions.
const DecisionEventWebhookPath = "/api/v1/events/notification-hub"

// DecisionResolvedWebhook receives notification-hub's
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook and
// applies it via the shared operatordecision.Webhook. The substrate verifies the
// HMAC signature, decodes the envelope, gates on the ask-resolved event type,
// and fans the resolution to the remediation effect. This closes the loop the
// manual /approve REST route opened: an approve on a LOW-RISK candidate
// auto-executes server-side; a high-risk approve records the approval and leaves
// the explicit manual apply step to the operator; a reject/no-answer records a
// declined outcome and executes nothing.
func DecisionResolvedWebhook(h *Handlers, secret string) http.Handler {
	return operatordecision.Webhook(secret, remediationDecisionEffect{handlers: h})
}

// remediationDecisionEffect reconstructs the apply call from a resolved
// remediation decision. It owns ONLY autoheal's remediation decision event type,
// so a resolution for any other producer falls through (handled=false) and the
// webhook returns its ignored response.
type remediationDecisionEffect struct{ handlers *Handlers }

func (remediationDecisionEffect) Handles(sourceEventType string) bool {
	return sourceEventType == incidents.RemediationDecisionEventType
}

func (e remediationDecisionEffect) Apply(ctx context.Context, res operatordecision.DecisionResolution) (bool, error) {
	if e.handlers == nil {
		return false, nil
	}
	// The correlation map is the only trustworthy carrier of which incident and
	// candidate the operator answered about; the webhook never trusts a
	// caller-asserted approval, and the service method re-verifies the ask and
	// re-matches the fingerprint against the live incident.
	incidentID := res.Correlation["incident_id"]
	candidateID := res.Correlation["candidate_id"]
	fingerprint := res.Correlation["incident_fingerprint"]
	if incidentID == "" || candidateID == "" {
		// A resolution without the keys we need is not ours to act on; ack it as a
		// benign no-op so vrooli-events does not retry a poison message.
		return true, nil
	}

	// Only an explicit approve crosses into the apply path. A reject, a default,
	// an expiry, or any non-approve answer records a declined outcome and
	// executes nothing.
	if res.State != operatordecision.StateAnswered || !remediation.ApprovedAsk(remediation.AskApproval{Answer: res.Answer}) {
		reason := "operator declined remediation (" + res.State + "/" + res.Answer + ")"
		if err := e.handlers.recordDeclinedRemediation(ctx, incidentID, candidateID, res.AskID, fingerprint, reason); err != nil {
			return true, err
		}
		return true, nil
	}

	outcome, err := e.handlers.ApproveRemediationFromDecision(ctx, remediationApprovalInput{
		IncidentID:          incidentID,
		CandidateID:         candidateID,
		AskID:               res.AskID,
		IncidentFingerprint: fingerprint,
		// The event consumer is the auto-apply path: it risk-gates so only a
		// low-risk candidate executes on approve.
		RiskGated: true,
	})
	if err != nil {
		// An already-consumed authorisation (a replayed/again-claimed resolution)
		// is a benign duplicate: ack it rather than letting vrooli-events retry a
		// poison message forever.
		if errors.Is(err, ErrRemediationAlreadyApplied) {
			return true, nil
		}
		return true, err
	}
	_ = outcome
	return true, nil
}
