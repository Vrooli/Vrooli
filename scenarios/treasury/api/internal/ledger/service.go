// Package ledger emits settled movement through money-ledger's neutral contract.
package ledger

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	ledgerflow "treasury/internal/ledger/flow"
)

type Emitter interface {
	Emit(context.Context, Emission) (duplicate bool, err error)
}

type Service struct {
	repository Repository
	emitter    Emitter
	now        func() time.Time
}

func NewService(repository Repository, emitter Emitter, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, emitter: emitter, now: now}
}

// DrainPending is deliberately separate from settlement. The outbox row is
// already committed with the settlement, so any downstream delay or failure
// changes only delivery metadata and never the financial outcome.
func (s *Service) DrainPending(ctx context.Context) error {
	if s == nil || s.repository == nil || s.emitter == nil {
		return errors.New("ledger repository and emitter are required")
	}
	values, err := s.repository.Pending(ctx, 100)
	if err != nil {
		return err
	}
	var failures []error
	for _, value := range values {
		if _, emitErr := s.emitter.Emit(ctx, value); emitErr != nil {
			if _, transitionErr := ledgerflow.Transition(ledgerflow.State{Status: ledgerflow.Status(value.Status)}, ledgerflow.EventDeliveryFailed); transitionErr != nil {
				failures = append(failures, transitionErr)
				continue
			}
			_ = s.repository.MarkFailure(context.WithoutCancel(ctx), value.ID, emitErr.Error())
			failures = append(failures, fmt.Errorf("emit settlement %s: %w", value.SettlementID, emitErr))
			continue
		}
		if _, transitionErr := ledgerflow.Transition(ledgerflow.State{Status: ledgerflow.Status(value.Status)}, ledgerflow.EventDeliveryAccepted); transitionErr != nil {
			failures = append(failures, transitionErr)
			continue
		}
		if markErr := s.repository.MarkAccepted(context.WithoutCancel(ctx), value.ID, s.now().UTC()); markErr != nil {
			failures = append(failures, markErr)
		}
	}
	return errors.Join(failures...)
}

// Run drains on startup and at a bounded interval. A durable outbox makes
// process restarts and prolonged dependency outages ordinary retry cases.
func (s *Service) Run(ctx context.Context, interval time.Duration, logger *log.Logger) {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	drain := func() {
		drainCtx, cancel := context.WithTimeout(ctx, min(interval, 5*time.Second))
		defer cancel()
		if err := s.DrainPending(drainCtx); err != nil && logger != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("money-ledger emission deferred: %v", err)
		}
	}
	drain()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drain()
		}
	}
}

type UnavailableEmitter struct{ Cause error }

func (u UnavailableEmitter) Emit(context.Context, Emission) (bool, error) {
	if u.Cause == nil {
		u.Cause = errors.New("money-ledger destination is not configured")
	}
	return false, u.Cause
}

func Basis(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "operator_attestation") {
		return "operator_asserted"
	}
	return "authoritative"
}
