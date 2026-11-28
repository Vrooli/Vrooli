package redemption

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type State string

const (
	StatePendingApproval State = "pending_approval"
	StateSettled         State = "settled"
	StateDenied          State = "denied"
)

type ReservationState string

const (
	ReservationStateActive   ReservationState = "active"
	ReservationStateSettled  ReservationState = "settled"
	ReservationStateReleased ReservationState = "released"
)

type ApprovalPosture string

const (
	ApprovalPostureImmediate        ApprovalPosture = "immediate"
	ApprovalPostureRequiresApproval ApprovalPosture = "requires_approval"
)

type Redemption struct {
	ID              string
	HolderID        string
	CatalogEntryID  string
	TokenTypeID     string
	GrantID         string
	Amount          int64
	IdempotencyKey  string
	State           State
	DeciderSubject  string
	DecisionReason  string
	RequestedAt     time.Time
	DecidedAt       *time.Time
	SettledAt       *time.Time
	ApprovalPosture ApprovalPosture
}

type Reservation struct {
	ID           string
	RedemptionID string
	HolderID     string
	TokenTypeID  string
	Amount       int64
	State        ReservationState
	CreatedAt    time.Time
	ReleasedAt   *time.Time
}

type CatalogEntry struct {
	ID              string
	TokenTypeID     string
	CostAmount      int64
	ApprovalPosture ApprovalPosture
}

type RequestInput struct {
	AuthenticatedSubject string
	CatalogEntryID       string
	GrantID              string
	IdempotencyKey       string
	Evidence             []string
}

type DecisionInput struct {
	RedemptionID   string
	DeciderSubject string
	Reason         string
	IdempotencyKey string
}

var (
	ErrRedemptionNotFound  = errors.New("redemption not found")
	ErrRedemptionConflict  = errors.New("redemption state conflict")
	ErrInsufficientBalance = errors.New("insufficient available balance")
	ErrGrantRefused        = errors.New("grant refused redemption")
	ErrCatalogChanged      = errors.New("catalog entry changed during redemption")
)

type InvalidRedemptionError struct{ Reason string }

func (e *InvalidRedemptionError) Error() string { return "invalid redemption: " + e.Reason }

type Holder struct{ ID string }

type HolderReader interface {
	GetBySubject(context.Context, string) (Holder, error)
}

type HolderReaderFunc func(context.Context, string) (Holder, error)

func (f HolderReaderFunc) GetBySubject(ctx context.Context, subject string) (Holder, error) {
	return f(ctx, subject)
}

type CatalogReader interface {
	RequireAvailable(context.Context, string) (CatalogEntry, error)
}

type CatalogReaderFunc func(context.Context, string) (CatalogEntry, error)

func (f CatalogReaderFunc) RequireAvailable(ctx context.Context, id string) (CatalogEntry, error) {
	return f(ctx, id)
}

type GrantEvaluation struct {
	Allowed bool
	Reason  string
}

type GrantEvaluator interface {
	Evaluate(context.Context, string, string, []string, int64, int64, time.Time) (GrantEvaluation, error)
}

type GrantEvaluatorFunc func(context.Context, string, string, []string, int64, int64, time.Time) (GrantEvaluation, error)

func (f GrantEvaluatorFunc) Evaluate(ctx context.Context, grantID, scope string, evidence []string, available, requested int64, now time.Time) (GrantEvaluation, error) {
	return f(ctx, grantID, scope, evidence, available, requested, now)
}

type Relay interface {
	Pending(context.Context, Redemption) error
}

type Service interface {
	Request(context.Context, RequestInput) (Redemption, error)
	ListForSubject(context.Context, string) ([]Redemption, error)
	ListPending(context.Context) ([]Redemption, error)
	Approve(context.Context, DecisionInput) (Redemption, error)
	Deny(context.Context, DecisionInput) (Redemption, error)
}

func (s *service) ListForSubject(ctx context.Context, authenticatedSubject string) ([]Redemption, error) {
	authenticatedSubject = strings.TrimSpace(authenticatedSubject)
	if authenticatedSubject == "" {
		return nil, &InvalidRedemptionError{Reason: "authenticated subject is required"}
	}
	holder, err := s.holders.GetBySubject(ctx, authenticatedSubject)
	if err != nil {
		return nil, err
	}
	return s.repository.ListForHolder(ctx, holder.ID)
}

