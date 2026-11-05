// Package settlement executes authorized charges exactly once.
package settlement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"treasury/internal/authorization"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/rail"
	settlementflow "treasury/internal/settlement/flow"

	"github.com/vrooli/api-core/schedule"
)

var (
	ErrInvalid    = errors.New("invalid settlement")
	ErrInProgress = errors.New("settlement call is still in progress; operator reconciliation is required after process interruption")
)

const RetentionWindow = 180 * 24 * time.Hour

const (
	BasisX402PreflightOnly = "x402_preflight_only"
	BasisX402PaymentReady  = "x402_payment_prepared"
)

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

type FreezeReader interface {
	IsFrozen(context.Context, string, string) (bool, budget.FreezeScope, error)
}

type Service struct {
	repository     Repository
	authorizations Authorizations
	instruments    Instruments
	rails          Rails
	verifier       identity.Verifier
	clock          schedule.Clock
	freezes        FreezeReader
	locks          keyLocks
	recoveryVault  recoveryCredentialVault
}

type recoveryCredentialVault interface {
	StoreSettlementRecoveryInput(context.Context, string, string) (string, error)
	ResolveSettlementRecoveryInput(context.Context, string) (string, error)
	DeleteSettlementRecoveryInput(context.Context, string) error
	DeleteUnreferencedSettlementRecoveryInput(context.Context, string) error
}

func NewService(repository Repository, authorizations Authorizations, instruments Instruments, rails Rails, verifier identity.Verifier, clock schedule.Clock, freezes ...FreezeReader) *Service {
	service := &Service{repository: repository, authorizations: authorizations, instruments: instruments, rails: rails, verifier: verifier, clock: clock}
	if vault, ok := instruments.(recoveryCredentialVault); ok {
		service.recoveryVault = vault
	}
	if len(freezes) > 0 {
		service.freezes = freezes[0]
	}
	return service
}

// Settle returns the first durable outcome for an idempotency key. The unique
// claim is committed before the adapter is invoked, so no later caller is ever
// allowed to repeat a partially observable side effect.
func (s *Service) Settle(ctx context.Context, in SettleInput) (Record, error) {
	if in.Attestation != nil {
		return Record{}, fmt.Errorf("%w: manual attestation requires the operator-authenticated settlement surface", ErrInvalid)
	}
	return s.settle(ctx, in, "", false)
}

// SettleOperator binds a manual attestation to an already-authenticated
// operator identity. Callers never choose ActorIdentity, and this path cannot
// execute an automated rail.
func (s *Service) SettleOperator(ctx context.Context, in SettleInput, operatorSubject string) (Record, error) {
	operatorSubject = strings.TrimSpace(operatorSubject)
	if operatorSubject == "" || in.Attestation == nil {
		return Record{}, fmt.Errorf("%w: operator identity and manual attestation are required", ErrInvalid)
	}
	copy := *in.Attestation
	copy.ActorIdentity = operatorSubject
	in.Attestation = &copy
	return s.settle(ctx, in, operatorSubject, true)
}

