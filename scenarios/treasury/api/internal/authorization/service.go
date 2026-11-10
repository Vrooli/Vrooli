// Package authorization evaluates proposed charges against stored authority.
package authorization

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/cli-core/cliutil"

	authorizationflow "treasury/internal/authorization/flow"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/mandate"
)

var ErrInvalid = errors.New("invalid authorization")

type Verdict string

const (
	VerdictRefused  Verdict = "refused"
	VerdictPending  Verdict = "pending"
	VerdictApproved Verdict = "approved"
	VerdictReleased Verdict = "released"
	VerdictSettled  Verdict = "settled"
)

type Record struct {
	ID                       string
	IdempotencyKey           string
	BookID                   string
	MandateID                string
	BudgetID                 string
	RequestingAgent          string
	AmountMinor              int64
	Currency                 string
	Counterparty             string
	SettlementID             string
	InstrumentID             string
	SettlementIdempotencyKey string
	Verdict                  Verdict
	ViolatedConstraint       string
	Remediation              string
	HoldMinor                int64
	CreatedAt                time.Time
	ExpiresAt                time.Time
}

type ProposeInput struct {
	ID                       string
	IdempotencyKey           string
	MandateID                string
	IdentityToken            string
	OperationToken           string
	AmountMinor              int64
	Currency                 string
	Counterparty             string
	SettlementID             string
	InstrumentID             string
	SettlementIdempotencyKey string
}

type Mandates interface {
	RequireLive(context.Context, string) (mandate.Mandate, error)
}

type Budgets interface {
	Get(context.Context, string) (budget.Budget, error)
}

type FreezeReader interface {
	IsFrozen(context.Context, string, string) (bool, budget.FreezeScope, error)
}

type DecisionEvidence struct {
	ID                 string
	AuthorizationID    string
	MandateID          string
	AgentSubject       string
	Verdict            Verdict
	ViolatedConstraint string
	Detail             string
	IdempotencyKey     string
	AmountMinor        int64
	Currency           string
	Counterparty       string
	CreatedAt          time.Time
}

type EvidenceRecorder interface {
	RecordDecision(context.Context, DecisionEvidence) error
}

type ApprovalAdmission struct {
	ID              string
	AuthorizationID string
	BookID          string
	MandateID       string
	RequestingAgent string
	AmountMinor     int64
	Currency        string
	Counterparty    string
	ExpiresAt       time.Time
}

type ApprovalQueue interface {
	Admit(context.Context, ApprovalAdmission) error
}

type Service struct {
	repository Repository
	verifier   identity.Verifier
	mandates   Mandates
	budgets    Budgets
	evidence   EvidenceRecorder
	approvals  ApprovalQueue
	clock      schedule.Clock
	mu         sync.Mutex
}

func NewService(repository Repository, verifier identity.Verifier, mandates Mandates, budgets Budgets, evidence EvidenceRecorder, clock schedule.Clock, approvals ...ApprovalQueue) *Service {
	service := &Service{repository: repository, verifier: verifier, mandates: mandates, budgets: budgets, evidence: evidence, clock: clock}
	if len(approvals) > 0 {
		service.approvals = approvals[0]
	}
	return service
}

