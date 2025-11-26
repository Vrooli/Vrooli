package tasks

import (
	"fmt"

	"github.com/ecosystem-manager/api/pkg/internal/timeutil"
)

// RuntimeCoordinator defines the processor hooks needed to enact lifecycle side effects.
type RuntimeCoordinator interface {
	TerminateRunningProcess(taskID string) error
	ForceStartTask(taskID string, allowOverflow bool) error
	StartTaskIfSlotAvailable(taskID string) error
	Wake()
}

// Broadcaster is the minimal interface needed to emit websocket updates.
type Broadcaster interface {
	BroadcastUpdate(event string, payload any)
}

// Coordinator centralizes lifecycle application, persistence of caller-provided field edits,
// and execution of runtime side effects. This keeps transition logic in one place.
type Coordinator struct {
	LC          *Lifecycle
	Store       StorageAPI
	Runtime     RuntimeCoordinator
	Broadcaster Broadcaster
}

// ApplyOptions controls how ApplyTransition executes.
type ApplyOptions struct {
	// Mutate allows callers to update additional fields after lifecycle-managed status changes.
	Mutate func(*TaskItem)
	// BroadcastEvent emits this event with the updated task if set.
	BroadcastEvent string
	// ForceResave forces an additional persistence pass (e.g., when Mutate is nil but caller wants UpdatedAt bumped).
	ForceResave bool
	// SkipRuntimeEffects allows internal callers (e.g., processor) to bypass runtime hooks to avoid recursion.
	SkipRuntimeEffects bool
}

// ApplyTransition applies the lifecycle transition, executes runtime side effects, persists caller mutations,
// and emits an optional broadcast. It returns the updated task and lifecycle outcome.
func (c *Coordinator) ApplyTransition(req TransitionRequest, opts ApplyOptions) (*TaskItem, *TransitionOutcome, error) {
	if c == nil || c.LC == nil {
		return nil, nil, fmt.Errorf("coordinator unavailable")
	}

	outcome, err := c.LC.ApplyTransition(req)
	if err != nil {
		return nil, nil, err
	}
	if outcome == nil || outcome.Task == nil {
		return nil, outcome, fmt.Errorf("no task returned from transition")
	}

	task := outcome.Task

	// Apply caller-provided mutations after lifecycle-managed status fields are set.
	if opts.Mutate != nil {
		opts.Mutate(task)
		task.UpdatedAt = timeutil.NowRFC3339()
		if err := c.Store.SaveQueueItemSkipCleanup(*task, task.Status); err != nil {
			return nil, outcome, err
		}
	} else if opts.ForceResave {
		task.UpdatedAt = timeutil.NowRFC3339()
		if err := c.Store.SaveQueueItemSkipCleanup(*task, task.Status); err != nil {
			return nil, outcome, err
		}
	}

	if !opts.SkipRuntimeEffects {
		c.applyRuntimeEffects(task.ID, outcome.Effects)
	}

	if c.Broadcaster != nil && opts.BroadcastEvent != "" {
		c.Broadcaster.BroadcastUpdate(opts.BroadcastEvent, *task)
	}

	return task, outcome, nil
}

func (c *Coordinator) applyRuntimeEffects(taskID string, effects TransitionEffects) {
	if c.Runtime == nil {
		return
	}
	if effects.TerminateProcess {
		if err := c.Runtime.TerminateRunningProcess(taskID); err != nil {
			// Avoid failing caller flows for termination errors; log via systemlog when available.
		}
	}
	if effects.ForceStart {
		_ = c.Runtime.ForceStartTask(taskID, true)
	} else if effects.StartIfSlotAvailable {
		_ = c.Runtime.StartTaskIfSlotAvailable(taskID)
	}
	if effects.WakeProcessorAfterSave {
		c.Runtime.Wake()
	}
}