func (s *Service) settle(ctx context.Context, in SettleInput, verifiedSubject string, operator bool) (Record, error) {
	if err := s.validateDependencies(!operator); err != nil {
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
	if !operator {
		claims, err := s.verifier.Verify(ctx, strings.TrimSpace(in.IdentityToken))
		if err != nil {
			return Record{}, fmt.Errorf("%w: live agent identity verification failed: %v", ErrInvalid, err)
		}
		verifiedSubject = claims.Subject
	}

	if existing, err := s.repository.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		if err := sameRequest(existing, in); err != nil {
			return Record{}, err
		}
		auth, authErr := s.authorizations.Get(ctx, existing.AuthorizationID)
		if authErr != nil || !operator && auth.RequestingAgent != verifiedSubject {
			return Record{}, fmt.Errorf("%w: settlement identity does not own the authorization", ErrInvalid)
		}
		if existing.Outcome == OutcomeCalling {
			return existing, ErrInProgress
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
	if !operator && auth.RequestingAgent != verifiedSubject {
		return Record{}, fmt.Errorf("%w: settlement identity does not own the authorization", ErrInvalid)
	}
	scoped, err := s.instruments.ResolveForUse(ctx, in.InstrumentID)
	if err != nil {
		return Record{}, fmt.Errorf("resolve instrument: %w", err)
	}
	if scoped.Instrument.MandateID != auth.MandateID || scoped.Instrument.BookID == "" || scoped.Instrument.CapMinor < auth.AmountMinor || scoped.Instrument.Currency != auth.Currency || scoped.Instrument.Counterparty != auth.Counterparty {
		return Record{}, fmt.Errorf("%w: instrument scope does not cover the authorization", ErrInvalid)
	}
	if operator && scoped.Instrument.Rail != "manual" {
		return Record{}, fmt.Errorf("%w: operator-attested settlement is restricted to the manual rail", ErrInvalid)
	}
	if s.freezes != nil {
		frozen, scope, freezeErr := s.freezes.IsFrozen(ctx, scoped.Instrument.BookID, auth.BudgetID)
		if freezeErr != nil {
			return Record{}, fmt.Errorf("read kill switch before rail dispatch: %w", freezeErr)
		}
		if frozen {
			if _, releaseErr := s.authorizations.Release(context.WithoutCancel(ctx), auth.ID); releaseErr != nil {
				return Record{}, fmt.Errorf("release authorization stopped by %s freeze: %w", scope, releaseErr)
			}
			return Record{}, fmt.Errorf("%w: %s freeze stopped settlement before rail dispatch", ErrInvalid, scope)
		}
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
	if candidate.Rail == "x402" {
		candidate.Basis = BasisX402PreflightOnly
	}
	claim, err := s.repository.Claim(ctx, candidate)
	if err != nil {
		return Record{}, err
	}
	if err := sameRequest(claim.Record, in); err != nil {
		return Record{}, err
	}
	if !claim.Claimed {
		if claim.Record.Outcome == OutcomeCalling {
			return claim.Record, ErrInProgress
		}
		return s.repairAuthorization(ctx, claim.Record)
	}

	railReply, callErr := adapter.Settle(ctx, rail.SettleCommand{
		SettlementID: claim.Record.ID, AuthorizationID: auth.ID, MandateReference: auth.MandateID,
		InstrumentID: scoped.Instrument.ID, IdempotencyKey: in.IdempotencyKey, AmountMinor: auth.AmountMinor,
		Currency: auth.Currency, Counterparty: auth.Counterparty, Credential: scoped.Value, Attestation: in.Attestation,
		RecordRecoveryInput: func(recordCtx context.Context, digest, payload string) error {
			if s.recoveryVault == nil {
				return errors.New("protected x402 recovery credential vault unavailable")
			}
			reference, storeErr := s.recoveryVault.StoreSettlementRecoveryInput(recordCtx, claim.Record.ID, payload)
			if storeErr != nil {
				return storeErr
			}
			if recordErr := s.repository.RecordX402RecoveryInput(recordCtx, claim.Record.ID, digest, reference); recordErr != nil {
				_ = s.recoveryVault.DeleteSettlementRecoveryInput(context.WithoutCancel(recordCtx), reference)
				return recordErr
			}
			return nil
		},
	})
	outcome, normalized, resultErr := normalizeRailResult(railReply, false)
	recoveryReference := ""
	if candidate.Rail == "x402" {
		if prepared, getErr := s.repository.Get(ctx, claim.Record.ID); getErr == nil && prepared.Basis == BasisX402PaymentReady {
			recoveryReference = prepared.ReceiptReference
			// Use the protected reference from the calling fence, never the payload.
			if outcome == OutcomeUnknown {
				normalized.ExternalID = prepared.ExternalID
				normalized.ReceiptReference = prepared.ReceiptReference
			}
		}
	}
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
	artifacts, err := completionArtifacts(auth, claim.Record, normalized, outcome)
	if err != nil {
		return Record{}, fmt.Errorf("construct immutable settlement evidence: %w", err)
	}
	completed, err := s.repository.Complete(durableCtx, claim.Record.ID, outcome, normalized, completedAt.Format(time.RFC3339Nano), completedAt.Add(RetentionWindow).Format(time.RFC3339Nano), artifacts)
	if err != nil {
		return Record{}, fmt.Errorf("record rail outcome: %w", err)
	}
	completed, err = s.repairAuthorization(durableCtx, completed)
	if err != nil {
		return completed, err
	}
	if candidate.Rail == "x402" && outcome != OutcomeUnknown && s.recoveryVault != nil && recoveryReference != "" {
		if err := s.recoveryVault.DeleteSettlementRecoveryInput(context.WithoutCancel(durableCtx), recoveryReference); err != nil {
			return completed, fmt.Errorf("settlement completed but protected replay credential cleanup failed: %w", err)
		}
	}
	return completed, nil
}

// ResolveUnknown is the operator reconciliation operation for durable unknown
// outcomes and confirmed interrupted calling fences. A retry of Settle never
// invokes the adapter again for either state.
func (s *Service) ResolveUnknown(ctx context.Context, id string) (Record, error) {
	if err := s.validateDependencies(false); err != nil {
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
		if current.Outcome != OutcomeCalling {
			return Record{}, fmt.Errorf("%w: only calling or unknown settlements can be queried", ErrInvalid)
		}
		current, err = s.recoverInterruptedCall(ctx, current)
		if err != nil {
			return Record{}, err
		}
		if current.Outcome != OutcomeUnknown {
			return current, nil
		}
	}
	adapter, err := s.rails.Get(current.Rail)
	if err != nil {
		return Record{}, fmt.Errorf("load rail: %w", err)
	}
	scoped, err := s.instruments.ResolveForUse(ctx, current.InstrumentID)
	if err != nil {
		return current, fmt.Errorf("resolve instrument for rail query: %w", err)
	}
	if scoped.Instrument.MandateID != current.MandateID || scoped.Instrument.Rail != current.Rail || scoped.Instrument.Counterparty != current.Counterparty {
		return current, fmt.Errorf("%w: instrument scope no longer covers the unknown settlement", ErrInvalid)
	}
	recoveryInput := current.ReceiptReference
	if current.Rail == "x402" {
		if current.Basis != BasisX402PaymentReady || s.recoveryVault == nil {
			return current, fmt.Errorf("%w: protected x402 replay authorization is unavailable", ErrInvalid)
		}
		recoveryInput, err = s.recoveryVault.ResolveSettlementRecoveryInput(ctx, current.ReceiptReference)
		if err != nil {
			return current, fmt.Errorf("resolve protected x402 replay authorization: %w", err)
		}
	}
	railReply, err := adapter.QueryOutcome(ctx, rail.Query{SettlementID: current.ID, MandateReference: current.MandateID, InstrumentID: current.InstrumentID, ExternalID: current.ExternalID, ReceiptReference: recoveryInput, IdempotencyKey: current.IdempotencyKey, Counterparty: current.Counterparty, Credential: scoped.Value})
	if err != nil {
		return current, fmt.Errorf("query rail outcome: %w", err)
	}
	outcome, normalized, resultErr := normalizeRailResult(railReply, true)
	if current.Rail == "x402" && outcome == OutcomeUnknown {
		// QueryOutcome necessarily received the signed input in memory. Keep the
		// protected reference in SQLite if the result is still inconclusive.
		normalized.ExternalID = current.ExternalID
		normalized.ReceiptReference = current.ReceiptReference
	}
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
	auth, err := s.authorizations.Get(durableCtx, current.AuthorizationID)
	if err != nil {
		return Record{}, fmt.Errorf("load authorization for immutable evidence: %w", err)
	}
	artifacts, err := completionArtifacts(auth, current, normalized, outcome)
	if err != nil {
		return Record{}, fmt.Errorf("construct immutable settlement evidence: %w", err)
	}
	completed, err := s.repository.Complete(durableCtx, current.ID, outcome, normalized, completedAt.Format(time.RFC3339Nano), completedAt.Add(RetentionWindow).Format(time.RFC3339Nano), artifacts)
	if err != nil {
		return Record{}, err
	}
	completed, err = s.repairAuthorization(durableCtx, completed)
	if err != nil {
		return completed, err
	}
	if current.Rail == "x402" && s.recoveryVault != nil {
		if err := s.recoveryVault.DeleteSettlementRecoveryInput(context.WithoutCancel(durableCtx), current.ReceiptReference); err != nil {
			return completed, fmt.Errorf("settlement resolved but protected replay credential cleanup failed: %w", err)
		}
	}
	return completed, nil
}

func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) validateDependencies(requireVerifier bool) error {
	if s == nil || s.repository == nil || s.authorizations == nil || s.instruments == nil || s.rails == nil || requireVerifier && s.verifier == nil || s.clock == nil {
		return fmt.Errorf("%w: repository, authorizations, instruments, rails, clock, and any required identity verifier are required", ErrInvalid)
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

// recoverInterruptedCall turns an orphaned durable call fence into an unknown
// outcome, or a versioned x402 preflight-only fence into a definite failure.
// The per-key process lock excludes another caller in this API process. The
// operator confirmation is still required because a separate Treasury process
// could share the database. Treasury never guesses that no side effect occurred
// or invokes Settle again; the hold stays in place until reconciliation.
func (s *Service) recoverInterruptedCall(ctx context.Context, value Record) (Record, error) {
	if value.Outcome != OutcomeCalling {
		return s.repairAuthorization(ctx, value)
	}
	now := s.clock.Now().UTC()
	outcome := OutcomeUnknown
	result := RailResult{
		ExternalID:       value.ExternalID,
		ReceiptReference: value.ReceiptReference,
		Basis:            "interrupted_execution",
		Detail:           "Treasury recovered a durable rail-call fence without a conclusive response; the external side effect may have occurred and was not retried",
	}
	if value.Rail == "x402" && value.Basis == BasisX402PaymentReady {
		result.Basis = BasisX402PaymentReady
	}
	if value.Rail == "x402" && value.Basis == BasisX402PreflightOnly && value.ExternalID == "" && value.ReceiptReference == "" {
		// New x402 calls persist this phase before adapter work. The paid request
		// is strictly after signed-payload persistence, so this versioned state
		// proves a crash here happened before any payment dispatch. Empty-basis
		// legacy rows remain unknown because their dispatch cutpoint is unclear.
		outcome = OutcomeFailed
		result.Basis = "x402_dispatch_not_started"
		result.Detail = "Treasury recovered an x402 call that had not durably prepared a signed payment; no paid request was dispatched"
		if s.recoveryVault != nil {
			if cleanupErr := s.recoveryVault.DeleteUnreferencedSettlementRecoveryInput(context.WithoutCancel(ctx), value.ID); cleanupErr != nil {
				return Record{}, fmt.Errorf("delete unreferenced x402 recovery credential: %w", cleanupErr)
			}
		}
	}
	var artifacts []CompletionArtifacts
	if outcome == OutcomeFailed {
		auth, getErr := s.authorizations.Get(context.WithoutCancel(ctx), value.AuthorizationID)
		if getErr != nil {
			return Record{}, fmt.Errorf("load authorization for pre-dispatch recovery: %w", getErr)
		}
		completedArtifacts, artifactErr := completionArtifacts(auth, value, result, outcome)
		if artifactErr != nil {
			return Record{}, fmt.Errorf("construct pre-dispatch recovery evidence: %w", artifactErr)
		}
		artifacts = append(artifacts, completedArtifacts)
	}
	recovered, err := s.repository.Complete(context.WithoutCancel(ctx), value.ID, outcome, result, now.Format(time.RFC3339Nano), now.Add(RetentionWindow).Format(time.RFC3339Nano), artifacts...)
	if err != nil {
		return Record{}, fmt.Errorf("recover interrupted settlement as unknown: %w", err)
	}
	return s.repairAuthorization(ctx, recovered)
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

func completionArtifacts(auth authorization.Record, value Record, result RailResult, outcome Outcome) (CompletionArtifacts, error) {
	request, err := json.Marshal(map[string]any{
		"authorization_id": auth.ID, "mandate_id": auth.MandateID, "settlement_id": value.ID,
		"instrument_id": value.InstrumentID, "idempotency_key": value.IdempotencyKey,
		"agent_subject": auth.RequestingAgent, "amount_minor": value.AmountMinor,
		"currency": value.Currency, "counterparty": value.Counterparty,
	})
	if err != nil {
		return CompletionArtifacts{}, err
	}
	railResponse, err := json.Marshal(map[string]any{
		"outcome": outcome, "external_id": result.ExternalID, "basis": result.Basis,
		"detail": result.Detail, "occurred_at": formatOptionalTime(result.OccurredAt), "from_query": result.FromQuery,
	})
	if err != nil {
		return CompletionArtifacts{}, err
	}
	receipt, err := json.Marshal(map[string]any{"reference": result.ReceiptReference, "external_id": result.ExternalID})
	if err != nil {
		return CompletionArtifacts{}, err
	}
	return CompletionArtifacts{AgentSubject: auth.RequestingAgent, RequestJSON: string(request), RailResponseJSON: string(railResponse), ReceiptJSON: string(receipt)}, nil
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
