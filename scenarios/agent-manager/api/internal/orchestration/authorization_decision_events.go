// This file implements the Phase 4 "ask, don't block" path. When a verified
// known caller (a service principal) lacks a covering standing grant for a
// costly operation (run.create) and no live human is present, agent-manager
// does NOT hard-refuse: it raises an operator decision via notification-hub
// (the same generic *.decision_requested.v1 contract runs/efforts use) and
// returns a typed "approval pending" to the caller. The operator's answer
// returns as notification_hub.ask.resolved.v1 and, on approval, issues a
// bounded grant into the Phase-2 standing-grant store attributed to the
// approving human — so the caller's next attempt (a heartbeat's next tick, or a
// woken ad-hoc run) finds the grant and proceeds. Reject issues nothing; a
// stale/unknown resolution is a benign no-op.
package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/supervision"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/eventbus"
)

// AuthorizationDecisionEventType is the decision event raised when a verified
// caller needs operator approval for a costly operation. The
// `.decision_requested.v1` suffix is the generic contract notification-hub
// already subscribes to, so no hub change is needed.
const AuthorizationDecisionEventType = "agent_manager.authorization.decision_requested.v1"

// InteractiveAuthorizationDecisionEventType is the decision event raised when an
// in-session (interactive) authorization REQUEST is prepared by an agent and is
// awaiting the operator's in-UI approval. Closing gap G3: the interactive path
// (RequestAuthorization) previously persisted a pending request but never
// notified the operator, so a request could sit indefinitely until the operator
// happened to open the UI.
//
// It is DELIBERATELY a distinct event type from
// AuthorizationDecisionEventType. The interactive grant is approved ONLY through
// ApproveAuthorization, which requires a live, independently-verified human AND
// a matching live process binding — properties a notification-hub resolution
// cannot carry. So this event is a PUSH-ONLY notification: it brings the
// operator to the review UI; it must never be wired to a hub resolver that would
// mint the grant, which would bypass the independent-human + process-binding
// gates. No hub resolver subscribes to this type (ResolveAuthorizationFromHub
// matches only AuthorizationDecisionEventType), so a resolution for it falls
// through as a benign no-op.
const InteractiveAuthorizationDecisionEventType = "agent_manager.interactive_authorization.decision_requested.v1"

// MaxOutstandingAuthorizationRequestsPerSubject caps how many distinct
// un-decided asks a single subject may have outstanding at once. It bounds the
// operator's notification exposure to a buggy or hostile caller: beyond the cap
// a new distinct ask is refused WITHOUT raising a notification. A repeated ask
// for the same (subject, operation, target) is idempotent and does not count
// again (it reuses the existing pending row).
const MaxOutstandingAuthorizationRequestsPerSubject = 3

// pendingAuthorizationWindow bounds how long a raised ask stays outstanding
// before it ages out of the anti-flood count and may be re-raised. It is long
// enough for an operator to answer an out-of-band notification, short enough
// that an unanswered ask does not permanently consume a subject's cap slot.
const pendingAuthorizationWindow = 24 * time.Hour

// standingGrantIssueTTL is the lifetime of a standing grant minted by an
// approval for a recurring, subject-keyed caller — the Phase-2 bound.
const standingGrantIssueTTL = MaxStandingGrantLifetime

// oneTimeGrantIssueTTL is the (short) lifetime of a grant minted by an approval
// for an ad-hoc, session-scoped caller.
const oneTimeGrantIssueTTL = time.Hour

// WithPendingAuthorizations supplies the durable pending-request store that
// makes the "ask, don't block" raise idempotent and bounded. Without it, the
// authority path falls back to a hard refuse (no regression from pre-Phase-4).
func WithPendingAuthorizations(repo domain.PendingAuthorizationRepository) Option {
	return func(o *Orchestrator) { o.pendingAuthorizations = repo }
}

// AuthorizationApprovalRequest describes a would-be grant an operator is asked
// to approve. The handler builds it from the refused operation; the raiser
// turns it into a durable pending record + a notification.
type AuthorizationApprovalRequest struct {
	Subject           string
	Operation         string
	Target            string
	Kind              string // domain.PendingAuthorizationKind*
	SessionID         string
	MaxConcurrentRuns int
}

