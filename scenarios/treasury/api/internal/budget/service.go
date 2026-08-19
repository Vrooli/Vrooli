// Package budget owns caps, counterparty scope, gating, freezes, and headroom.
package budget

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

var ErrInvalid = errors.New("invalid budget")

type Budget struct {
	ID                     string
	BookID                 string
	Currency               string
	TotalCapMinor          int64
	PeriodicCapMinor       int64
	PerTransactionCapMinor int64
	Period                 time.Duration
	AllowedCounterparties  []string
	DeniedCounterparties   []string
	RequiresApproval       bool
	Frozen                 bool
	CreatedAt              time.Time
}

// Usage is the derived spend admitted by Treasury's authorization records.
// It deliberately contains no Money Ledger position or mutable balance.
type Usage struct {
	TotalMinor  int64
	PeriodMinor int64
}

type UsageReader interface {
	BudgetUsage(context.Context, string, time.Time, time.Time) (Usage, error)
}

type Headroom struct {
	BudgetID, BookID, Currency                              string
	TotalCapMinor, TotalUsedMinor, TotalRemainingMinor      int64
	PeriodicCapMinor, PeriodUsedMinor, PeriodRemainingMinor int64
	PerTransactionCapMinor, AvailableMinor                  int64
	PeriodStartedAt, ComputedAt                             time.Time
}

type FreezeScope string

const (
	FreezeScopeBudget   FreezeScope = "budget"
	FreezeScopeBook     FreezeScope = "book"
	FreezeScopeScenario FreezeScope = "scenario"
)

type FreezeControl struct {
	Scope     FreezeScope
	ScopeID   string
	Frozen    bool
	UpdatedAt time.Time
}

type FreezeRepository interface {
	SetFreezeControl(context.Context, FreezeControl) (FreezeControl, error)
	GetFreezeControl(context.Context, FreezeScope, string) (FreezeControl, error)
}

type Service interface {
	Create(context.Context, Budget) (Budget, error)
	Get(context.Context, string) (Budget, error)
	SetCaps(context.Context, Budget) (Budget, error)
	SetGating(context.Context, string, bool) (Budget, error)
	SetFrozen(context.Context, string, bool) (Budget, error)
	SetScopeFrozen(context.Context, FreezeScope, string, bool) (FreezeControl, error)
	IsFrozen(context.Context, string, string) (bool, FreezeScope, error)
	Headroom(context.Context, string) (Headroom, error)
}

type service struct {
	repository Repository
	clock      schedule.Clock
	usage      UsageReader
}

func NewService(repository Repository, clock schedule.Clock, usage ...UsageReader) Service {
	value := &service{repository: repository, clock: clock}
	if len(usage) > 0 {
		value.usage = usage[0]
	}
	return value
}