type service struct {
	repository Repository
	holders    HolderReader
	catalog    CatalogReader
	grants     GrantEvaluator
	relay      Relay
	clock      schedule.Clock
}

func NewService(repository Repository, holders HolderReader, catalog CatalogReader, grants GrantEvaluator, relay Relay, clock schedule.Clock) Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &service{repository: repository, holders: holders, catalog: catalog, grants: grants, relay: relay, clock: clock}
}

func (s *service) Request(ctx context.Context, input RequestInput) (Redemption, error) {
	normalizeRequest(&input)
	if input.AuthenticatedSubject == "" || input.CatalogEntryID == "" || input.GrantID == "" || input.IdempotencyKey == "" {
		return Redemption{}, &InvalidRedemptionError{Reason: "authenticated subject, catalog entry id, grant id, and idempotency key are required"}
	}
	if existing, found, err := s.repository.FindByIdempotency(ctx, input.IdempotencyKey); err != nil {
		return Redemption{}, err
	} else if found {
		return existing, nil
	}
	holder, err := s.holders.GetBySubject(ctx, input.AuthenticatedSubject)
	if err != nil {
		return Redemption{}, err
	}
	entry, err := s.catalog.RequireAvailable(ctx, input.CatalogEntryID)
	if err != nil {
		return Redemption{}, err
	}
	available, err := s.repository.AvailableBalance(ctx, holder.ID, entry.TokenTypeID)
	if err != nil {
		return Redemption{}, err
	}
	now := s.clock.Now().UTC()
	decision, err := s.grants.Evaluate(ctx, input.GrantID, entry.ID, input.Evidence, available, entry.CostAmount, now)
	if err != nil {
		return Redemption{}, err
	}
	if !decision.Allowed {
		return Redemption{}, fmt.Errorf("%w: %s", ErrGrantRefused, decision.Reason)
	}
	redemption := Redemption{
		ID: uuid.NewString(), HolderID: holder.ID, CatalogEntryID: entry.ID,
		TokenTypeID: entry.TokenTypeID, GrantID: input.GrantID, Amount: entry.CostAmount,
		IdempotencyKey: input.IdempotencyKey, RequestedAt: now, ApprovalPosture: entry.ApprovalPosture,
	}
	created, err := s.repository.Request(ctx, redemption, entry)
	if err != nil {
		return Redemption{}, err
	}
	if created.State == StatePendingApproval && s.relay != nil {
		// The queue is authoritative. Relay failure cannot block or roll it back.
		_ = s.relay.Pending(ctx, created)
	}
	return created, nil
}

func (s *service) ListPending(ctx context.Context) ([]Redemption, error) {
	return s.repository.ListPending(ctx)
}

func (s *service) Approve(ctx context.Context, input DecisionInput) (Redemption, error) {
	if err := normalizeDecision(&input); err != nil {
		return Redemption{}, err
	}
	return s.repository.Approve(ctx, input, s.clock.Now().UTC())
}

func (s *service) Deny(ctx context.Context, input DecisionInput) (Redemption, error) {
	if err := normalizeDecision(&input); err != nil {
		return Redemption{}, err
	}
	return s.repository.Deny(ctx, input, s.clock.Now().UTC())
}

func normalizeRequest(input *RequestInput) {
	input.AuthenticatedSubject = strings.TrimSpace(input.AuthenticatedSubject)
	input.CatalogEntryID = strings.TrimSpace(input.CatalogEntryID)
	input.GrantID = strings.TrimSpace(input.GrantID)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	for index := range input.Evidence {
		input.Evidence[index] = strings.TrimSpace(input.Evidence[index])
	}
}

func normalizeDecision(input *DecisionInput) error {
	input.RedemptionID = strings.TrimSpace(input.RedemptionID)
	input.DeciderSubject = strings.TrimSpace(input.DeciderSubject)
	input.Reason = strings.TrimSpace(input.Reason)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	if input.RedemptionID == "" || input.DeciderSubject == "" || input.Reason == "" || input.IdempotencyKey == "" {
		return &InvalidRedemptionError{Reason: "redemption id, decider subject, reason, and idempotency key are required"}
	}
	return nil
}
