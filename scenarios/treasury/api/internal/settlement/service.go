// Package settlement executes authorized charges exactly once.
package settlement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
	"treasury/internal/authorization"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/rail"
	settlementflow "treasury/internal/settlement/flow"
)

var ErrInvalid = errors.New("invalid settlement")

const RetentionWindow = 180 * 24 * time.Hour

type Outcome string

const (
	OutcomeReady   Outcome = "ready"
	OutcomeCalling Outcome = "calling"
	OutcomeSettled Outcome = "settled"
	OutcomeFailed  Outcome = "failed"
	OutcomeUnknown Outcome = "unknown"
)

type Record struct {
	ID, AuthorizationID, MandateID, InstrumentID, Rail, IdempotencyKey string
	AmountMinor                                                        int64
	Currency, Counterparty, ExternalID, ReceiptReference, Basis        string
	Detail                                                             string
	Outcome                                                            Outcome
	OccurredAt, CreatedAt, UpdatedAt, RetainUntil                      time.Time
}

type SettleInput struct {
	ID, AuthorizationID, InstrumentID, IdempotencyKey string
	IdentityToken                                     string
	Attestation                                       *rail.Attestation
}

type RailResult struct {
	ExternalID, ReceiptReference, Basis, Detail string
	OccurredAt                                  time.Time
	FromQuery                                   bool
}

type Authorizations interface {
	Get(context.Context, string) (authorization.Record, error)
	Settle(context.Context, string) (authorization.Record, error)
	Release(context.Context, string) (authorization.Record, error)
}

type Instruments interface {
	ResolveForUse(context.Context, string) (instrument.ScopedCredential, error)
}

type Rails interface {
	Get(string) (rail.Adapter, error)
}

type Service struct {
	repository     Repository
	authorizations Authorizations
	instruments    Instruments
	rails          Rails
	verifier       identity.Verifier
	clock          schedule.Clock
	locks          keyLocks
}

func NewService(repository Repository, authorizations Authorizations, instruments Instruments, rails Rails, verifier identity.Verifier, clock schedule.Clock) *Service {
	return &Service{repository: repository, authorizations: authorizations, instruments: instruments, rails: rails, verifier: verifier, clock: clock}
}