func (s *service) Create(ctx context.Context, value Budget) (Budget, error) {
	if s.repository == nil || s.clock == nil {
		return Budget{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	value, err := validate(value)
	if err != nil {
		return Budget{}, err
	}
	value.CreatedAt = s.clock.Now().UTC()
	return s.repository.Create(ctx, value)
}

func validate(value Budget) (Budget, error) {
	value.ID = strings.TrimSpace(value.ID)
	value.BookID = strings.TrimSpace(value.BookID)
	value.Currency = strings.ToUpper(strings.TrimSpace(value.Currency))
	if value.ID == "" {
		return Budget{}, fmt.Errorf("%w: id is required", ErrInvalid)
	}
	if value.BookID == "" {
		return Budget{}, fmt.Errorf("%w: book_id is required", ErrInvalid)
	}
	if value.Currency == "" {
		return Budget{}, fmt.Errorf("%w: currency is required", ErrInvalid)
	}
	if value.TotalCapMinor <= 0 {
		return Budget{}, fmt.Errorf("%w: total_cap_minor must be positive", ErrInvalid)
	}
	if value.PeriodicCapMinor <= 0 || value.Period <= 0 {
		return Budget{}, fmt.Errorf("%w: periodic cap and period must be positive", ErrInvalid)
	}
	if value.PerTransactionCapMinor <= 0 {
		return Budget{}, fmt.Errorf("%w: per_transaction_cap_minor must be positive", ErrInvalid)
	}
	if value.PerTransactionCapMinor > value.PeriodicCapMinor || value.PeriodicCapMinor > value.TotalCapMinor {
		return Budget{}, fmt.Errorf("%w: caps must satisfy per-transaction <= periodic <= total", ErrInvalid)
	}
	value.AllowedCounterparties = normalizeCounterparties(value.AllowedCounterparties)
	value.DeniedCounterparties = normalizeCounterparties(value.DeniedCounterparties)
	return value, nil
}

func (s *service) Get(ctx context.Context, id string) (Budget, error) {
	if s.repository == nil {
		return Budget{}, fmt.Errorf("%w: repository is required", ErrInvalid)
	}
	return s.repository.Get(ctx, strings.TrimSpace(id))
}

// SetCaps creates the initial policy or updates only its cap and counterparty
// fields. Approval gating and the emergency freeze have separate commands so a
// cap edit cannot silently weaken either safety control.
func (s *service) SetCaps(ctx context.Context, requested Budget) (Budget, error) {
	if s.repository == nil || s.clock == nil {
		return Budget{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
	existing, err := s.Get(ctx, requested.ID)
	if errors.Is(err, ErrNotFound) {
		requested.RequiresApproval = false
		requested.Frozen = false
		return s.Create(ctx, requested)
	}
	if err != nil {
		return Budget{}, err
	}
	requested.ID = existing.ID
	requested.BookID = existing.BookID
	requested.Currency = existing.Currency
	requested.RequiresApproval = existing.RequiresApproval
	requested.Frozen = existing.Frozen
	requested.CreatedAt = existing.CreatedAt
	requested, err = validate(requested)
	if err != nil {
		return Budget{}, err
	}
	return s.repository.Update(ctx, requested)
}

func (s *service) SetGating(ctx context.Context, id string, required bool) (Budget, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return Budget{}, err
	}
	value.RequiresApproval = required
	return s.repository.Update(ctx, value)
}

func (s *service) SetFrozen(ctx context.Context, id string, frozen bool) (Budget, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return Budget{}, err
	}
	next, err := TransitionFreeze(FreezeState{Frozen: value.Frozen}, freezeEvent(frozen))
	if err != nil {
		return Budget{}, err
	}
	value.Frozen = next.Frozen
	return s.repository.Update(ctx, value)
}

func (s *service) SetScopeFrozen(ctx context.Context, scope FreezeScope, id string, frozen bool) (FreezeControl, error) {
	repository, ok := s.repository.(FreezeRepository)
	if !ok || s.clock == nil {
		return FreezeControl{}, fmt.Errorf("%w: freeze repository and clock are required", ErrInvalid)
	}
	id = strings.TrimSpace(id)
	if scope == FreezeScopeScenario {
		id = "*"
	}
	if scope != FreezeScopeBook && scope != FreezeScopeScenario || id == "" {
		return FreezeControl{}, fmt.Errorf("%w: book or scenario freeze scope is required", ErrInvalid)
	}
	current := FreezeState{}
	if stored, err := repository.GetFreezeControl(ctx, scope, id); err == nil {
		current.Frozen = stored.Frozen
	} else if !errors.Is(err, ErrNotFound) {
		return FreezeControl{}, err
	}
	next, err := TransitionFreeze(current, freezeEvent(frozen))
	if err != nil {
		return FreezeControl{}, err
	}
	return repository.SetFreezeControl(ctx, FreezeControl{Scope: scope, ScopeID: id, Frozen: next.Frozen, UpdatedAt: s.clock.Now().UTC()})
}

func (s *service) IsFrozen(ctx context.Context, bookID, budgetID string) (bool, FreezeScope, error) {
	repository, ok := s.repository.(FreezeRepository)
	if !ok {
		return false, "", fmt.Errorf("%w: freeze repository is required", ErrInvalid)
	}
	if control, err := repository.GetFreezeControl(ctx, FreezeScopeScenario, "*"); err == nil && control.Frozen {
		return true, FreezeScopeScenario, nil
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return false, "", err
	}
	bookID = strings.TrimSpace(bookID)
	if bookID != "" {
		if control, err := repository.GetFreezeControl(ctx, FreezeScopeBook, bookID); err == nil && control.Frozen {
			return true, FreezeScopeBook, nil
		} else if err != nil && !errors.Is(err, ErrNotFound) {
			return false, "", err
		}
	}
	policy, err := s.Get(ctx, budgetID)
	if err != nil {
		return false, "", err
	}
	if policy.BookID != bookID {
		return false, "", fmt.Errorf("%w: budget does not belong to book", ErrInvalid)
	}
	if policy.Frozen {
		return true, FreezeScopeBudget, nil
	}
	return false, "", nil
}

func (s *service) Headroom(ctx context.Context, id string) (Headroom, error) {
	if s.usage == nil || s.clock == nil {
		return Headroom{}, fmt.Errorf("%w: authorization usage reader and clock are required", ErrInvalid)
	}
	policy, err := s.Get(ctx, id)
	if err != nil {
		return Headroom{}, err
	}
	now := s.clock.Now().UTC()
	periodStart := time.Unix(0, now.UnixNano()-(now.UnixNano()%policy.Period.Nanoseconds())).UTC()
	usage, err := s.usage.BudgetUsage(ctx, policy.ID, periodStart, now)
	if err != nil {
		return Headroom{}, fmt.Errorf("compute budget usage: %w", err)
	}
	totalRemaining := max(int64(0), policy.TotalCapMinor-usage.TotalMinor)
	periodRemaining := max(int64(0), policy.PeriodicCapMinor-usage.PeriodMinor)
	available := min(policy.PerTransactionCapMinor, min(totalRemaining, periodRemaining))
	if policy.Frozen {
		available = 0
	}
	return Headroom{
		BudgetID: policy.ID, BookID: policy.BookID, Currency: policy.Currency,
		TotalCapMinor: policy.TotalCapMinor, TotalUsedMinor: usage.TotalMinor, TotalRemainingMinor: totalRemaining,
		PeriodicCapMinor: policy.PeriodicCapMinor, PeriodUsedMinor: usage.PeriodMinor, PeriodRemainingMinor: periodRemaining,
		PerTransactionCapMinor: policy.PerTransactionCapMinor, AvailableMinor: available,
		PeriodStartedAt: periodStart, ComputedAt: now,
	}, nil
}

func (b Budget) AllowsCounterparty(counterparty string) bool {
	counterparty = strings.ToLower(strings.TrimSpace(counterparty))
	if counterparty == "" || slices.Contains(b.DeniedCounterparties, counterparty) {
		return false
	}
	return len(b.AllowedCounterparties) == 0 || slices.Contains(b.AllowedCounterparties, counterparty)
}

func normalizeCounterparties(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	slices.Sort(out)
	return out
}

var _ Service = (*service)(nil)
