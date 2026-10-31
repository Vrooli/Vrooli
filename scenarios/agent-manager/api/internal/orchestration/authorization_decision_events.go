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
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/cliutil"
)

// AuthorizationDecisionEventType is the decision event raised when a verified
// caller needs operator approval for a costly operation. The
// `.decision_requested.v1` suffix is the generic contract notification-hub
// already subscribes to, so no hub change is needed.
const AuthorizationDecisionEventType = "agent_manager.authorization.decision_requested.v1"

// InteractiveAuthorizationNotificationEventType is a neutral FYI push to the
// native consent page. Notification Hub answers are never connected to the
// interactive authorization authority path.
const InteractiveAuthorizationNotificationEventType = "agent_manager.authorization_requested.notification.v1"

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
		// Idempotent: a successful publication is not repeated, but a durable
		// pending row whose first publication failed remains retryable.
		if existing.PublishedAt == nil {
			if err := o.publishAuthorizationDecision(ctx, existing); err != nil {
				approvalLog().Warn("authorization decision event publish failed", "subject", subject, obs.KeyError, err.Error())
			}
		}
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
	pending.DecisionBigPicture, pending.DecisionTechnicalDetails = authorizationDecisionContext(pending)
	pending.EventID = operatordecision.EventID(o.authorizationDecisionEvent(ctx, pending))
	if err := o.pendingAuthorizations.Put(ctx, pending); err != nil {
		return false, err
	}
	if err := o.publishAuthorizationDecision(ctx, pending); err != nil {
		// The pending obligation stands; a later identical raise retries the event.
		approvalLog().Warn("authorization decision event publish failed", "subject", subject, obs.KeyError, err.Error())
	}
	return true, nil
}

func (o *Orchestrator) publishAuthorizationDecision(ctx context.Context, pending *domain.PendingAuthorizationRequest) error {
	if o.approvalPublisher == nil || pending == nil {
		return nil
	}
	event := o.authorizationDecisionEvent(ctx, pending)
	if pending.EventID != "" && operatordecision.EventID(event) != pending.EventID {
		return fmt.Errorf("authorization decision event changed within pending generation %s", pending.ID)
	}
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		return err
	}
	publishedAt := o.now().UTC()
	pending.EventID = operatordecision.EventID(event)
	pending.PublishedAt = &publishedAt
	return o.pendingAuthorizations.Put(ctx, pending)
}

// ErrAuthorizationFlood is returned when a subject already has the maximum
// number of outstanding distinct authorization asks. The handler refuses
// without raising another operator notification.
var ErrAuthorizationFlood = fmt.Errorf("too many outstanding authorization requests for this subject")

// authorizationDecisionContext explains the effect and exact bounds available
// from the durable request. It does not invent a caller-specific business reason.
func authorizationDecisionContext(p *domain.PendingAuthorizationRequest) (string, string) {
	if p == nil {
		return "", ""
	}
	bigPicture := "The verified caller lacks a covering grant for this operation. Approval creates only the permission described here; it does not execute the requested operation. Rejection issues no grant from this request; existing grants are unaffected."
	concurrency := "no concurrency limit"
	if p.MaxConcurrentRuns > 0 {
		concurrency = fmt.Sprintf("at most %d concurrent runs", p.MaxConcurrentRuns)
	}
	technical := fmt.Sprintf(
		"Subject: %s\nOperation: %s\nTarget: %s\nGrant kind: %s\nGrant lifetime: %s\nConcurrency: %s\nApproval is explicit, owner-attributed, and revocable; no answer issues no grant.",
		p.Subject, p.Operation, p.Target, p.Kind, humanizeGrantTTL(p.GrantTTL), concurrency,
	)
	if p.SessionID != "" {
		technical += "\nGrant scope: session-scoped."
	}
	return bigPicture, technical
}

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

	contextURL := ""
	if base := strings.TrimRight(o.resolveWebConsoleNoticeBase(ctx), "/"); base != "" {
		contextURL = base + "/?authorization=" + p.ID
	}

	facts := operatordecision.Facts{
		Subject:          subjectLine,
		Question:         question,
		BigPicture:       p.DecisionBigPicture,
		TechnicalDetails: p.DecisionTechnicalDetails,
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve grant"},
			{Key: "reject", Label: "Reject"},
		},
		Recommended: "reject",
		// Issuing a grant is reversible: it is bounded, revocable, and expires. No
		// silent default is set, so nothing is granted without an explicit answer.
		Reversible: true,
		Urgency:    "normal",
		ContextURL: contextURL,
		Correlation: map[string]string{
			"authorization_request_id": p.ID,
			"subject":                  p.Subject,
			"operation":                p.Operation,
			"target":                   p.Target,
			"kind":                     p.Kind,
		},
	}

	// Deterministic occurrence keeps the event id stable for the pending window,
	// so a redelivery (or a re-raise inside the window) maps to the same ask
	// instead of a duplicate notification. Emitted through BuildEventTyped so the
	// established underscore event type and the hyphen Source are both preserved.
	return operatordecision.BuildEventTyped(runApprovalSource, AuthorizationDecisionEventType,
		p.RaisedAt.UTC().Truncate(pendingAuthorizationWindow), facts)
}

