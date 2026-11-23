package launch

import (
	"context"
	"errors"
	"fmt"
	"time"

	codexrollout "github.com/vrooli/vrooli/packages/codex-rollout"

	"research-desk/internal/home"
)

// RecoverEvery is the interval between recovery passes. The shortest
// backoff in turn_errors.json is a minute.
const RecoverEvery = time.Minute

// RunRecovery runs a recovery pass every interval until ctx ends.
func (s *Service) RunRecovery(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		for _, line := range s.Recover(ctx) {
			s.Logger.Printf("recovery: %s", line)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Recover is one pass over the halted commissions (SYSTEM.md §6): a halt
// whose plan is resume and whose resume is due gets /goal resume typed into
// its own thread, and one whose plan is escalate gets its escalation
// recorded once. Every act is a LAUNCH.json recovery entry. Each commission is
// re-read under its claim before anything is typed, so an act is never
// repeated for a halt that changed or was already answered. It returns one
// line per act.
func (s *Service) Recover(ctx context.Context) []string {
	if err := s.Writer.Held(); err != nil {
		return nil
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return []string{"pass skipped: " + err.Error()}
	}
	var out []string
	for _, c := range snap.Commissions {
		h := c.Halt
		if c.State != home.StateHalted || h == nil {
			continue
		}
		due := h.Plan.Action == codexrollout.ActionResume && !snap.Now.Before(h.Plan.Due)
		escalate := h.Plan.Action == codexrollout.ActionEscalate && !h.Escalated
		if !due && !escalate {
			continue
		}
		if line := s.recoverOne(ctx, c.ID, h); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// recoverOne acts on one halt seen in the pass's snapshot, after re-reading
// the commission under its claim.
func (s *Service) recoverOne(ctx context.Context, id string, seen *home.Halt) string {
	if !s.claim(id) {
		return ""
	}
	defer s.release(id)
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), controlTime)
	defer cancel()
	_, c, err := s.find(ctx, id)
	if err != nil {
		return id + ": not re-read: " + err.Error()
	}
	h := c.Halt
	now := s.Clock.Now().UTC()
	if c.State != home.StateHalted || h == nil || h.TurnID != seen.TurnID || h.Plan.Action != seen.Plan.Action || h.Plan.Attempt != seen.Plan.Attempt {
		return ""
	}
	record := launchRecord(c)
	t := targetOf(c)
	entry := home.RecoveryEntry{At: now, TurnID: h.TurnID, Class: h.Class, Rule: h.Rule, Action: h.Plan.Action}
	switch {
	case h.Plan.Action == codexrollout.ActionEscalate && !h.Escalated:
		entry.Outcome = "escalated to the steward: " + h.Plan.Escalation
	case h.Plan.Action == codexrollout.ActionResume && !now.Before(h.Plan.Due):
		entry.Attempt = h.Plan.Attempt
		status, rerr := s.Launcher.Resume(ctx, t)
		entry.GoalStatus = status
		switch {
		case rerr == nil && (status == "blocked" || status == "paused"):
			entry.Outcome = fmt.Sprintf("typed /goal resume (attempt %d of %d), but the goal is still %s", h.Plan.Attempt, h.Recovery.MaxAttempts, status)
		case rerr == nil:
			entry.Outcome = fmt.Sprintf("resumed (attempt %d of %d); goal %s", h.Plan.Attempt, h.Recovery.MaxAttempts, orUnknown(status))
		case errors.Is(rerr, ErrNotReady):
			entry.Outcome = fmt.Sprintf("not resumed (attempt %d of %d): %v", h.Plan.Attempt, h.Recovery.MaxAttempts, rerr)
		default:
			entry.Outcome = fmt.Sprintf("resume failed (attempt %d of %d): %v", h.Plan.Attempt, h.Recovery.MaxAttempts, rerr)
		}
	default:
		return ""
	}
	fillTarget(record, t)
	record.Recoveries = append(record.Recoveries, entry)
	if _, err := s.Writer.WriteLaunch(id, record); err != nil {
		return fmt.Sprintf("%s: %s, but LAUNCH.json was not written: %v", id, entry.Outcome, err)
	}
	return fmt.Sprintf("%s: %s error at %s: %s", id, h.Class, h.At.UTC().Format("2006-01-02 15:04Z"), entry.Outcome)
}

func orUnknown(s string) string {
	if s == "" {
		return "status unknown"
	}
	return s
}