// authorizationCorrelationID is the stable idempotency key for an ask. The same
// (subject, operation, target) always hashes to the same id, so a repeated raise
// reuses the one pending row and the one operator notification.
func authorizationCorrelationID(subject, operation, target string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(subject) + "\x00" + strings.TrimSpace(operation) + "\x00" + strings.TrimSpace(target)))
	return "auth-" + hex.EncodeToString(sum[:16])
}

// RaiseAuthorizationApproval records (idempotently) a pending authorization ask
// for a verified caller that lacks a grant, and raises the operator decision.
// It returns raised=true when the ask is outstanding (new or already pending),
// so the handler answers "approval pending". It returns ErrAuthorizationFlood
// when the subject is already at its outstanding-ask cap, so the handler refuses
// WITHOUT notifying the operator. It never issues authority — only an operator
// approval does that.
func (o *Orchestrator) RaiseAuthorizationApproval(ctx context.Context, req AuthorizationApprovalRequest) (bool, error) {
	if o == nil || o.pendingAuthorizations == nil {
		// Without a pending store the path degrades to the prior hard refuse.
		return false, nil
	}
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		return false, nil
	}
	now := o.now()
	id := authorizationCorrelationID(subject, req.Operation, req.Target)

	existing, err := o.pendingAuthorizations.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if existing != nil && existing.Outstanding(now) {
		// Idempotent: the same ask is already outstanding. Do NOT re-emit a
		// notification or re-count it against the cap; just report it pending.
		return true, nil
	}

	// A genuinely new (or re-raisable) ask. Enforce the per-subject anti-flood
	// cap before persisting or notifying.
	outstanding, err := o.pendingAuthorizations.CountOutstandingForSubject(ctx, subject, now)
	if err != nil {
		return false, err
	}
	if outstanding >= MaxOutstandingAuthorizationRequestsPerSubject {
		return false, ErrAuthorizationFlood
	}

	kind := req.Kind
	if kind != domain.PendingAuthorizationKindOneTime {
		kind = domain.PendingAuthorizationKindStanding
	}
	ttl := standingGrantIssueTTL
	if kind == domain.PendingAuthorizationKindOneTime {
		ttl = oneTimeGrantIssueTTL
	}
	pending := &domain.PendingAuthorizationRequest{
		ID:                id,
		Subject:           subject,
		Operation:         strings.TrimSpace(req.Operation),
		Target:            strings.TrimSpace(req.Target),
		Kind:              kind,
		SessionID:         strings.TrimSpace(req.SessionID),
		GrantTTL:          ttl,
		MaxConcurrentRuns: req.MaxConcurrentRuns,
		State:             domain.PendingAuthorizationPending,
		RaisedAt:          now.UTC(),
		ExpiresAt:         now.Add(pendingAuthorizationWindow).UTC(),
	}
	if err := o.pendingAuthorizations.Put(ctx, pending); err != nil {
		return false, err
	}

	if o.approvalPublisher != nil {
		event := o.authorizationDecisionEvent(ctx, pending)
		if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
			// Best-effort notify, like run approval: the durable pending record
			// already stands, so the operator can still approve from the UI, and a
			// later raise re-attempts the notification.
			approvalLog().Warn("authorization decision event publish failed",
				"subject", subject, obs.KeyError, err.Error())
		}
	}
	return true, nil
}

// ErrAuthorizationFlood is returned when a subject already has the maximum
// number of outstanding distinct authorization asks. The handler refuses
// without raising another operator notification.
var ErrAuthorizationFlood = fmt.Errorf("too many outstanding authorization requests for this subject")