// Propose verifies identity live, recomputes policy exclusively from stored
// state, and serializes headroom reservation for this single-instance SQLite
// scenario. Caller-supplied verdicts or allowance fields do not exist.
func (s *Service) Propose(ctx context.Context, in ProposeInput) (Record, error) {
	if s.repository == nil || s.verifier == nil || s.mandates == nil || s.budgets == nil || s.evidence == nil || s.clock == nil {
		return Record{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	in.ID = strings.TrimSpace(in.ID)
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	in.MandateID = strings.TrimSpace(in.MandateID)
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	in.Counterparty = strings.ToLower(strings.TrimSpace(in.Counterparty))
	in.SettlementID = strings.TrimSpace(in.SettlementID)
	in.InstrumentID = strings.TrimSpace(in.InstrumentID)
	in.SettlementIdempotencyKey = strings.TrimSpace(in.SettlementIdempotencyKey)
	if in.ID == "" || in.IdempotencyKey == "" || in.MandateID == "" || in.AmountMinor <= 0 || in.Currency == "" || in.Counterparty == "" || in.SettlementID == "" || in.InstrumentID == "" || in.SettlementIdempotencyKey == "" {
		return Record{}, fmt.Errorf("%w: authorization fields and complete settlement intent are required", ErrInvalid)
	}

	now := s.clock.Now().UTC()
	claims, err := verifySpendIdentity(ctx, s.verifier, in.IdentityToken, in.OperationToken)
	if err != nil {
		refusal := baseRecord(in, now)
		refusal.Verdict, _ = initialVerdict(authorizationflow.AuthorizationRefuse)
		refusal.ViolatedConstraint = "identity"
		refusal.Remediation = "provide an active agent-manager identity token and retry while the identity authority is reachable"
		if evidenceErr := s.recordEvidence(ctx, refusal, "identity verification failed: "+err.Error()); evidenceErr != nil {
			return Record{}, fmt.Errorf("record identity refusal: %w", evidenceErr)
		}
		return refusal, nil
	}

	// The mutex covers the read-derived headroom calculation and insert. SQLite
	// serializes writers; this additionally prevents two goroutines in the one
	// supported process from observing the same pre-insert snapshot.
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, err := s.repository.GetByIdempotencyKey(ctx, in.IdempotencyKey); err == nil {
		if existing.ID != in.ID || existing.MandateID != in.MandateID || existing.AmountMinor != in.AmountMinor || existing.Currency != in.Currency || existing.Counterparty != in.Counterparty || existing.SettlementID != in.SettlementID || existing.InstrumentID != in.InstrumentID || existing.SettlementIdempotencyKey != in.SettlementIdempotencyKey || existing.RequestingAgent != claims.Subject {
			return Record{}, fmt.Errorf("%w: idempotency key was already used for a different charge", ErrInvalid)
		}
		if err := identity.Authorize(claims, "treasury.spend", treasurySpendIntentTarget(existing), now); err != nil {
			return s.deny(ctx, in, claims.Subject, now, "operation_grant", "obtain a live operation grant covering this authorization")
		}
		if existing.Verdict == VerdictPending {
			if err := s.ensureApproval(ctx, existing); err != nil {
				_, _ = s.repository.Release(ctx, existing.ID)
				return Record{}, fmt.Errorf("repair approval admission and release failed hold: %w", err)
			}
		}
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Record{}, err
	}

	grant, err := s.mandates.RequireLive(ctx, in.MandateID)
	if err != nil {
		return s.deny(ctx, in, claims.Subject, now, "mandate_live", "issue or select a live, unexpired mandate")
	}
	if err := identity.Authorize(claims, "treasury.spend", cliutil.TreasurySpendIntentAuthorizationTarget(treasurySpendIntent(in)), now); err != nil {
		return s.deny(ctx, in, claims.Subject, now, "operation_grant", "obtain a live operation grant covering this authorization")
	}
	if grant.AgentSubject == "" || grant.AgentSubject != claims.Subject || grant.AgentRunID != claims.RunID {
		return s.deny(ctx, in, claims.Subject, now, "mandate_delegate", "use the mandate issued to this agent run")
	}
	policy, err := s.budgets.Get(ctx, grant.BudgetID)
	if err != nil {
		return Record{}, fmt.Errorf("load mandate budget: %w", err)
	}
	if grant.BookID != policy.BookID {
		return s.deny(ctx, in, claims.Subject, now, "book_scope", "use a mandate backed by a budget in the same book")
	}
	if in.Currency != grant.Currency || in.Currency != policy.Currency {
		return s.deny(ctx, in, claims.Subject, now, "currency", "submit the charge in the mandate and budget currency")
	}
	if in.AmountMinor > grant.CapMinor {
		return s.deny(ctx, in, claims.Subject, now, "per_transaction_cap", "reduce the amount or ask the operator for a larger scoped grant")
	}
	if !policy.AllowsCounterparty(in.Counterparty) || !grant.AllowsCounterparty(in.Counterparty) {
		return s.deny(ctx, in, claims.Subject, now, "counterparty_scope", "use an allowed counterparty or ask the operator to issue a different grant")
	}
	if freezes, ok := s.budgets.(FreezeReader); ok {
		frozen, scope, freezeErr := freezes.IsFrozen(ctx, grant.BookID, policy.ID)
		if freezeErr != nil {
			return Record{}, fmt.Errorf("read kill switch: %w", freezeErr)
		}
		if frozen {
			return s.deny(ctx, in, claims.Subject, now, string(scope)+"_frozen", "ask the operator to release the "+string(scope)+" kill switch")
		}
	} else if policy.Frozen {
		return s.deny(ctx, in, claims.Subject, now, "budget_frozen", "ask the operator to unfreeze the budget")
	}
	periodStart := periodBoundary(now, policy.Period)
	usage, err := s.repository.Usage(ctx, policy.ID, grant.ID, periodStart, now)
	if err != nil {
		return Record{}, fmt.Errorf("compute headroom: %w", err)
	}
	if funded, ok := s.budgets.(budget.FundingReader); ok {
		funding, fundingErr := funded.Funding(ctx, policy.ID)
		if fundingErr != nil {
			return Record{}, fmt.Errorf("read accepted receipt funding: %w", fundingErr)
		}
		if funding.Required && (funding.CreditedMinor <= usage.BudgetTotalMinor || in.AmountMinor > funding.CreditedMinor-usage.BudgetTotalMinor) {
			return s.deny(ctx, in, claims.Subject, now, "receipt_funding", "fund the budget and wait for Money Ledger to accept the receipt")
		}
	}
	switch {
	case in.AmountMinor > policy.TotalCapMinor-usage.BudgetTotalMinor:
		return s.deny(ctx, in, claims.Subject, now, "total_headroom", "reduce the amount or ask the operator to increase the total cap")
	case in.AmountMinor > policy.PeriodicCapMinor-usage.BudgetPeriodMinor:
		return s.deny(ctx, in, claims.Subject, now, "periodic_headroom", "reduce the amount or wait for the next budget period")
	case in.AmountMinor > grant.CapMinor-usage.MandateTotalMinor:
		return s.deny(ctx, in, claims.Subject, now, "mandate_headroom", "reduce the amount or ask the operator for a new mandate")
	}

	record := baseRecord(in, now)
	record.BookID = grant.BookID
	record.BudgetID = policy.ID
	record.RequestingAgent = claims.Subject
	record.HoldMinor = in.AmountMinor
	record.ExpiresAt = minTime(grant.ExpiresAt, now.Add(15*time.Minute))
	if policy.RequiresApproval || in.AmountMinor > policy.PerTransactionCapMinor {
		record.Verdict, err = initialVerdict(authorizationflow.AuthorizationRequireApproval)
		if in.AmountMinor > policy.PerTransactionCapMinor {
			record.ViolatedConstraint = "per_transaction_cap"
			record.Remediation = "operator approval is required for this exact charge above the automatic transaction cap"
		}
	} else {
		record.Verdict, err = initialVerdict(authorizationflow.AuthorizationApprove)
	}
	if err != nil {
		return Record{}, err
	}
	created, err := s.repository.Create(ctx, record)
	if err != nil {
		if errors.Is(err, ErrFrozen) {
			return s.deny(ctx, in, claims.Subject, now, "freeze_gate", "the kill switch engaged before this authorization was admitted")
		}
		return Record{}, err
	}
	if created.Verdict == VerdictPending {
		if err := s.ensureApproval(ctx, created); err != nil {
			_, _ = s.repository.Release(ctx, created.ID)
			return Record{}, fmt.Errorf("admit approval and release failed hold: %w", err)
		}
	}
	if err := s.recordEvidence(ctx, created, "authorization evaluated from stored policy"); err != nil {
		return Record{}, fmt.Errorf("record authorization evidence: %w", err)
	}
	return created, nil
}

// verifySpendIdentity keeps the ordinary run identity and the approved,
// operation-specific consent as a pair at the Treasury authorization
// boundary. A grant token is never accepted as a replacement for the run
// identity, and a caller that supplies an operation token cannot fall back to
// verifying only its baseline token.
func verifySpendIdentity(ctx context.Context, verifier identity.Verifier, baseline, operation string) (identity.Claims, error) {
	baseline = strings.TrimSpace(baseline)
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return verifier.Verify(ctx, baseline)
	}
	pairVerifier, ok := verifier.(identity.PairVerifier)
	if !ok {
		return identity.Claims{}, fmt.Errorf("%w: paired operation verification is unavailable", identity.ErrUnverifiable)
	}
	if baseline == "" || baseline == operation {
		return identity.Claims{}, fmt.Errorf("%w: identity token pair is incomplete", identity.ErrUnverifiable)
	}
	return pairVerifier.VerifyPair(ctx, baseline, operation)
}

func (s *Service) ensureApproval(ctx context.Context, record Record) error {
	if s.approvals == nil {
		return fmt.Errorf("%w: approval queue is required for gated spend", ErrInvalid)
	}
	return s.approvals.Admit(ctx, ApprovalAdmission{
		ID:              record.ID + ":approval",
		AuthorizationID: record.ID,
		BookID:          record.BookID,
		MandateID:       record.MandateID,
		RequestingAgent: record.RequestingAgent,
		AmountMinor:     record.AmountMinor,
		Currency:        record.Currency,
		Counterparty:    record.Counterparty,
		ExpiresAt:       record.ExpiresAt,
	})
}

func (s *Service) Get(ctx context.Context, id string) (Record, error) {
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

func (s *Service) Release(ctx context.Context, id string) (Record, error) {
	current, err := s.repository.Get(ctx, strings.TrimSpace(id))
	if err != nil {
		return Record{}, err
	}
	if _, err := authorizationflow.TransitionAuthorization(authorizationflow.AuthorizationState{Status: authorizationflow.AuthorizationStatus(current.Verdict)}, authorizationflow.AuthorizationRelease); err != nil {
		return Record{}, err
	}
	return s.repository.Release(ctx, strings.TrimSpace(id))
}

func (s *Service) deny(ctx context.Context, in ProposeInput, subject string, now time.Time, constraint, remediation string) (Record, error) {
	record := baseRecord(in, now)
	record.RequestingAgent = subject
	record.Verdict, _ = initialVerdict(authorizationflow.AuthorizationRefuse)
	record.ViolatedConstraint = constraint
	record.Remediation = remediation
	if err := s.recordEvidence(ctx, record, "policy denied the proposed charge"); err != nil {
		return Record{}, fmt.Errorf("record denial evidence: %w", err)
	}
	return record, nil
}

func (s *Service) recordEvidence(ctx context.Context, record Record, detail string) error {
	return s.evidence.RecordDecision(ctx, DecisionEvidence{ID: record.ID + ":decision", AuthorizationID: record.ID, MandateID: record.MandateID, AgentSubject: record.RequestingAgent, Verdict: record.Verdict, ViolatedConstraint: record.ViolatedConstraint, Detail: detail, IdempotencyKey: record.IdempotencyKey, AmountMinor: record.AmountMinor, Currency: record.Currency, Counterparty: record.Counterparty, CreatedAt: record.CreatedAt})
}

func baseRecord(in ProposeInput, now time.Time) Record {
	return Record{
		ID: in.ID, IdempotencyKey: in.IdempotencyKey, MandateID: in.MandateID,
		AmountMinor: in.AmountMinor, Currency: in.Currency, Counterparty: in.Counterparty,
		SettlementID: in.SettlementID, InstrumentID: in.InstrumentID, SettlementIdempotencyKey: in.SettlementIdempotencyKey,
		CreatedAt: now, ExpiresAt: now,
	}
}

func treasurySpendIntent(in ProposeInput) cliutil.TreasurySpendIntent {
	return cliutil.TreasurySpendIntent{
		AuthorizationID: in.ID, MandateID: in.MandateID, AmountMinor: in.AmountMinor,
		Currency: in.Currency, Counterparty: in.Counterparty, AuthorizationIdempotencyKey: in.IdempotencyKey,
		SettlementID: in.SettlementID, InstrumentID: in.InstrumentID, SettlementIdempotencyKey: in.SettlementIdempotencyKey,
	}
}

func treasurySpendIntentTarget(record Record) string {
	return cliutil.TreasurySpendIntentAuthorizationTarget(cliutil.TreasurySpendIntent{
		AuthorizationID: record.ID, MandateID: record.MandateID, AmountMinor: record.AmountMinor,
		Currency: record.Currency, Counterparty: record.Counterparty, AuthorizationIdempotencyKey: record.IdempotencyKey,
		SettlementID: record.SettlementID, InstrumentID: record.InstrumentID, SettlementIdempotencyKey: record.SettlementIdempotencyKey,
	})
}

func periodBoundary(now time.Time, period time.Duration) time.Time {
	return time.Unix(0, now.UnixNano()-(now.UnixNano()%period.Nanoseconds())).UTC()
}

func minTime(left, right time.Time) time.Time {
	if left.Before(right) {
		return left
	}
	return right
}

func initialVerdict(event authorizationflow.AuthorizationEvent) (Verdict, error) {
	next, err := authorizationflow.TransitionAuthorization(authorizationflow.InitialAuthorizationState(), event)
	return Verdict(next.Status), err
}
