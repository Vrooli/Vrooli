package approval

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"treasury/internal/module"
)

const (
	DecisionSource      = "treasury"
	DecisionKind        = "approval"
	DecisionWebhookPath = "/api/v1/events/notification-hub"
	DecisionAskWindow   = 24 * time.Hour
)

// DecisionAsker raises durable operator decisions after the Treasury queue
// record exists. The queue remains authoritative if event delivery is down.
type DecisionAsker interface {
	Raise(context.Context, Request) error
}

// DecisionFlow links Treasury's local queue to Notification Hub. Hub events
// can only resolve the exact local approval named in the Hub-verified
// correlation; they never alter mandate or budget policy.
type DecisionFlow struct {
	applyMu   sync.Mutex
	store     DecisionAskStore
	publisher operatordecision.Publisher
	clock     schedule.Clock
	raiser    operatordecision.Raiser
	approvals *Service
	limits    LimitsSource
}

func NewDecisionFlow(store DecisionAskStore, publisher operatordecision.Publisher, clock schedule.Clock) *DecisionFlow {
	if clock == nil {
		clock = schedule.System()
	}
	flow := &DecisionFlow{store: store, publisher: publisher, clock: clock}
	flow.raiser = operatordecision.Raiser{Source: DecisionSource, Kind: DecisionKind, Store: store, Publisher: publisher, Clock: clock, Window: DecisionAskWindow}
	return flow
}

func (f *DecisionFlow) WithApprovalService(approvals *Service) *DecisionFlow {
	f.approvals = approvals
	return f
}

// WithLimits lets the decision show how the charge sits against its budget.
// Without it, or when it fails, the decision still carries the charge itself.
func (f *DecisionFlow) WithLimits(limits LimitsSource) *DecisionFlow {
	f.limits = limits
	return f
}

func decisionAskID(requestID string) string {
	return operatordecision.CorrelationID("treasury-approval", requestID)
}

func (f *DecisionFlow) Raise(ctx context.Context, request Request) error {
	if f == nil || f.store == nil {
		return fmt.Errorf("Treasury decision ask store is unavailable")
	}
	if request.Status != StatusQueued || strings.TrimSpace(request.ID) == "" {
		return fmt.Errorf("queued Treasury approval is required")
	}
	id := decisionAskID(request.ID)
	existing, err := f.store.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("load Treasury decision ask: %w", err)
	}
	if existing != nil && existing.State != operatordecision.PendingStatePending && f.clock.Now().Before(existing.ExpiresAt) {
		return nil
	}
	var limits *Limits
	if f.limits != nil {
		if value, err := f.limits(ctx, request); err == nil {
			limits = &value
		}
	}
	facts := approvalDecisionFacts(request, limits)
	if err := facts.Validate(); err != nil {
		return fmt.Errorf("validate Treasury decision: %w", err)
	}
	ask := operatordecision.PendingAsk{ID: id, Subject: request.RequestingAgent, Extra: map[string]string{
		"approval_id": request.ID, "authorization_id": request.AuthorizationID,
	}}
	if _, err := f.raiser.Raise(ctx, ask, facts); err != nil {
		return fmt.Errorf("raise Treasury approval decision: %w", err)
	}
	return nil
}

// RetryQueued republishes only durable, still-queued approval asks. It is safe
// to call on a timer: the Raiser preserves event identity and Hub idempotency.
func (f *DecisionFlow) RetryQueued(ctx context.Context, queue interface {
	List(context.Context, Status, ...string) ([]Request, error)
},
) error {
	if f == nil || queue == nil {
		return nil
	}
	requests, err := queue.List(ctx, StatusQueued)
	if err != nil {
		return err
	}
	var failures []error
	for _, request := range requests {
		if err := f.Raise(ctx, request); err != nil {
			failures = append(failures, fmt.Errorf("approval %s: %w", request.ID, err))
		}
	}
	return errorsJoin(failures)
}

func errorsJoin(errs []error) error {
	return errors.Join(errs...)
}

func (f *DecisionFlow) Handles(sourceEventType string) bool {
	return sourceEventType == operatordecision.DecisionRequestedEventType(DecisionSource, DecisionKind)
}

