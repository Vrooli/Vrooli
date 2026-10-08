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
//
// A reversible ask with a default never escalates or expires: the sweep
// applies its default once it is due (see applyDefaultIfDue). Every other
// overdue ask follows the recipient's escalation chain and then expires.
func (s *Service) ProcessEscalations(ctx context.Context) error {
	now := s.clock.Now().UTC()
	rows, err := s.db.QueryContext(ctx, `SELECT id, notification_id, default_answer, reversible FROM asks WHERE state IN ('pending', 'escalated') AND deadline <= ? ORDER BY deadline`, now.Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	defer rows.Close()
	type dueAsk struct {
		id, notificationID, defaultAnswer string
		reversible                        bool
	}
	var due []dueAsk
	for rows.Next() {
		var item dueAsk
		var reversible int
		if err := rows.Scan(&item.id, &item.notificationID, &item.defaultAnswer, &reversible); err != nil {
			return err
		}
		item.reversible = reversible != 0
		due = append(due, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_ = rows.Close()
	for _, item := range due {
		if item.defaultAnswer != "" && item.reversible {
			if err := s.applyDefaultIfDue(ctx, item.id); err != nil {
				return err
			}
			continue
		}
		if err := s.escalateAsk(ctx, item.id, item.notificationID); err != nil {
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
		nowText := now.Format(time.RFC3339Nano)
		result, err := s.db.ExecContext(ctx, `UPDATE asks SET state='expired', reason='escalation chain exhausted without an answer', resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, nowText, nowText, askID)
		if err != nil {
			return err
		}
		if changed, _ := result.RowsAffected(); changed == 1 {
			s.kickResolutions(ctx)
		}
		return nil
	}
	// An escalation is a new notification to the operator, so it waits out a
	// quiet window exactly as the original ask did; the next sweep retries.
	if quiet, quietErr := s.inQuietWindow(ctx, n.RequestedBy, n.Urgency, now); quietErr != nil {
		return quietErr
	} else if quiet {
		return nil
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
