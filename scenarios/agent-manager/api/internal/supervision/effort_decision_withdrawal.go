package supervision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/vrooli/api-core/operatordecision"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// An effort decision is one immutable question under its own id, so it never
// piles up by repeating. It goes stale when the question is settled somewhere
// other than the phone ask, or when the effort it belongs to ends. Either way
// the hub ask is closed by the request's event id, from a durable queue so a
// failed close is retried and a repeat is harmless.
const (
	decisionAnsweredHereReason = "Already answered in Agent Manager."
	decisionEffortEndedReason  = "The effort this question belongs to has ended."
)

type decisionWithdrawal struct {
	EventID   string
	Reason    string
	CreatedAt time.Time
}

// QueueDecisionWithdrawal records that the request event's hub ask must be
// closed. A second call for the same event keeps the first reason and time, so
// the withdrawal event keeps one identity across retries.
func (r *Repository) QueueDecisionWithdrawal(ctx context.Context, eventID, reason string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO supervision_decision_withdrawals(event_id,reason,created_at) VALUES(?,?,?) ON CONFLICT(event_id) DO NOTHING`, eventID, reason, formatTime(r.now()))
	return err
}

// PendingDecisionWithdrawals lists closes the hub has not yet accepted, oldest first.
func (r *Repository) PendingDecisionWithdrawals(ctx context.Context, limit int) ([]decisionWithdrawal, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT event_id,reason,created_at FROM supervision_decision_withdrawals WHERE sent_at IS NULL ORDER BY created_at, event_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []decisionWithdrawal
	for rows.Next() {
		var w decisionWithdrawal
		var created string
		if err := rows.Scan(&w.EventID, &w.Reason, &created); err != nil {
			return nil, err
		}
		if w.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, fmt.Errorf("decision withdrawal %s: %w", w.EventID, err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// MarkDecisionWithdrawalSent records the hub's receipt.
func (r *Repository) MarkDecisionWithdrawalSent(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE supervision_decision_withdrawals SET sent_at=? WHERE event_id=? AND sent_at IS NULL`, formatTime(r.now()), eventID)
	return err
}

// queueDecisionWithdrawal queues the close of a decision's published request.
// A request that never reached the hub has no ask to close. Callers hold decisionMu.
func (s *EffortService) queueDecisionWithdrawal(ctx context.Context, d *pb.EffortDecision, reason string) {
	if d == nil || d.GetEventId() == "" || d.GetPublishedAt() == nil {
		return
	}
	if err := s.repo.QueueDecisionWithdrawal(ctx, d.GetEventId(), reason); err != nil {
		slog.Warn("queue decision withdrawal failed", "decision", d.GetDecisionId(), "error", err)
	}
}

// withdrawDecisionOfEndedEffort queues the close of a still-pending decision
// whose effort was withdrawn. Callers hold decisionMu.
func (s *EffortService) withdrawDecisionOfEndedEffort(ctx context.Context, d *pb.EffortDecision) {
	if d.GetState() != DecisionPending || d.GetEffortRef() == "" || d.GetPublishedAt() == nil {
		return
	}
	if e, _, err := s.repo.GetEffort(ctx, d.GetEffortRef()); err == nil && e != nil && e.Withdrawn {
		s.queueDecisionWithdrawal(ctx, d, decisionEffortEndedReason)
	}
}

// flushDecisionWithdrawals sends what is queued; the next tick retries failures.
func (s *EffortService) flushDecisionWithdrawals(ctx context.Context) {
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	if err := s.publishDecisionWithdrawals(ctx); err != nil {
		slog.Warn("decision withdrawal publish failed", "error", err)
	}
}

// publishDecisionWithdrawals sends every queued close. Callers hold decisionMu.
// The withdrawal event is built from the stored row alone, so a retry publishes
// the same event id.
func (s *EffortService) publishDecisionWithdrawals(ctx context.Context) error {
	if s.decisionPublisher == nil {
		return nil
	}
	pending, err := s.repo.PendingDecisionWithdrawals(ctx, decisionPublicationPageSize)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var errs []error
	for _, w := range pending {
		event := operatordecision.BuildWithdrawalEventTyped(effortdecision.Source, goalhome.DecisionRequestedEventType, w.CreatedAt, operatordecision.Withdrawal{
			RequesterRef: w.EventID, Reason: w.Reason,
		})
		if err := s.decisionPublisher.PublishDomainEvent(ctx, event); err != nil {
			errs = append(errs, fmt.Errorf("withdraw decision request %s: %w", w.EventID, err))
			continue
		}
		if err := s.repo.MarkDecisionWithdrawalSent(ctx, w.EventID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