// Settle returns the first durable outcome for an idempotency key. The unique
// claim is committed before the adapter is invoked, so no later caller is ever
// allowed to repeat a partially observable side effect.
func (s *Service) Settle(ctx context.Context, in SettleInput) (Record, error) {
	if err := s.validateDependencies(); err != nil {
		return Record{}, err
	}
	in.ID = strings.TrimSpace(in.ID)
	in.AuthorizationID = strings.TrimSpace(in.AuthorizationID)
	in.InstrumentID = strings.TrimSpace(in.InstrumentID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.ID == "" || in.AuthorizationID == "" || in.InstrumentID == "" || in.IdempotencyKey == "" {
		return Record{}, fmt.Errorf("%w: id, authorization_id, instrument_id, and idempotency_key are required", ErrInvalid)
	}

	unlock := s.locks.lock(in.IdempotencyKey)
	defer unlock()
	claims, err := s.verifier.Verify(ctx, strings.TrimSpace(in.IdentityToken))
	if err != nil {
		return Record{}, fmt.Errorf("%w: live agent identity verification failed: %v", ErrInvalid, err)
	}

	if existing, err := s.repository.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		if err := sameRequest(existing, in); err != nil {
			return Record{}, err
		}
		auth, authErr := s.authorizations.Get(ctx, existing.AuthorizationID)
		if authErr != nil || auth.RequestingAgent != claims.Subject {
			return Record{}, fmt.Errorf("%w: settlement identity does not own the authorization", ErrInvalid)
		}
		return s.repairAuthorization(ctx, existing)
	} else if !errors.Is(err, ErrNotFound) {
		return Record{}, err
	}

	auth, err := s.authorizations.Get(ctx, in.AuthorizationID)
	if err != nil {
		return Record{}, fmt.Errorf("load authorization: %w", err)
	}
	now := s.clock.Now().UTC()
	if auth.Verdict != authorization.VerdictApproved || !now.Before(auth.ExpiresAt) || auth.HoldMinor != auth.AmountMinor {
		return Record{}, fmt.Errorf("%w: authorization must be approved, unexpired, and fully held", ErrInvalid)
	}
	if auth.RequestingAgent != claims.Subject {
		return Record{}, fmt.Errorf("%w: settlement identity does not own the authorization", ErrInvalid)
	}
	scoped, err := s.instruments.ResolveForUse(ctx, in.InstrumentID)
	if err != nil {
		return Record{}, fmt.Errorf("resolve instrument: %w", err)
	}
	if scoped.Instrument.MandateID != auth.MandateID || scoped.Instrument.BookID == "" || scoped.Instrument.CapMinor < auth.AmountMinor || scoped.Instrument.Currency != auth.Currency || scoped.Instrument.Counterparty != auth.Counterparty {
		return Record{}, fmt.Errorf("%w: instrument scope does not cover the authorization", ErrInvalid)
	}
	adapter, err := s.rails.Get(scoped.Instrument.Rail)
	if err != nil {
		return Record{}, fmt.Errorf("load rail: %w", err)
	}
	candidate := Record{
		ID: in.ID, AuthorizationID: auth.ID, MandateID: auth.MandateID, InstrumentID: scoped.Instrument.ID,
		Rail: scoped.Instrument.Rail, IdempotencyKey: in.IdempotencyKey, AmountMinor: auth.AmountMinor,
		Currency: auth.Currency, Counterparty: auth.Counterparty, Outcome: OutcomeReady,
		CreatedAt: now, UpdatedAt: now, RetainUntil: now.Add(RetentionWindow),
	}
	claim, err := s.repository.Claim(ctx, candidate)
	if err != nil {
		return Record{}, err
	}
	if err := sameRequest(claim.Record, in); err != nil {
		return Record{}, err
	}
	if !claim.Claimed {
		return s.repairAuthorization(ctx, claim.Record)
	}

	result, callErr := adapter.Settle(ctx, rail.SettleCommand{
		SettlementID: claim.Record.ID, AuthorizationID: auth.ID, MandateReference: auth.MandateID,
		InstrumentID: scoped.Instrument.ID, IdempotencyKey: in.IdempotencyKey, AmountMinor: auth.AmountMinor,
		Currency: auth.Currency, Counterparty: auth.Counterparty, Credential: scoped.Value, Attestation: in.Attestation,
	})
	outcome, normalized, resultErr := normalizeRailResult(result, false)
	if callErr != nil {
		outcome = OutcomeUnknown
		normalized.Detail = "rail response unavailable: " + callErr.Error()
	} else if resultErr != nil {
		outcome = OutcomeUnknown
		normalized.Detail = "rail response invalid: " + resultErr.Error()
	}
	event := settlementflow.SettlementResponseLost
	switch outcome {
	case OutcomeSettled:
		event = settlementflow.SettlementRailSettled
	case OutcomeFailed:
		event = settlementflow.SettlementRailFailed
	}
	if next, transitionErr := transitionOutcome(OutcomeCalling, event); transitionErr != nil {
		return Record{}, fmt.Errorf("apply settlement transition: %w", transitionErr)
	} else if next != outcome {
		return Record{}, fmt.Errorf("apply settlement transition: expected %s, got %s", outcome, next)
	}
	durableCtx := context.WithoutCancel(ctx)
	completedAt := s.clock.Now().UTC()
	completed, err := s.repository.Complete(durableCtx, claim.Record.ID, outcome, normalized, completedAt.Format(time.RFC3339Nano), completedAt.Add(RetentionWindow).Format(time.RFC3339Nano))
	if err != nil {
		return Record{}, fmt.Errorf("record rail outcome: %w", err)
	}
	return s.repairAuthorization(durableCtx, completed)
}

// ResolveUnknown is deliberately the only operation that can transition an
// unknown record. A retry of Settle only returns the unknown record and never
// invokes the adapter again.
func (s *Service) ResolveUnknown(ctx context.Context, id string) (Record, error) {
	if err := s.validateDependencies(); err != nil {
		return Record{}, err
	}
	id = strings.TrimSpace(id)
	current, err := s.repository.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	unlock := s.locks.lock(current.IdempotencyKey)
	defer unlock()
	current, err = s.repository.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if current.Outcome != OutcomeUnknown {
		return Record{}, fmt.Errorf("%w: only unknown settlements can be queried", ErrInvalid)
	}
	adapter, err := s.rails.Get(current.Rail)
	if err != nil {
		return Record{}, fmt.Errorf("load rail: %w", err)
	}
	result, err := adapter.Query(ctx, rail.Query{SettlementID: current.ID, MandateReference: current.MandateID, ExternalID: current.ExternalID, IdempotencyKey: current.IdempotencyKey})
	if err != nil {
		return current, fmt.Errorf("query rail outcome: %w", err)
	}
	outcome, normalized, resultErr := normalizeRailResult(result, true)
	if resultErr != nil {
		return current, fmt.Errorf("validate rail query result: %w", resultErr)
	}
	if outcome == OutcomeUnknown {
		return current, nil
	}
	event := settlementflow.SettlementQuerySettled
	if outcome == OutcomeFailed {
		event = settlementflow.SettlementQueryFailed
	}
	if next, transitionErr := transitionOutcome(OutcomeUnknown, event); transitionErr != nil {
		return Record{}, fmt.Errorf("apply rail-query transition: %w", transitionErr)
	} else if next != outcome {
		return Record{}, fmt.Errorf("apply rail-query transition: expected %s, got %s", outcome, next)
	}
	durableCtx := context.WithoutCancel(ctx)
	completedAt := s.clock.Now().UTC()
	completed, err := s.repository.Complete(durableCtx, current.ID, outcome, normalized, completedAt.Format(time.RFC3339Nano), completedAt.Add(RetentionWindow).Format(time.RFC3339Nano))
	if err != nil {
		return Record{}, err
	}
	return s.repairAuthorization(durableCtx, completed)
}