// authorizationDecisionEvent builds the operator Ask. The question states
// exactly what a one-tap approve grants — the grant KIND, subject, bounds and
// expiry — so the operator can never approve more than they can see.
func (o *Orchestrator) authorizationDecisionEvent(ctx context.Context, p *domain.PendingAuthorizationRequest) eventbus.DomainEvent {
	var question, subjectLine string
	bound := "no concurrency limit"
	if p.MaxConcurrentRuns > 0 {
		bound = fmt.Sprintf("max %d concurrent runs", p.MaxConcurrentRuns)
	}
	if p.Kind == domain.PendingAuthorizationKindOneTime {
		subjectLine = fmt.Sprintf("Approve one run for %s?", p.Subject)
		question = fmt.Sprintf("Approve a short-lived (%s) run-creation grant for %q (%s)? It expires automatically and is revocable anytime.",
			humanizeGrantTTL(p.GrantTTL), p.Subject, bound)
	} else {
		subjectLine = fmt.Sprintf("Approve run creation for %s?", p.Subject)
		question = fmt.Sprintf("Approve a bounded %s run-creation standing grant for %q (%s)? It is owner-attributed to you, revocable anytime, and expires automatically.",
			humanizeGrantTTL(p.GrantTTL), p.Subject, bound)
	}

	payload := map[string]any{
		"subject":  subjectLine,
		"question": question,
		"options": []any{
			map[string]any{"key": "approve", "label": "Approve grant"},
			map[string]any{"key": "reject", "label": "Reject"},
		},
		"recommended": "reject",
		// Issuing a grant is reversible: it is bounded, revocable, and expires. No
		// silent default is set, so nothing is granted without an explicit answer.
		"reversible": true,
		"urgency":    "normal",
		"correlation": map[string]any{
			"authorization_request_id": p.ID,
			"subject":                  p.Subject,
			"operation":                p.Operation,
			"target":                   p.Target,
			"kind":                     p.Kind,
		},
	}
	if base := strings.TrimRight(o.resolveWebConsoleNoticeBase(ctx), "/"); base != "" {
		payload["context_url"] = base + "/?authorization=" + p.ID
	}

	return eventbus.DomainEvent{
		Source:    runApprovalSource,
		EventType: AuthorizationDecisionEventType,
		// Deterministic occurrence keeps the event id stable for the pending
		// window, so a redelivery (or a re-raise inside the window) maps to the
		// same ask instead of a duplicate notification.
		Occurred: p.RaisedAt.UTC().Truncate(pendingAuthorizationWindow),
		Payload:  payload,
	}
}

// NotifyAuthorizationRequested raises the operator decision event for an
// in-session (interactive) authorization request the moment it is persisted, so
// the operator is pushed to the review UI instead of discovering a pending
// request only by chance. Delivery is best-effort, exactly like the run-approval
// and standing-grant notifications: a publish failure is logged and never fails
// the request — the durable pending record already stands and the operator can
// still approve from the UI. A nil publisher is a silent no-op (no regression).
//
// This NEVER issues authority: the interactive grant is minted only by
// ApproveAuthorization under a live independent-human proof and matching process
// binding. The event carries no path to the grant.
func (o *Orchestrator) NotifyAuthorizationRequested(ctx context.Context, g *domain.AuthorizationGrant) {
	if o == nil || o.approvalPublisher == nil || g == nil {
		return
	}
	event := o.interactiveAuthorizationEvent(ctx, g)
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("interactive authorization decision event publish failed",
			"authorization_id", g.ID.String(), obs.KeyError, err.Error())
	}
}

// interactiveAuthorizationEvent builds the operator Ask for a pending interactive
// authorization request. The question states exactly what is being asked for —
// the reviewed policy name, the targets, the session, and the requested lifetime
// — so the operator knows what they are approving before they open the UI. The
// options point the operator at the in-UI review (there is no one-tap approve:
// interactive consent requires the operator's live verified login at the UI).
func (o *Orchestrator) interactiveAuthorizationEvent(ctx context.Context, g *domain.AuthorizationGrant) eventbus.DomainEvent {
	targets := strings.Join(g.Targets, ", ")
	question := fmt.Sprintf("An agent session (%s) is requesting authority %q for %s, for up to %s. Review and approve it in the web console with your verified login.",
		strings.TrimSpace(g.HarnessKind), g.Policy, targets, humanizeDurationSeconds(g.DurationSeconds))

	payload := map[string]any{
		"subject":  fmt.Sprintf("Authorization requested: %s", g.Policy),
		"question": question,
		"options": []any{
			map[string]any{"key": "review", "label": "Review in web console"},
		},
		// No recommended default and no reversible-grant claim: nothing is granted
		// by answering this notification — approval happens only at the UI under a
		// live verified login. This is a push to the operator, not a decision.
		"urgency": "normal",
		"correlation": map[string]any{
			"authorization_id": g.ID.String(),
			"run_id":           g.RunID.String(),
			"harness_kind":     g.HarnessKind,
			"harness_session":  g.HarnessSession,
			"policy":           g.Policy,
		},
	}
	if base := strings.TrimRight(o.resolveWebConsoleNoticeBase(ctx), "/"); base != "" {
		payload["context_url"] = base + "/?authorization=" + g.ID.String()
	}

	return eventbus.DomainEvent{
		Source:    runApprovalSource,
		EventType: InteractiveAuthorizationDecisionEventType,
		// A stable occurrence (the request's creation time) keeps the event id
		// deterministic, so a redelivery maps to the same ask rather than a
		// duplicate notification.
		Occurred: g.CreatedAt.UTC(),
		Payload:  payload,
	}
}

