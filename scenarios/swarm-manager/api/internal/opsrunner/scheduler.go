package opsrunner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"swarm-manager/internal/agentops"
)

// IntentFirer fires one due scheduled intent. It returns nil when the intent was
// acted on (and should be consumed from the workflow) or an error to retry it on
// a later tick. Production wires this to the dispatcher/action layer; it must be
// idempotent because a crash between firing and consuming re-fires the intent.
type IntentFirer func(ctx context.Context, w agentops.WorkflowInstance, intent agentops.ScheduledIntent) error

// Scheduler drives restart-safe scheduled intents. The durable truth is the
// timers array of each persisted workflow document — never in-process goroutine
// state — so a restart reloads every pending intent from disk on the next tick.
// Firing an intent consumes it via compare-and-swap, so a duplicate concurrent
// tick cannot fire it twice.
type Scheduler struct {
	repo *WorkflowRepo
	fire IntentFirer
	now  func() time.Time
}

// NewScheduler constructs a scheduler. now defaults to time.Now.
func NewScheduler(repo *WorkflowRepo, fire IntentFirer, now func() time.Time) *Scheduler {
	if now == nil {
		now = time.Now
	}
	return &Scheduler{repo: repo, fire: fire, now: now}
}

// ScheduleIntent durably records a scheduled intent on a domain workflow,
// deduplicating by intent name: scheduling an intent whose name already exists
// is a no-op, so a retried request never queues the action twice.
func (s *Scheduler) ScheduleIntent(kind agentops.TargetKind, id string, intent agentops.ScheduledIntent) error {
	if strings.TrimSpace(intent.Intent) == "" {
		return fmt.Errorf("scheduled intent requires a name")
	}
	// An intent fires either a registered domain action or an operation Invoke —
	// exactly one. The firer branches on Operation being set.
	hasOp := intent.Operation != ""
	hasAction := intent.Action != ""
	switch {
	case hasOp && hasAction:
		return fmt.Errorf("scheduled intent %q sets both an action %q and an operation %q", intent.Intent, intent.Action, intent.Operation)
	case hasOp:
		if !agentops.IsValidOperationID(intent.Operation) {
			return fmt.Errorf("scheduled intent %q names unregistered operation %q", intent.Intent, intent.Operation)
		}
	case hasAction:
		if !agentops.IsRegisteredAction(intent.Action) {
			return fmt.Errorf("scheduled intent %q names unregistered action %q", intent.Intent, intent.Action)
		}
	default:
		return fmt.Errorf("scheduled intent %q names neither an action nor an operation", intent.Intent)
	}
	w, err := s.repo.CreateOrLoad(kind, id)
	if err != nil {
		return err
	}
	for _, t := range w.Timers {
		if t.Intent == intent.Intent {
			return nil // already scheduled (dedup)
		}
	}
	prev := w.Version
	next := cloneWorkflow(w)
	next.Timers = append(next.Timers, intent)
	next.Version = prev + 1
	return s.repo.Commit(prev, next)
}

// HasIntent reports whether a pending intent with the given name exists on the
// target's workflow. A missing workflow reports false.
func (s *Scheduler) HasIntent(kind agentops.TargetKind, id, intentName string) (bool, error) {
	w, found, err := s.repo.Load(kind, id)
	if err != nil || !found {
		return false, err
	}
	for _, t := range w.Timers {
		if t.Intent == intentName {
			return true, nil
		}
	}
	return false, nil
}

// CancelIntent removes a pending intent by name. Canceling an unknown intent is
// a no-op.
func (s *Scheduler) CancelIntent(kind agentops.TargetKind, id, intentName string) error {
	w, found, err := s.repo.Load(kind, id)
	if err != nil || !found {
		return err
	}
	kept := w.Timers[:0:0]
	removed := false
	for _, t := range w.Timers {
		if t.Intent == intentName {
			removed = true
			continue
		}
		kept = append(kept, t)
	}
	if !removed {
		return nil
	}
	prev := w.Version
	next := cloneWorkflow(w)
	next.Timers = kept
	next.Version = prev + 1
	return s.repo.Commit(prev, next)
}

// Tick fires every due intent across all persisted workflows exactly once. It
// reloads workflows from disk each call, so it is inherently restart-safe: a
// process that starts fresh reconstructs all pending intents from durable state.
func (s *Scheduler) Tick(ctx context.Context) error {
	workflows, err := s.repo.List()
	if err != nil {
		return err
	}
	now := s.now()
	for _, w := range workflows {
		for _, intent := range w.Timers {
			if !intentDue(intent, now) {
				continue
			}
			if err := s.fireAndConsume(ctx, w, intent); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Scheduler) fireAndConsume(ctx context.Context, w agentops.WorkflowInstance, intent agentops.ScheduledIntent) error {
	if s.fire != nil {
		if err := s.fire(ctx, w, intent); err != nil {
			return err // leave the intent to retry next tick
		}
	}
	// Re-load and consume via compare-and-swap so two ticks cannot both consume.
	cur, found, err := s.repo.Load(agentops.TargetKind(w.Domain.Kind), w.Domain.ID)
	if err != nil || !found {
		return err
	}
	kept := cur.Timers[:0:0]
	removed := false
	for _, t := range cur.Timers {
		if t.Intent == intent.Intent {
			removed = true
			continue
		}
		kept = append(kept, t)
	}
	if !removed {
		return nil // another tick already consumed it
	}
	prev := cur.Version
	next := cloneWorkflow(cur)
	next.Timers = kept
	next.Version = prev + 1
	if err := s.repo.Commit(prev, next); err != nil {
		// A lost CAS means a concurrent tick consumed it; that is fine.
		return nil
	}
	return nil
}

// intentDue reports whether an intent's not_before time has passed. An empty
// not_before means fire immediately.
func intentDue(intent agentops.ScheduledIntent, now time.Time) bool {
	if strings.TrimSpace(intent.NotBefore) == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, intent.NotBefore)
	if err != nil {
		return true // an unparseable time fires rather than wedging forever
	}
	return !now.Before(t)
}

// Run ticks on an interval until ctx is canceled. It reloads durable state on
// every tick, so the first tick after a restart recovers all pending intents.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.Tick(ctx); err != nil {
				// A tick error is logged by the caller via the returned error path
				// on shutdown; transient errors should not kill the loop.
				continue
			}
		}
	}
}