func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) validateDependencies() error {
	if s == nil || s.repository == nil || s.authorizations == nil || s.instruments == nil || s.rails == nil || s.verifier == nil || s.clock == nil {
		return fmt.Errorf("%w: repository, authorizations, instruments, rails, identity verifier, and clock are required", ErrInvalid)
	}
	return nil
}

func (s *Service) repairAuthorization(ctx context.Context, value Record) (Record, error) {
	var err error
	switch value.Outcome {
	case OutcomeSettled:
		_, err = s.authorizations.Settle(ctx, value.AuthorizationID)
	case OutcomeFailed:
		_, err = s.authorizations.Release(ctx, value.AuthorizationID)
	}
	if err != nil {
		return Record{}, fmt.Errorf("project settlement to authorization: %w", err)
	}
	return value, nil
}

func sameRequest(value Record, in SettleInput) error {
	if value.ID != in.ID || value.AuthorizationID != in.AuthorizationID || value.InstrumentID != in.InstrumentID {
		return fmt.Errorf("%w: idempotency key was already used for a different settlement", ErrInvalid)
	}
	return nil
}

func normalizeRailResult(value rail.Result, fromQuery bool) (Outcome, RailResult, error) {
	outcome := OutcomeUnknown
	switch value.Outcome {
	case rail.OutcomeSettled:
		outcome = OutcomeSettled
	case rail.OutcomeFailed:
		outcome = OutcomeFailed
	case rail.OutcomeUnknown:
		outcome = OutcomeUnknown
	default:
		return OutcomeUnknown, RailResult{FromQuery: fromQuery}, fmt.Errorf("undeclared outcome %q", value.Outcome)
	}
	normalized := RailResult{ExternalID: strings.TrimSpace(value.ExternalID), ReceiptReference: strings.TrimSpace(value.ReceiptReference), Basis: strings.TrimSpace(value.Basis), Detail: strings.TrimSpace(value.Detail), OccurredAt: value.OccurredAt.UTC(), FromQuery: fromQuery}
	if outcome == OutcomeSettled && (normalized.ExternalID == "" || normalized.ReceiptReference == "" || normalized.Basis == "" || value.OccurredAt.IsZero()) {
		return OutcomeUnknown, normalized, fmt.Errorf("settled result requires external_id, receipt_reference, basis, and occurred_at")
	}
	if outcome == OutcomeFailed && (normalized.Basis == "" || normalized.Detail == "") {
		return OutcomeUnknown, normalized, fmt.Errorf("failed result requires basis and detail")
	}
	return outcome, normalized, nil
}

type keyLocks struct {
	mu    sync.Mutex
	locks map[string]*keyLock
}

type keyLock struct {
	mu   sync.Mutex
	refs int
}

func (k *keyLocks) lock(key string) func() {
	k.mu.Lock()
	if k.locks == nil {
		k.locks = make(map[string]*keyLock)
	}
	entry := k.locks[key]
	if entry == nil {
		entry = &keyLock{}
		k.locks[key] = entry
	}
	entry.refs++
	k.mu.Unlock()
	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		k.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(k.locks, key)
		}
		k.mu.Unlock()
	}
}

func transitionOutcome(current Outcome, event settlementflow.SettlementEvent) (Outcome, error) {
	next, err := settlementflow.TransitionSettlement(settlementflow.SettlementState{Status: settlementflow.SettlementStatus(current)}, event)
	return Outcome(next.Status), err
}
