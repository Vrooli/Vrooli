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
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/eventbus"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/cli-core/cliutil"
)

// AuthorizationDecisionEventType is the decision event raised when a verified
// caller needs operator approval for a costly operation. The
// `.decision_requested.v1` suffix is the generic contract notification-hub
// already subscribes to, so no hub change is needed.
const AuthorizationDecisionEventType = "agent_manager.authorization.decision_requested.v1"

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

const researchBudgetIngestOperation = "research-budget.ingest"
const browserEvidenceRecordOperation = "browser-evidence.record"

// TreasuryServiceSubject is Treasury's service principal at Agent Manager: the
// only subject that may submit research-budget evidence, and the subject of the
// one-time grant an operator approves for it. Treasury holds one principal per
// peer (treasury-money-ledger, treasury-notification-hub), so a token Treasury
// holds for another peer is not accepted here. It is issued by
// `scenario-authenticator service-principals sync --apply` from Treasury's
// .vrooli/service-principals.json.
const TreasuryServiceSubject = "treasury-agent-manager"

var researchBudgetIngestTargetPattern = regexp.MustCompile(`^agent-manager:run/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var browserEvidenceRecordTargetPattern = researchBudgetIngestTargetPattern

// BrowserEvidenceServiceSubject is BAS's dedicated Agent Manager recorder
// principal. Its declaration alone grants no authority; a human-approved
// one-run grant is still required for browser evidence authorization.
const BrowserEvidenceServiceSubject = "browser-automation-studio-agent-manager"

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
	if !validAuthorizationApprovalRequest(req) {
		return false, domain.NewValidationError("authorization_request", "operation and target are outside supported approval paths")
	}
	if strings.TrimSpace(req.Operation) == browserEvidenceRecordOperation {
		runID, err := uuid.Parse(strings.TrimPrefix(strings.TrimSpace(req.Target), "agent-manager:run/"))
		if err != nil || !o.IsBrowserEvidenceRun(ctx, runID) {
			return false, domain.NewValidationError("authorization_request", "active browser-capable run required")
		}
	}
	now := o.now()
	id := authorizationCorrelationID(subject, req.Operation, req.Target)

	existing, err := o.pendingAuthorizations.Get(ctx, id)
	if err != nil {
		return false, err
	}
	if existing != nil && existing.Outstanding(now) {
		if err := o.refreshStaleAsk(ctx, existing); err != nil {
			approvalLog().Warn("authorization decision refresh failed", "subject", subject, obs.KeyError, err.Error())
		}
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
		SubjectKey:        authorizationSubjectKey(id),
		RaisedAt:          now.UTC(),
		ExpiresAt:         now.Add(pendingAuthorizationWindow).UTC(),
	}
	freezeAuthorizationDecision(pending)
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

// validAuthorizationApprovalRequest keeps general service asks on the existing
// run.create path and admits only named, exact-run reporting capabilities.
// Reporting approvals are one-time, have no standing/concurrency scope and
// cannot use wildcard targets.
func validAuthorizationApprovalRequest(req AuthorizationApprovalRequest) bool {
	operation, target, subject := strings.TrimSpace(req.Operation), strings.TrimSpace(req.Target), strings.TrimSpace(req.Subject)
	if operation == "run.create" {
		return target != ""
	}
	return req.Kind == domain.PendingAuthorizationKindOneTime && req.SessionID == "" && req.MaxConcurrentRuns == 0 &&
		((operation == researchBudgetIngestOperation && subject == TreasuryServiceSubject && researchBudgetIngestTargetPattern.MatchString(target)) ||
			(operation == browserEvidenceRecordOperation && subject == BrowserEvidenceServiceSubject && browserEvidenceRecordTargetPattern.MatchString(target)))
}

func isResearchBudgetIngestRequest(p *domain.PendingAuthorizationRequest) bool {
	return p != nil && p.Subject == TreasuryServiceSubject && p.Operation == researchBudgetIngestOperation &&
		researchBudgetIngestTargetPattern.MatchString(p.Target) && p.Kind == domain.PendingAuthorizationKindOneTime
}

func isBrowserEvidenceRecordRequest(p *domain.PendingAuthorizationRequest) bool {
	return p != nil && p.Subject == BrowserEvidenceServiceSubject && p.Operation == browserEvidenceRecordOperation &&
		browserEvidenceRecordTargetPattern.MatchString(p.Target) && p.Kind == domain.PendingAuthorizationKindOneTime && p.SessionID == ""
}

// IsBrowserEvidenceRun binds an approval ask to a currently active persisted
// run that actually has a resolved browser capability. It is rechecked when an
// operator answers so a stopped run cannot receive stale authority.
func (o *Orchestrator) IsBrowserEvidenceRun(ctx context.Context, runID uuid.UUID) bool {
	if o == nil || runID == uuid.Nil {
		return false
	}
	run, err := o.GetStoredRun(ctx, runID)
	if err != nil || run == nil || run.ID != runID || run.Status != domain.RunStatusRunning || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		return false
	}
	return browsercapability.Validate(*run.ResolvedConfig.Features.BrowserCapability) == nil
}

func (o *Orchestrator) createBrowserEvidenceRecordGrant(ctx context.Context, pending *domain.PendingAuthorizationRequest, approver string) (*domain.StandingGrant, error) {
	if o.standingGrants == nil || !isBrowserEvidenceRecordRequest(pending) || strings.TrimSpace(approver) == "" {
		return nil, domain.NewValidationError("authorization_request", "exact BAS browser-evidence request required")
	}
	runID, err := uuid.Parse(strings.TrimPrefix(pending.Target, "agent-manager:run/"))
	if err != nil || !o.IsBrowserEvidenceRun(ctx, runID) {
		return nil, domain.NewValidationError("authorization_request", "active browser-capable run required")
	}
	ttl := pending.GrantTTL
	if ttl <= 0 || ttl > oneTimeGrantIssueTTL {
		return nil, domain.NewValidationError("ttl", "browser evidence grant lifetime must be at most one hour")
	}
	now := o.now()
	id := uuid.NewString()
	expires := now.Add(ttl)
	grant := &domain.StandingGrant{
		ID: id, Subject: BrowserEvidenceServiceSubject, Grant: &requestidentity.OperationGrant{
			ID: id, Policy: "browser-evidence-record", Approver: strings.TrimSpace(approver),
			Operations: []string{browserEvidenceRecordOperation}, Targets: []string{pending.Target}, ExpiresAt: expires,
		},
		State: domain.StandingGrantApproved, ApprovedBy: strings.TrimSpace(approver), ApprovedAt: now, ExpiresAt: expires,
	}
	if err := o.standingGrants.Put(ctx, grant); err != nil {
		return nil, err
	}
	return grant, nil
}

// IsResearchBudgetRun proves that the requested report target names an
// existing Agent Manager run. The report approval remains exact-run and
// one-time; a test tag is not an authorization boundary.
func (o *Orchestrator) IsResearchBudgetRun(ctx context.Context, runID uuid.UUID) bool {
	if o == nil || runID == uuid.Nil {
		return false
	}
	run, err := o.GetStoredRun(ctx, runID)
	return err == nil && run != nil && run.ID == runID
}

func (o *Orchestrator) createResearchBudgetIngestGrant(ctx context.Context, pending *domain.PendingAuthorizationRequest, approver string) (*domain.StandingGrant, error) {
	if o.standingGrants == nil || !isResearchBudgetIngestRequest(pending) || strings.TrimSpace(approver) == "" {
		return nil, domain.NewValidationError("authorization_request", "exact Treasury research-budget request required")
	}
	ttl := pending.GrantTTL
	if ttl <= 0 || ttl > oneTimeGrantIssueTTL {
		return nil, domain.NewValidationError("ttl", "research-budget grant lifetime must be at most one hour")
	}
	now := o.now()
	id := uuid.NewString()
	expires := now.Add(ttl)
	grant := &domain.StandingGrant{
		ID: id, Subject: TreasuryServiceSubject, Grant: &requestidentity.OperationGrant{
			ID: id, Policy: "research-budget-ingest", Approver: strings.TrimSpace(approver),
			Operations: []string{researchBudgetIngestOperation}, Targets: []string{pending.Target}, ExpiresAt: expires,
		},
		State: domain.StandingGrantApproved, ApprovedBy: strings.TrimSpace(approver), ApprovedAt: now, ExpiresAt: expires,
	}
	if err := o.standingGrants.Put(ctx, grant); err != nil {
		return nil, err
	}
	return grant, nil
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

// authorizationDecisionEvent builds the operator Ask from the frozen request.
// The cover states exactly what a one-tap approve grants (who, for how long,
// how many at once) so the operator can never approve more than they can see.
func (o *Orchestrator) authorizationDecisionEvent(ctx context.Context, p *domain.PendingAuthorizationRequest) eventbus.DomainEvent {
	facts := authorizationFacts(p)
	if base := strings.TrimRight(o.resolveWebConsoleNoticeBase(ctx), "/"); base != "" {
		facts.ContextURL = base + "/?authorization=" + p.ID
	}
	// Deterministic occurrence keeps the event id stable for the pending window,
	// so a redelivery (or a re-raise inside the window) maps to the same ask
	// instead of a duplicate notification. Emitted through BuildEventTyped so the
	// established underscore event type and the hyphen Source are both preserved.
	if p.SubjectKey != "" {
		// Orders this request against a later withdrawal of the same subject.
		facts.AssertedAt = p.RaisedAt.UTC().Truncate(time.Second)
	}
	return operatordecision.BuildEventTyped(runApprovalSource, AuthorizationDecisionEventType,
		p.RaisedAt.UTC().Truncate(pendingAuthorizationWindow), facts)
}

// authorizationSubjectKey names the one open operator decision for a caller's
// (subject, operation, target) ask: a re-raise after the window updates it
// instead of adding a second.
func authorizationSubjectKey(id string) string { return "authorization:" + id }

// refreshStaleAsk brings a still-pending ask raised by an older build up to
// date: one published without a subject key is closed by its event id, and one
// without frozen text (no page) is frozen now. The request is then published
// again as a new generation under its key. Idempotent: a current ask is left alone.
func (o *Orchestrator) refreshStaleAsk(ctx context.Context, p *domain.PendingAuthorizationRequest) error {
	if p == nil || p.State != domain.PendingAuthorizationPending || o.approvalPublisher == nil || (p.SubjectKey != "" && p.DecisionQuestion != "") {
		return nil
	}
	if p.SubjectKey == "" && p.EventID != "" && p.PublishedAt != nil {
		withdrawal := operatordecision.BuildWithdrawalEventTyped(runApprovalSource, AuthorizationDecisionEventType, o.now().UTC(), operatordecision.Withdrawal{
			RequesterRef: p.EventID, Reason: "Replaced by an updated version of this request.",
		})
		if err := o.approvalPublisher.PublishDomainEvent(ctx, withdrawal); err != nil {
			return err
		}
	}
	p.SubjectKey = authorizationSubjectKey(p.ID)
	freezeAuthorizationDecision(p)
	p.EventID = operatordecision.EventID(o.authorizationDecisionEvent(ctx, p))
	p.PublishedAt = nil
	return o.pendingAuthorizations.Put(ctx, p)
}

// closeAuthorizationAsk withdraws the open run-creation ask for subject because
// an operator-issued grant now covers it. It never issues, widens or checks
// authority: it only tells the hub the question no longer needs an answer. A
// failure is logged; the hub's own deadline remains the backstop.
func (o *Orchestrator) closeAuthorizationAsk(ctx context.Context, subject string) {
	subject = strings.TrimSpace(subject)
	if o == nil || o.pendingAuthorizations == nil || o.approvalPublisher == nil || subject == "" {
		return
	}
	id := authorizationCorrelationID(subject, "run.create", standingGrantRunCreateTarget)
	pending, err := o.pendingAuthorizations.Get(ctx, id)
	if err != nil || pending == nil || pending.State != domain.PendingAuthorizationPending {
		return
	}
	w := operatordecision.Withdrawal{SubjectKey: pending.SubjectKey, Reason: "A grant now covers this request, so it no longer needs your approval."}
	if w.SubjectKey == "" {
		w.RequesterRef = pending.EventID
	}
	if pending.PublishedAt != nil || pending.EventID != "" {
		event := operatordecision.BuildWithdrawalEventTyped(runApprovalSource, AuthorizationDecisionEventType, o.now().UTC(), w)
		if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
			approvalLog().Warn("authorization ask withdrawal failed", "subject", subject, obs.KeyError, err.Error())
			return
		}
	}
	resolvedAt := o.now().UTC()
	pending.State, pending.ResolvedAt = domain.PendingAuthorizationWithdrawn, &resolvedAt
	if err := o.pendingAuthorizations.Put(ctx, pending); err != nil {
		approvalLog().Warn("record authorization ask withdrawal failed", "subject", subject, obs.KeyError, err.Error())
	}
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
		legacy := grant.NotificationSnapshot != nil && grant.NotificationSnapshot.EventType != ConsentDecisionEventType
		if grant.State == domain.AuthorizationPending && grant.EffectiveState(now) == domain.AuthorizationPending && (grant.NotificationPublishedAt == nil || legacy) && grant.Binding.PID > 0 {
			live, err := cliutil.ObserveProcessBinding(grant.Binding.PID)
			if ctx.Err() != nil {
				return lastCompletedID, ctx.Err()
			}
			if err == nil && live == grant.Binding {
				attempts++
				if legacy {
					err = o.retireLegacyAuthorizationNotice(ctx, grant)
				}
				if err == nil {
					err = o.notifyAuthorizationRequested(ctx, grant)
				}
				if err != nil {
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

// retireLegacyAuthorizationNotice clears the stored FYI notice of a request
// raised before consent decisions existed, so the next publish raises it as the
// operator's decision with its page. It changes no terms or authority.
func (o *Orchestrator) retireLegacyAuthorizationNotice(ctx context.Context, g *domain.AuthorizationGrant) error {
	next := cloneAuthorizationGrant(g)
	next.NotificationSnapshot, next.NotificationPublishedAt, next.NotificationUnreached = nil, nil, ""
	_, err := o.authorizations.UpdateNotification(ctx, g, next)
	return err
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
	// The request is the operator's decision, with its page; the consent page
	// stays the only place it is approved (authorization_consent_decision.go).
	event := operatordecision.BuildEventTyped(runApprovalSource, ConsentDecisionEventType, g.CreatedAt.UTC(), consentFacts(g, consentURL, o.consentSessionLabel(ctx, g)))
	return &domain.AuthorizationNotificationSnapshot{Source: event.Source, EventType: event.EventType, EventID: operatordecision.EventID(event), Occurred: event.Occurred, Payload: event.Payload}, nil
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
	if o != nil && res.SourceEventType == ConsentDecisionEventType {
		// An interactive consent is approved only on Agent Manager's consent
		// page with the operator's own sign-in; a Hub answer grants nothing.
		approvalLog().Info("consent decision resolved in notification-hub; no authority changes",
			"state", res.State, "answer", res.Answer, "authorization_id", res.Correlation["authorization_id"])
		return true, nil
	}
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
		var grant *domain.StandingGrant
		if isResearchBudgetIngestRequest(pending) {
			pending.GrantTTL = ttl
			grant, err = o.createResearchBudgetIngestGrant(ctx, pending, actor)
		} else if isBrowserEvidenceRecordRequest(pending) {
			pending.GrantTTL = ttl
			grant, err = o.createBrowserEvidenceRecordGrant(ctx, pending, actor)
		} else {
			grant, err = o.createStandingGrant(ctx, StandingGrantRequest{
				Subject: pending.Subject, Approver: actor, SessionID: pending.SessionID,
				Limits: authn.StandingGrantLimits{MaxConcurrentRuns: pending.MaxConcurrentRuns}, TTL: ttl,
			})
		}
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

// consentSessionLabel is the label or tag of the run that asked, so the
// decision says what the session is working on. Empty when the request names no
// run Agent Manager knows; the decision then names the harness and session.
func (o *Orchestrator) consentSessionLabel(ctx context.Context, g *domain.AuthorizationGrant) string {
	if o == nil || o.runs == nil || g == nil || g.RunID == uuid.Nil {
		return ""
	}
	run, err := o.runs.Get(ctx, g.RunID)
	if err != nil || run == nil {
		return ""
	}
	if strings.TrimSpace(run.Label) != "" {
		return run.Label
	}
	return run.Tag
}
