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

type Service interface {
	Create(context.Context, Budget) (Budget, error)
	Get(context.Context, string) (Budget, error)
}

type service struct {
	repository Repository
	clock      schedule.Clock
}

func NewService(repository Repository, clock schedule.Clock) Service {
	return &service{repository: repository, clock: clock}
}

func (s *service) Create(ctx context.Context, value Budget) (Budget, error) {
	if s.repository == nil || s.clock == nil {
		return Budget{}, fmt.Errorf("%w: service dependencies are required", ErrInvalid)
	}
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
	value.CreatedAt = s.clock.Now().UTC()
	return s.repository.Create(ctx, value)
}

func (s *service) Get(ctx context.Context, id string) (Budget, error) {
	if s.repository == nil {
		return Budget{}, fmt.Errorf("%w: repository is required", ErrInvalid)
	}
	return s.repository.Get(ctx, strings.TrimSpace(id))
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
