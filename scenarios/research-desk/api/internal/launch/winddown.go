package launch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"research-desk/internal/home"
)

// Automatic wind-down is the control side of the charter's allowance end
// (SYSTEM.md §6): a manager that is still active near its allowance gets the
// same OPERATOR.md wind-down the operator would write, then, if it still has
// not stopped after the grace, the desk pauses its goal. A pause is not a
// kill: the session and every owner stay available for a resumed closeout.
const (
	WindDownEvery = time.Minute
	// WindDownLead is how early the manager gets the wind-down. It is longer
	// than the 20-minute unit wait in the current charter, so a manager
	// blocked on an owner still wakes to read it.
	WindDownLead = 30 * time.Minute
	// WindDownGrace is the wrap-up time after the allowance before the desk
	// pauses a goal that ignored the wind-down. The manager's own step 7
	// writes the closeout and pauses it in this window.
	WindDownGrace = 15 * time.Minute
	// autoPauseRetry is how soon a failed pause is tried again. A readiness
	// failure is usually a transient screen/transcript read.
	autoPauseRetry = 5 * time.Minute
)

// RunWindDown runs a wind-down pass every interval until ctx ends.
func (s *Service) RunWindDown(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		for _, line := range s.WindDown(ctx) {
			s.Logger.Printf("wind-down: %s", line)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// WindDown gives every active manager whose allowance is near or over the
// existing wind-down control once, then pauses a goal that ignored it past
// the grace. A commission that already has a wind-down after its start, or
// an auto-pause after that wind-down, is left alone.
func (s *Service) WindDown(ctx context.Context) []string {
	if err := s.Writer.Held(); err != nil {
		return nil
	}
	snap, err := s.Reader.Snapshot(ctx, home.Options{})
	if err != nil {
		return []string{"pass skipped: " + err.Error()}
	}
	var out []string
	for _, c := range snap.Commissions {
		if c.State != home.StateLive || !c.AllowanceEnd.Known {
			continue
		}
		if snap.Now.Before(c.AllowanceEnd.Value.Add(-WindDownLead)) {
			continue
		}
		if line := s.windDownOne(ctx, c.ID); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// windDownOne re-reads one commission under its claim before writing the
// wind-down or pausing, so a manager that stopped in the meantime is not
// touched.
func (s *Service) windDownOne(ctx context.Context, id string) string {
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
	if c.State != home.StateLive || !c.AllowanceEnd.Known {
		return ""
	}
	now := s.Clock.Now().UTC()
	if now.Before(c.AllowanceEnd.Value.Add(-WindDownLead)) {
		return ""
	}
	windDown := c.Operator.Newest(home.EntryWindDown, "")
	if windDown == nil || (c.Started.Known && windDown.At.Before(c.Started.Value)) {
		entry := home.Entry{At: now, Kind: home.EntryWindDown, Via: home.ViaDesk, Body: home.WindDownText}
		if _, err := s.Writer.AppendOperator(id, entry); err != nil {
			return fmt.Sprintf("%s: wind-down not written: %v", id, err)
		}
		return fmt.Sprintf("%s: wind-down written (allowance ended %s)", id, c.AllowanceEnd.Value.UTC().Format("2006-01-02 15:04Z"))
	}
	if now.Before(c.AllowanceEnd.Value.Add(WindDownGrace)) || now.Before(windDown.At.Add(WindDownGrace)) || pausedAfter(c, windDown.At, now) {
		return ""
	}
	status, pauseErr := s.Launcher.Pause(ctx, targetOf(c))
	outcome := "auto-pause after wind-down; goal " + orUnknown(status)
	if pauseErr != nil {
		outcome = "auto-pause after wind-down failed: " + pauseErr.Error()
	}
	record := launchRecord(c)
	fillTarget(record, targetOf(c))
	record.Controls = append(record.Controls, home.ControlEntry{Action: "auto-pause", At: now, Outcome: outcome})
	if _, err := s.Writer.WriteLaunch(id, record); err != nil {
		return fmt.Sprintf("%s: %s, but LAUNCH.json was not written: %v", id, outcome, err)
	}
	return fmt.Sprintf("%s: %s", id, outcome)
}

// pausedAfter reports whether the desk already recorded a successful
// auto-pause after the wind-down. A failed attempt is retried after a short
// interval, so a transient readiness failure does not leave the goal open.
func pausedAfter(c *home.Commission, windDownAt, now time.Time) bool {
	if c.Launch == nil {
		return false
	}
	for i := len(c.Launch.Controls) - 1; i >= 0; i-- {
		control := c.Launch.Controls[i]
		if control.Action != "auto-pause" || control.At.Before(windDownAt) {
			continue
		}
		if !strings.Contains(control.Outcome, "failed") || now.Sub(control.At) < autoPauseRetry {
			return true
		}
	}
	return false
}