// NotifyAuthorizationRequested sends a durable FYI event for a pending
// interactive request. The event links to native consent; Notification Hub
// answers are never connected to interactive authority issuance.
func (o *Orchestrator) NotifyAuthorizationRequested(ctx context.Context, g *domain.AuthorizationGrant) {
	if err := o.notifyAuthorizationRequested(ctx, g); err != nil && g != nil {
		o.logAuthorizationNotificationError(g.ID, err)
	}
}

const (
	authorizationNotificationScanLimit            = 100
	authorizationNotificationAttemptLimit         = 10
	authorizationNotificationRetryTimeout         = 10 * time.Second
	authorizationNotificationPublishTimeout       = 5 * time.Second
	authorizationNotificationFailureRecordTimeout = 500 * time.Millisecond
)

// retryPendingAuthorizationNotifications retries a bounded repository page.
// The reconciler owns the keyset cursor; this method checks live request
// eligibility and delegates to the same immutable-snapshot/CAS publisher used
// by the original request path.
func (o *Orchestrator) retryPendingAuthorizationNotifications(ctx context.Context, afterID string, scanLimit, attemptLimit int) (string, error) {
	if o == nil || o.approvalPublisher == nil || scanLimit <= 0 || attemptLimit <= 0 {
		return "", nil
	}
	if o.authorizations == nil {
		return afterID, errors.New("authorization repository is unavailable")
	}
	retryRepo, ok := o.authorizations.(domain.AuthorizationNotificationRetryRepository)
	if !ok {
		return afterID, errors.New("authorization notification retry repository is unavailable")
	}
	page, err := retryRepo.ListNotificationRetryScanPage(ctx, afterID, scanLimit)
	if err != nil {
		return afterID, err
	}
	if len(page) == 0 && afterID != "" {
		page, err = retryRepo.ListNotificationRetryScanPage(ctx, "", scanLimit)
		if err != nil {
			return afterID, err
		}
	}
	if len(page) == 0 {
		return "", nil
	}

	now := o.now()
	attempts := 0
	lastCompletedID := afterID
	for _, grant := range page {
		if err := ctx.Err(); err != nil {
			return lastCompletedID, err
		}
		if grant == nil {
			continue
		}
		if attempts >= attemptLimit {
			break
		}
		if grant.State == domain.AuthorizationPending && grant.EffectiveState(now) == domain.AuthorizationPending && grant.NotificationPublishedAt == nil && grant.Binding.PID > 0 {
			live, err := cliutil.ObserveProcessBinding(grant.Binding.PID)
			if ctx.Err() != nil {
				return lastCompletedID, ctx.Err()
			}
			if err == nil && live == grant.Binding {
				attempts++
				if err := o.notifyAuthorizationRequested(ctx, grant); err != nil {
					if ctx.Err() != nil {
						if recordErr := o.recordCancelledAuthorizationNotificationFailure(ctx, grant.ID, err); recordErr != nil {
							o.logAuthorizationNotificationError(grant.ID, recordErr)
						}
						o.logAuthorizationNotificationError(grant.ID, err)
						// Keep this row eligible next cycle. A timed-out persistence
						// operation must not advance the in-memory keyset past it.
						return lastCompletedID, ctx.Err()
					}
					o.logAuthorizationNotificationError(grant.ID, err)
				}
			}
		}
		lastCompletedID = grant.ID.String()
	}
	if attempts >= attemptLimit {
		return lastCompletedID, nil
	}
	if len(page) < scanLimit {
		return "", nil
	}
	return lastCompletedID, nil
}

