package heartbeat

import (
	"context"
	"fmt"
	"log"
	"time"
)

// ResetEligibility is the runtime owner's signal that the condition which
// paused a member's obligation (for example, a quota wait) has cleared and the
// parked run may be resumed. It is the consuming half of the pause/resume
// contract: the provider/credential-pool owner classifies reset eligibility,
// and this package consumes each event at most once. The later supervision
// child owns the concrete provider/credential-pool classification and live
// integration; this port is deliberately narrow so the consuming behavior is
// qualified with synthetic owner events first.
type ResetEligibility struct {
	// EventID is the owner's stable identity for this eligibility event. A
	// replayed event cannot resume a run twice. When empty, a deterministic
	// key is derived from the event fields.
	EventID string
	// TeamID and AgentID identify the paused obligation.
	TeamID  string
	AgentID string
	// RunID is the parked owner run. An event naming a different run is stale
	// and ignored.
	RunID string
	// Known is true when the owner supplied a definite reset time. An unknown
	// reset is retained, never polled and never consumed.
	Known bool
	// ResumesAt is the owner-observed time the allowance becomes eligible.
	// Meaningful only when Known is true.
	ResumesAt time.Time
	// Reason is owner diagnostic context, forwarded to the wake call.
	Reason string
}

// ResetEligibilitySource yields runtime-owner eligibility events. Implementations
// are event-driven: they must not poll an unknown reset on a timer.
type ResetEligibilitySource interface {
	// Next blocks until the next owner eligibility event, ctx is done, or the
	// source is exhausted. Exhaustion returns ok=false with a nil error.
	Next(ctx context.Context) (event ResetEligibility, ok bool, err error)
}

// PausedRunResumer is the narrow runtime-owner port used to resume a
// quota-paused (parked) obligation after the owner reports reset eligibility.
// It maps to agent-manager's existing parked->running WakeRun primitive.
type PausedRunResumer interface {
	WakePausedRun(ctx context.Context, runID, reason string) error
}

// ResetConsumeOutcome reports what ConsumeResetEligibility did with an event.
type ResetConsumeOutcome string

const (
	// ResetConsumeNone means the event did not apply: an unknown team/member, a
	// stale run identity, or a member that is not paused.
	ResetConsumeNone ResetConsumeOutcome = "none"
	// ResetConsumeDuplicate means this event (or an equivalent one) was already
	// consumed; the owner is not woken again.
	ResetConsumeDuplicate ResetConsumeOutcome = "duplicate"
	// ResetConsumeDeferred means the obligation is correctly paused but no wake
	// is due yet: the reset is unknown, or the known reset time has not
	// arrived. The caller waits for owner evidence rather than polling.
	ResetConsumeDeferred ResetConsumeOutcome = "deferred"
	// ResetConsumeResumed means the owner accepted the wake and the slot
	// returned to running ownership.
	ResetConsumeResumed ResetConsumeOutcome = "resumed"
)

func (c *TeamExecutionContext) clockNow() time.Time {
	if c.clock != nil {
		return c.clock()
	}
	return time.Now()
}

// resetEligibilityEventKey derives an idempotency key for an owner event. The
// owner-supplied EventID wins; otherwise the key is deterministic over the
// event fields so equivalent replays collapse.
func resetEligibilityEventKey(ev ResetEligibility) string {
	if ev.EventID != "" {
		return ev.EventID
	}
	return fmt.Sprintf("%s|%t|%s|%s", ev.RunID, ev.Known, ev.ResumesAt.UTC().Format(time.RFC3339Nano), ev.Reason)
}

// ConsumeResetEligibility processes one runtime-owner reset-eligibility event.
// It resumes the matching paused obligation through the owner exactly once per
// event:
//
//   - an unknown reset is retained and never consumed (no polling);
//   - a known reset is consumed only once the clock reaches ResumesAt;
//   - a duplicate event (same key) is a no-op;
//   - a failed owner wake keeps the obligation paused so distinct owner
//     evidence can retry — it is not a reason to admit a duplicate heartbeat.
//
// The event key is marked under the context lock before the owner call so a
// concurrent or replayed event cannot wake the same run twice. The hourly tick
// remains refused throughout because the paused obligation still holds its slot.
func (c *TeamExecutionContext) ConsumeResetEligibility(ctx context.Context, ev ResetEligibility) (ResetConsumeOutcome, error) {
	if ev.TeamID != "" && ev.TeamID != c.teamID {
		return ResetConsumeNone, nil
	}

	c.mu.Lock()
	entry, ok := c.running[ev.AgentID]
	if !ok || entry.State != ObligationPaused {
		c.mu.Unlock()
		return ResetConsumeNone, nil
	}
	if ev.RunID != "" && entry.RunID != "" && entry.RunID != ev.RunID {
		c.mu.Unlock()
		return ResetConsumeNone, nil
	}
	key := resetEligibilityEventKey(ev)
	if key != "" && entry.ResetConsumedEventID == key {
		c.mu.Unlock()
		return ResetConsumeDuplicate, nil
	}
	if !ev.Known {
		// Unknown reset: retain without consuming or probing. A later definite
		// owner event is required.
		c.mu.Unlock()
		return ResetConsumeDeferred, nil
	}
	if c.clockNow().Before(ev.ResumesAt) {
		c.mu.Unlock()
		return ResetConsumeDeferred, nil
	}
	if c.resumer == nil {
		c.mu.Unlock()
		return ResetConsumeDeferred, fmt.Errorf("no paused-run resumer configured for %s", c.teamID)
	}
	if entry.RunID == "" {
		c.mu.Unlock()
		return ResetConsumeDeferred, nil
	}

	// Mark the event consumed before the owner call. A duplicate event while
	// the wake is in flight is a no-op.
	entry.ResetConsumedEventID = key
	c.running[ev.AgentID] = entry
	c.persistLocked()
	runID := entry.RunID
	profileKey := entry.ProfileKey
	taskID := entry.TaskID
	runTag := entry.RunTag
	c.mu.Unlock()

	reason := ev.Reason
	if reason == "" {
		reason = "quota window reset; resume parked heartbeat"
	}
	if err := c.resumer.WakePausedRun(ctx, runID, reason); err != nil {
		// The wake did not happen: clear the marker so a distinct owner event
		// can retry, but keep the obligation paused. The hourly tick stays
		// refused in the meantime.
		c.mu.Lock()
		if cur, ok := c.running[ev.AgentID]; ok && cur.State == ObligationPaused && cur.ResetConsumedEventID == key {
			cur.ResetConsumedEventID = ""
			c.running[ev.AgentID] = cur
			c.persistLocked()
		}
		c.mu.Unlock()
		return ResetConsumeNone, err
	}

	// The owner accepted the wake: the run is live again. Return the slot to
	// running ownership and re-attach a completion waiter so terminal owner
	// state releases it.
	c.mu.Lock()
	if cur, ok := c.running[ev.AgentID]; ok && cur.State == ObligationPaused {
		cur.State = ObligationRunning
		c.running[ev.AgentID] = cur
		c.persistLocked()
	}
	c.mu.Unlock()

	c.watchRetainedRun(ev.AgentID, runID, profileKey, taskID, runTag)
	log.Printf("team_execution: resumed paused obligation for %s/%s (run %s) after owner reset eligibility", c.teamID, ev.AgentID, runID)
	return ResetConsumeResumed, nil
}