// humanizeDurationSeconds renders a grant lifetime for the operator notification.
func humanizeDurationSeconds(seconds int64) string {
	return humanizeGrantTTL(time.Duration(seconds) * time.Second)
}

func humanizeGrantTTL(d time.Duration) string {
	days := int(d.Hours()) / 24
	if days >= 1 {
		if days == 1 {
			return "1-day"
		}
		return fmt.Sprintf("%d-day", days)
	}
	hours := int(d.Hours())
	if hours <= 1 {
		return "1-hour"
	}
	return fmt.Sprintf("%d-hour", hours)
}

// ResolveAuthorizationFromHub applies a notification-hub ask resolution that
// answers an authorization decision this scenario raised. It reports
// handled=true when the resolution belongs to an authorization ask (so the
// webhook stops routing it), and false for any other decision (it then falls
// through to the effort-decision resolver). Approve issues a bounded grant
// attributed to the approving human; reject records the refusal; a resolution
// for an unknown/already-decided ask is a benign no-op.
func (o *Orchestrator) ResolveAuthorizationFromHub(ctx context.Context, res supervision.DecisionResolution) (bool, error) {
	if o == nil || res.SourceEventType != AuthorizationDecisionEventType {
		return false, nil
	}
	if o.pendingAuthorizations == nil || o.standingGrants == nil {
		return true, fmt.Errorf("authorization approval resolution received but the authorization stores are not configured")
	}
	if res.State != "answered" {
		approvalLog().Info("authorization resolution ignored (no operator answer)",
			"state", res.State, "ask_id", res.AskID)
		return true, nil
	}
	id := strings.TrimSpace(res.Correlation["authorization_request_id"])
	if id == "" {
		return true, fmt.Errorf("authorization resolution lacks an authorization_request_id correlation")
	}
	pending, err := o.pendingAuthorizations.Get(ctx, id)
	if err != nil {
		return true, err
	}
	if pending == nil {
		// Unknown ask (never raised, or pruned): acknowledge and drop so
		// vrooli-events does not retry a poison message forever.
		approvalLog().Info("authorization resolution references an unknown ask; ignoring", "ask_id", res.AskID, "authorization_request_id", id)
		return true, nil
	}
	if pending.State != domain.PendingAuthorizationPending {
		// Idempotent: already decided. A redelivery must not re-issue a grant.
		approvalLog().Info("authorization resolution is a no-op; ask already decided",
			"authorization_request_id", id, "state", pending.State)
		return true, nil
	}

	actor := strings.TrimSpace(res.AnsweredBy)
	now := o.now()
	switch res.Answer {
	case "approve":
		// HUMAN-ONLY issuance: the grant's authority comes from a verified human
		// approver carried on the hub resolution. An empty AnsweredBy is not an
		// operator authorization — refuse to mint and leave the ask pending.
		if actor == "" {
			return true, fmt.Errorf("authorization approval carries no approver identity; refusing to issue a grant")
		}
		ttl := pending.GrantTTL
		if ttl <= 0 {
			ttl = oneTimeGrantIssueTTL
		}
		grant, err := o.CreateStandingGrant(ctx, StandingGrantRequest{
			Subject:   pending.Subject,
			Approver:  actor,
			SessionID: pending.SessionID,
			Limits:    authn.StandingGrantLimits{MaxConcurrentRuns: pending.MaxConcurrentRuns},
			TTL:       ttl,
		})
		if err != nil {
			return true, fmt.Errorf("issue grant on authorization approval: %w", err)
		}
		pending.State = domain.PendingAuthorizationApproved
		pending.GrantID = grant.ID
	case "reject":
		pending.State = domain.PendingAuthorizationRejected
	default:
		return true, fmt.Errorf("authorization resolution has an unknown answer %q", res.Answer)
	}
	resolvedAt := now.UTC()
	pending.ResolvedAt = &resolvedAt
	pending.ResolvedBy = actor
	if err := o.pendingAuthorizations.Put(ctx, pending); err != nil {
		return true, fmt.Errorf("record authorization resolution: %w", err)
	}
	approvalLog().Info("authorization resolution applied from notification-hub",
		"authorization_request_id", id, "answer", res.Answer, "actor", actor, "subject", pending.Subject)
	return true, nil
}