func (o *Orchestrator) notifyAuthorizationRequested(ctx context.Context, g *domain.AuthorizationGrant) error {
	if o == nil || o.authorizations == nil || g == nil || g.ID == uuid.Nil {
		return nil
	}
	current, err := o.authorizations.Get(ctx, g.ID)
	if err != nil {
		return err
	}
	if current == nil || current.State != domain.AuthorizationPending || current.EffectiveState(o.now()) != domain.AuthorizationPending || current.NotificationPublishedAt != nil {
		return nil
	}
	if current.NotificationSnapshot == nil {
		snapshot, buildErr := o.authorizationNotificationSnapshot(ctx, current)
		next := cloneAuthorizationGrant(current)
		if buildErr != nil {
			next.NotificationUnreached = boundedAuthorizationNotificationFailure(buildErr)
		} else {
			next.NotificationSnapshot = snapshot
			next.NotificationUnreached = ""
		}
		stored, storeErr := o.authorizations.UpdateNotification(ctx, current, next)
		if storeErr != nil {
			return storeErr
		}
		if stored {
			current = next
		} else {
			current, err = o.authorizations.Get(ctx, g.ID)
			if err != nil {
				return err
			}
		}
		if current == nil || current.State != domain.AuthorizationPending || current.EffectiveState(o.now()) != domain.AuthorizationPending {
			return nil
		}
		// A lost CAS may mean another publisher stored this generation first.
		// Re-read and use that exact snapshot; never rebuild a changed link.
		if current.NotificationSnapshot == nil {
			if buildErr != nil {
				return buildErr
			}
			return errors.New("authorization notification snapshot unavailable after concurrent update")
		}
	}
	if current.State != domain.AuthorizationPending || current.EffectiveState(o.now()) != domain.AuthorizationPending || current.NotificationPublishedAt != nil || current.NotificationSnapshot == nil {
		return nil
	}
	if o.approvalPublisher == nil {
		return o.markAuthorizationNotificationUnreached(ctx, current, "event publisher unavailable")
	}
	event := eventFromAuthorizationNotification(current.NotificationSnapshot)
	if operatordecision.EventID(event) != current.NotificationSnapshot.EventID {
		return errors.New("stored authorization notification snapshot identity is invalid")
	}
	publishCtx, cancel := context.WithTimeout(ctx, authorizationNotificationPublishTimeout)
	err = o.approvalPublisher.PublishDomainEvent(publishCtx, event)
	cancel()
	if err != nil {
		bounded := boundedAuthorizationNotificationFailure(err)
		markErr := o.markAuthorizationNotificationUnreached(ctx, current, bounded)
		if markErr != nil {
			return errors.Join(err, markErr)
		}
		return err
	}
	// Record the receipt only against the exact still-pending body and snapshot.
	// If a human decision or a replacement request won while publishing, the CAS
	// fails harmlessly and never writes the stale pending body over that state.
	fresh, err := o.authorizations.Get(ctx, g.ID)
	if err != nil {
		return err
	}
	if fresh == nil || fresh.State != domain.AuthorizationPending || fresh.NotificationPublishedAt != nil || !sameAuthorizationNotification(fresh.NotificationSnapshot, current.NotificationSnapshot) {
		return nil
	}
	next := cloneAuthorizationGrant(fresh)
	publishedAt := o.now().UTC()
	next.NotificationPublishedAt = &publishedAt
	next.NotificationUnreached = ""
	if _, err := o.authorizations.UpdateNotification(ctx, fresh, next); err != nil {
		return err
	}
	return nil
}

func (o *Orchestrator) authorizationNotificationSnapshot(ctx context.Context, g *domain.AuthorizationGrant) (*domain.AuthorizationNotificationSnapshot, error) {
	base, err := authorizationPublicBase(ctx)
	if err != nil {
		return nil, err
	}
	consentURL, err := authorizationConsentURL(base, g.ID)
	if err != nil {
		return nil, fmt.Errorf("native authorization link unavailable: %w", err)
	}
	payload := map[string]any{
		"title":       "Agent needs approval: " + g.Policy,
		"body":        fmt.Sprintf("%s wants %s on %s.\nThe %s permission lifetime starts only after you approve on the Agent Manager page; this notification grants no authority.", strings.TrimSpace(g.HarnessKind), g.Policy, strings.Join(g.Targets, ", "), humanizeDurationSeconds(g.DurationSeconds)),
		"category":    "authorization-requested",
		"urgency":     "normal",
		"sensitivity": "private",
		"context_url": consentURL,
		"dedupe_key":  g.ID.String(),
		"correlation": map[string]any{"authorization_id": g.ID.String(), "run_id": g.RunID.String(), "policy": g.Policy},
	}
	event := eventbus.DomainEvent{Source: runApprovalSource, EventType: InteractiveAuthorizationNotificationEventType, Occurred: g.CreatedAt.UTC(), Payload: payload}
	return &domain.AuthorizationNotificationSnapshot{Source: event.Source, EventType: event.EventType, EventID: operatordecision.EventID(event), Occurred: event.Occurred, Payload: payload}, nil
}

