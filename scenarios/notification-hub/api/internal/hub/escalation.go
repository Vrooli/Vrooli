package hub

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const escalationSlice = 5 * time.Minute

// ProcessEscalations advances asks whose current response window has elapsed.
// It is deliberately exported so the scenario's acceptance tests and an
// operator can trigger the same durable worker action without waiting for the
// background sweep.
func (s *Service) ProcessEscalations(ctx context.Context) error {
	now := s.clock.Now().UTC()
	rows, err := s.db.QueryContext(ctx, `SELECT id, notification_id FROM asks WHERE state IN ('pending', 'escalated') AND deadline <= ? ORDER BY deadline`, now.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	defer rows.Close()
	var due [][2]string
	for rows.Next() {
		var item [2]string
		if err := rows.Scan(&item[0], &item[1]); err != nil {
			return err
		}
		due = append(due, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, item := range due {
		if err := s.escalateAsk(ctx, item[0], item[1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) escalateAsk(ctx context.Context, askID, notificationID string) error {
	n, _, err := s.Get(ctx, notificationID)
	if err != nil {
		return err
	}
	chain, err := s.GetEscalationChain(ctx, n.RequestedBy)
	if err != nil {
		return err
	}
	var used int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM escalation_steps WHERE ask_id=?`, askID).Scan(&used); err != nil {
		return err
	}
	now := s.clock.Now().UTC()
	if used >= len(chain.Steps) {
		_, err := s.db.ExecContext(ctx, `UPDATE asks SET state='expired', reason='escalation chain exhausted without an answer', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, now.Format(time.RFC3339Nano), askID)
		return err
	}
	step := chain.Steps[used]
	targets, err := s.channelTargets(ctx, n.RequestedBy)
	if err != nil {
		return err
	}
	var target *channelTarget
	for i := range targets {
		if targets[i].Channel == step.Channel {
			target = &targets[i]
			break
		}
	}
	outcome, reason := "failed", "escalation channel is not configured"
	if target != nil {
		reason = ""
		approved := approvedLabel(target.ApprovedLabels, n.SensitivityLabel)
		body := n.Body
		if !approved {
			body = "Notification available in notification-hub"
		}
		s.mu.RLock()
		push, email, desktop, remote := s.push, s.email, s.desktop, s.remote
		s.mu.RUnlock()
		var provider string
		for attempt := 1; attempt <= MaxAttempts; attempt++ {
			provider, err = s.deliverTarget(ctx, *target, n, body, push, email, desktop, remote)
			attemptOutcome := "delivered"
			attemptReason := ""
			if err != nil {
				attemptOutcome, attemptReason = "failed", safeReason(err)
				reason = attemptReason
			}
			_, _ = s.db.ExecContext(ctx, `INSERT INTO delivery_attempts (notification_id, channel, machine_id, attempt_number, outcome, reason, next_attempt_at, created_at) VALUES (?, ?, ?, ?, ?, ?, '', ?)`, n.ID, target.Channel, target.MachineID, attempt, attemptOutcome, attemptReason, s.clock.Now().UTC().Format(time.RFC3339Nano))
			if err == nil {
				outcome = "delivered"
				nowText := s.clock.Now().UTC().Format(time.RFC3339Nano)
				_, _ = s.db.ExecContext(ctx, `INSERT INTO receipts (id, notification_id, channel, machine_id, provider_id, delivered_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), n.ID, target.Channel, target.MachineID, provider, nowText)
				break
			}
			if attempt < MaxAttempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(1<<(attempt-1)) * 10 * time.Millisecond):
				}
			}
		}
	}
	stepReason := reason
	if stepReason == "" {
		stepReason = fmt.Sprintf("escalation step %d accepted by %s", step.Ordinal, step.Channel)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO escalation_steps (id, ask_id, ordinal, channel, outcome, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), askID, step.Ordinal, step.Channel, outcome, stepReason, now.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE asks SET state='escalated', reason=?, deadline=?, updated_at=? WHERE id=? AND state IN ('pending','escalated')`, stepReason, now.Add(escalationSlice).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), askID)
	return err
}