// Apply consumes only a Hub-read-back answer for this exact Treasury ask. The
// decision ask itself cannot mutate policy; approval service rechecks the
// authorization, budget, freeze, funding and expiry before changing the hold.
func (f *DecisionFlow) Apply(ctx context.Context, resolution operatordecision.DecisionResolution) (bool, error) {
	if f == nil || f.store == nil || f.approvals == nil {
		return false, nil
	}
	approvalID := strings.TrimSpace(resolution.Correlation["approval_id"])
	if approvalID == "" {
		return false, nil
	}
	// A Hub answer can arrive through the direct callback and the durable
	// Events subscription at the same time. Serialize the read/apply/readback
	// sequence so only one request observes the ask and approval as pending.
	f.applyMu.Lock()
	defer f.applyMu.Unlock()
	// The Hub assigns its own UUID to each ask. The producer's durable record is
	// keyed by our stable correlation ID, so bind the two through the immutable
	// requester event reference captured when this exact ask was raised.
	ask, err := f.store.Get(ctx, decisionAskID(approvalID))
	if err != nil {
		return true, fmt.Errorf("load Treasury approval decision: %w", err)
	}
	if ask == nil || ask.Extra["approval_id"] != approvalID || ask.State != operatordecision.PendingStatePending ||
		strings.TrimSpace(ask.EventID) == "" || resolution.RequesterRef != ask.EventID {
		return true, nil
	}
	request, err := f.approvals.Get(ctx, approvalID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return true, nil
		}
		return true, fmt.Errorf("load correlated Treasury approval: %w", err)
	}
	if resolution.Correlation["authorization_id"] != request.AuthorizationID ||
		resolution.Correlation["book_id"] != request.BookID || resolution.Correlation["mandate_id"] != request.MandateID ||
		resolution.Correlation["requesting_agent"] != request.RequestingAgent || resolution.Correlation["currency"] != request.Currency ||
		resolution.Correlation["counterparty"] != request.Counterparty || resolution.Correlation["amount_minor"] != fmt.Sprint(request.AmountMinor) {
		return true, nil
	}
	if resolution.State != operatordecision.StateAnswered || resolution.ByDefault ||
		!strings.EqualFold(strings.TrimSpace(resolution.AnsweredByIdentityKind), "verified_human") ||
		strings.TrimSpace(resolution.AnsweredBy) == "" || strings.TrimSpace(resolution.AnsweredByIdentitySource) == "" {
		return true, nil
	}
	var status Status
	switch resolution.Answer {
	case "approve":
		status = StatusApproved
	case "decline":
		status = StatusDeclined
	default:
		return true, nil
	}
	if request.Status == StatusQueued {
		if _, err := f.approvals.Resolve(ctx, approvalID, status, resolution.AnsweredBy); err != nil {
			return true, fmt.Errorf("apply verified Treasury approval decision: %w", err)
		}
	} else if request.Status != status || request.ResolverIdentity != resolution.AnsweredBy {
		// Approval resolution and Hub readback are persisted in separate owner
		// stores. If Treasury committed its one-time effect but crashed before
		// saving the Hub ask, an exact event replay may finish the correlation
		// only when the durable queue already records this answer and actor.
		// This path never repeats the authorization effect.
		return true, nil
	}
	resolvedAt := f.clock.Now().UTC()
	if parsed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(resolution.ResolvedAt)); parseErr == nil && !parsed.IsZero() {
		resolvedAt = parsed.UTC()
	}
	ask.State, ask.ResolvedBy, ask.ResolvedAt = resolution.State, resolution.AnsweredBy, &resolvedAt
	if err := f.store.Put(ctx, ask); err != nil {
		return true, fmt.Errorf("persist Treasury decision resolution: %w", err)
	}
	return true, nil
}

// DecisionWebhookModule mounts the shared, signature-checked and Hub-read-back
// verified operator-decision endpoint.
func DecisionWebhookModule(secret string, reader operatordecision.AskReader, flow *DecisionFlow) module.Module {
	return module.Module{Name: "approval-decision-webhook", Mount: func(router *mux.Router) {
		if strings.TrimSpace(secret) == "" || reader == nil || flow == nil {
			return
		}
		router.Handle(DecisionWebhookPath, operatordecision.NewWebhook(operatordecision.WebhookConfig{
			Secret: operatordecision.StaticSecret(secret), Source: DecisionSource, Reader: reader, Effects: []operatordecision.DomainEffect{flow},
		})).Methods("POST")
	}}
}

func NewDecisionAskReader(credential func(context.Context) (string, error), options ...askreader.ReaderOption) operatordecision.AskReader {
	return requireIssuedAskReader{inner: askreader.NewReader(DecisionSource, askreader.CredentialSource(credential), options...)}
}

func NewAuthorityDecisionAskReader() operatordecision.AskReader {
	return NewDecisionAskReader(func(context.Context) (string, error) {
		authority, err := credentialauthority.Default()
		if err != nil {
			return "", err
		}
		identity, err := credentialauthority.ParseIdentity(askreader.CredentialIdentity(DecisionSource))
		if err != nil {
			return "", err
		}
		return authority.Require(identity, askreader.CredentialField)
	})
}

// requireIssuedAskReader prevents the shared webhook's migration fallback from
// applying a signed event body without Hub readback. Money-moving approvals wait
// for Treasury's own ask-reader principal instead of using the unverified path.
type requireIssuedAskReader struct{ inner operatordecision.AskReader }

func (r requireIssuedAskReader) ReadAsk(ctx context.Context, id string) (askreader.Ask, error) {
	ask, err := r.inner.ReadAsk(ctx, id)
	if errors.Is(err, askreader.ErrNotIssued) {
		return askreader.Ask{}, fmt.Errorf("%w: Treasury approval decisions require a treasury-ask-reader credential", askreader.ErrHubUnavailable)
	}
	return ask, err
}

var (
	_ DecisionAsker                 = (*DecisionFlow)(nil)
	_ operatordecision.DomainEffect = (*DecisionFlow)(nil)
)