func eventFromAuthorizationNotification(snapshot *domain.AuthorizationNotificationSnapshot) eventbus.DomainEvent {
	return eventbus.DomainEvent{Source: snapshot.Source, EventType: snapshot.EventType, Occurred: snapshot.Occurred, Payload: snapshot.Payload}
}

func sameAuthorizationNotification(a, b *domain.AuthorizationNotificationSnapshot) bool {
	return a != nil && b != nil && a.EventID == b.EventID && a.EventType == b.EventType && a.Source == b.Source && a.Occurred.Equal(b.Occurred)
}

func authorizationPublicBase(ctx context.Context) (string, error) {
	configured := strings.TrimSpace(os.Getenv("AGENT_MANAGER_AUTHORIZATION_URL"))
	if configured == "" {
		var err error
		configured, err = resolveExternalURLForNotice(ctx, "agent-manager")
		if err != nil {
			return "", fmt.Errorf("canonical Agent Manager public origin unresolved: %w", err)
		}
	}
	u, err := url.Parse(configured)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("canonical Agent Manager public origin is invalid")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", errors.New("canonical Agent Manager public origin must use HTTP or HTTPS")
	}
	if u.Scheme == "http" && !isLoopbackHost(u.Hostname()) {
		return "", errors.New("non-loopback Agent Manager public origin must use HTTPS")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func authorizationConsentURL(base string, id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", errors.New("authorization identifier is missing")
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return "", errors.New("canonical Agent Manager origin is invalid")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/authorizations/" + id.String() + "/approve"
	return u.String(), nil
}

func cloneAuthorizationGrant(g *domain.AuthorizationGrant) *domain.AuthorizationGrant {
	cp := *g
	if g.NotificationPublishedAt != nil {
		published := *g.NotificationPublishedAt
		cp.NotificationPublishedAt = &published
	}
	if g.NotificationSnapshot != nil {
		snapshot := *g.NotificationSnapshot
		payload := make(map[string]any, len(g.NotificationSnapshot.Payload))
		for key, value := range g.NotificationSnapshot.Payload {
			payload[key] = value
		}
		snapshot.Payload = payload
		cp.NotificationSnapshot = &snapshot
	}
	return &cp
}

const authorizationNotificationFailureLimit = 256

func boundedAuthorizationNotificationFailure(err error) string {
	if err == nil {
		return "notification publication failed"
	}
	reason := strings.TrimSpace(err.Error())
	if len(reason) > authorizationNotificationFailureLimit {
		reason = reason[:authorizationNotificationFailureLimit]
	}
	if reason == "" {
		return "notification publication failed"
	}
	return reason
}

func (o *Orchestrator) markAuthorizationNotificationUnreached(ctx context.Context, current *domain.AuthorizationGrant, reason string) error {
	if current == nil || o == nil || o.authorizations == nil || current.State != domain.AuthorizationPending || current.EffectiveState(o.now()) != domain.AuthorizationPending {
		return nil
	}
	next := cloneAuthorizationGrant(current)
	next.NotificationUnreached = boundedAuthorizationNotificationFailure(errors.New(reason))
	if _, err := o.authorizations.UpdateNotification(ctx, current, next); err != nil {
		return err
	}
	return nil
}

func (o *Orchestrator) recordCancelledAuthorizationNotificationFailure(parent context.Context, id uuid.UUID, cause error) error {
	if o == nil || o.authorizations == nil || id == uuid.Nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), authorizationNotificationFailureRecordTimeout)
	defer cancel()
	current, err := o.authorizations.Get(ctx, id)
	if err != nil {
		return err
	}
	if current == nil || current.State != domain.AuthorizationPending || current.EffectiveState(o.now()) != domain.AuthorizationPending || current.NotificationPublishedAt != nil || current.NotificationUnreached != "" {
		return nil
	}
	reason := "notification retry ended before receipt: " + boundedAuthorizationNotificationFailure(cause)
	return o.markAuthorizationNotificationUnreached(ctx, current, reason)
}

func (o *Orchestrator) logAuthorizationNotificationError(id uuid.UUID, err error) {
	approvalLog().Warn("interactive authorization notification unreached",
		"authorization_id", id.String(), obs.KeyError, err.Error())
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
	if res.RequesterRef == "" || pending.EventID == "" || res.RequesterRef != pending.EventID {
		approvalLog().Info("authorization resolution belongs to a different request generation; ignoring",
			"authorization_request_id", id, "requester_ref", res.RequesterRef, "event_id", pending.EventID)
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
