package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
)

// TransitionContext captures caller intent and knobs for lifecycle operations.
type TransitionContext struct {
	Manual        bool // true when user manually moves a task (may bypass slots)
	ForceOverride bool // allow overriding auto-requeue lock for pending moves
	Now           func() time.Time
}

// TransitionRequest captures a desired status change.
type TransitionRequest struct {
	TaskID   string
	ToStatus string
	TransitionContext
}

// TransitionEffects describe the side effects callers should perform.
type TransitionEffects struct {
	TerminateProcess       bool // caller should terminate any running process for this task
	ForceStart             bool // caller should force start the task (manual move to active)
	StartIfSlotAvailable   bool // caller should start if capacity is available
	WakeProcessorAfterSave bool // caller should wake queue processor
}

// TransitionOutcome is the result of applying a transition.
type TransitionOutcome struct {
	Task    *TaskItem
	From    string
	Effects TransitionEffects
}

// Lifecycle bundles operations that mutate task state consistently.
type Lifecycle struct {
	Store StorageAPI
}

// Lifecycle rules (authoritative):
// - Manual movement intent wins for starting: moving to in-progress emits ForceStart even if slots are full; leaving in-progress emits TerminateProcess.
// - Terminal buckets (completed/failed/blocked/finalized/archived) always apply cooldown on entry; manual moves into terminal disable auto-requeue (blocked/finalized always disable).
// - Leaving terminal buckets re-enables auto-requeue, clears cooldown, and pending moves may opportunistically start if a slot exists.
// - Auto-requeue=false is a lock: pending moves are rejected unless leaving a terminal state or ForceOverride=true.
// - Recycler/automation only recycle from completed/failed with auto-requeue true and expired cooldown; they signal StartIfSlotAvailable for pending.
// - Side effects are surfaced (terminate, force start, wake), never executed here; callers must act on TransitionEffects.
//
// The goal: a single, declarative state machine that callers delegate to, rather than re-encoding rules in handlers/processors/recycler.

// StorageAPI is the minimal interface Lifecycle needs from storage.
type StorageAPI interface {
	MoveTaskTo(taskID, status string) (*TaskItem, string, error)
	SaveQueueItem(task TaskItem, status string) error
	SaveQueueItemSkipCleanup(task TaskItem, status string) error
	GetTaskByID(taskID string) (*TaskItem, string, error)
}

func defaultNow(ctx TransitionContext) time.Time {
	if ctx.Now != nil {
		return ctx.Now()
	}
	return time.Now()
}

func isTerminalStatus(status string) bool {
	switch status {
	case StatusCompleted, StatusFailed, StatusFailedBlocked, StatusCompletedFinalized, StatusArchived:
		return true
	default:
		return false
	}
}

// ApplyTransition enforces the task lifecycle rules and returns the updated task plus side effects.
// Canonical rules:
// 1) Manual intent wins for starting: moving to in-progress emits ForceStart (caller bypasses slots), and moving out of in-progress emits TerminateProcess.
// 2) Terminal buckets (completed/failed/blocked/finalized/archived) always apply cooldown and lock auto-requeue off on entry.
// 3) Leaving terminal buckets re-enables auto-requeue and clears cooldown; pending moves from terminal may be started immediately if capacity exists.
// 4) Auto-requeue=false is a lock: pending moves are rejected unless leaving a terminal state or ForceOverride=true.
// 5) Recycler/automation only recycles from completed/failed and must respect cooldown + auto-requeue; Pending starts are signaled via StartIfSlotAvailable.
// 6) Side effects are surfaced, never executed here: callers must terminate running processes, force-start, or wake processors per the returned flags.
func (lc *Lifecycle) ApplyTransition(req TransitionRequest) (*TransitionOutcome, error) {
	if lc == nil || lc.Store == nil {
		return nil, fmt.Errorf("lifecycle store unavailable")
	}
	task, fromStatus, err := lc.Store.GetTaskByID(req.TaskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task %s not found", req.TaskID)
	}

	toStatus := strings.TrimSpace(req.ToStatus)
	if toStatus == "" {
		return &TransitionOutcome{Task: task, From: fromStatus}, nil
	}

	now := defaultNow(req.TransitionContext).Format(time.RFC3339)
	effects := TransitionEffects{}

	// No-op transition, just update timestamp.
	if toStatus == fromStatus {
		task.UpdatedAt = now
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
			return nil, err
		}
		return &TransitionOutcome{Task: task, From: fromStatus}, nil
	}

	switch toStatus {
	case StatusInProgress:
		// Leaving finished/blocked re-enables auto-requeue and clears cooldown per manual rule.
		if isTerminalStatus(fromStatus) {
			task.ProcessorAutoRequeue = true
			task.CooldownUntil = ""
		}
		task.Status = StatusInProgress
		task.CurrentPhase = StatusInProgress
		task.Results = nil
		task.CompletedAt = ""
		if task.StartedAt == "" {
			task.StartedAt = now
		}
		task.UpdatedAt = now

		if _, _, err := lc.Store.MoveTaskTo(req.TaskID, StatusInProgress); err != nil {
			return nil, err
		}
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusInProgress); err != nil {
			return nil, err
		}

		if req.Manual {
			effects.ForceStart = true
		} else {
			effects.StartIfSlotAvailable = true
		}
		effects.WakeProcessorAfterSave = true

	case StatusPending:
		// Enforce lock unless leaving terminal state (which re-enables auto-requeue) or explicitly overridden.
		if !isTerminalStatus(fromStatus) && !task.ProcessorAutoRequeue && !req.ForceOverride {
			return nil, fmt.Errorf("auto-requeue disabled; cannot move %s to pending", req.TaskID)
		}
		if isTerminalStatus(fromStatus) {
			if remaining := cooldownRemaining(task); remaining > 0 && !req.ForceOverride {
				return nil, fmt.Errorf("task %s still cooling down (%v remaining)", req.TaskID, remaining)
			}
			task.ProcessorAutoRequeue = true
			task.CooldownUntil = ""
		} else if remaining := cooldownRemaining(task); remaining > 0 && !req.ForceOverride {
			return nil, fmt.Errorf("task %s still cooling down (%v remaining)", req.TaskID, remaining)
		}

		if fromStatus == StatusInProgress {
			effects.TerminateProcess = true
		}
		task.Status = StatusPending
		task.CurrentPhase = ""
		task.StartedAt = ""
		task.CompletedAt = ""
		task.Results = nil
		task.ConsecutiveCompletionClaims = 0
		task.ConsecutiveFailures = 0
		task.UpdatedAt = now

		if _, _, err := lc.Store.MoveTaskTo(req.TaskID, StatusPending); err != nil {
			return nil, err
		}
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusPending); err != nil {
			return nil, err
		}

		effects.StartIfSlotAvailable = true
		effects.WakeProcessorAfterSave = true

	case StatusCompleted, StatusFailed, StatusFailedBlocked, StatusCompletedFinalized:
		if fromStatus == StatusInProgress {
			effects.TerminateProcess = true
		}

		task.Status = toStatus
		task.CurrentPhase = toStatus
		if task.CompletedAt == "" {
			task.CompletedAt = now
		}
		if task.CompletionCount <= 0 {
			task.CompletionCount = 1
		} else {
			task.CompletionCount++
		}
		task.LastCompletedAt = task.CompletedAt

		// Apply cooldown consistently for manual moves too.
		cooldownSeconds := settings.GetSettings().CooldownSeconds
		if cooldownSeconds > 0 {
			task.CooldownUntil = defaultNow(req.TransitionContext).Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
		} else {
			task.CooldownUntil = ""
		}

		// Manual moves into finished/blocked disable auto-requeue; blocked/finalized are always locked.
		if req.Manual || toStatus == StatusFailedBlocked || toStatus == StatusCompletedFinalized {
			task.ProcessorAutoRequeue = false
		}

		task.UpdatedAt = now

		if _, _, err := lc.Store.MoveTaskTo(req.TaskID, toStatus); err != nil {
			return nil, err
		}
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
			return nil, err
		}

	case StatusArchived:
		if fromStatus == StatusInProgress {
			effects.TerminateProcess = true
		}
		task.Status = StatusArchived
		task.CurrentPhase = StatusArchived
		task.ProcessorAutoRequeue = false
		task.UpdatedAt = now
		if _, _, err := lc.Store.MoveTaskTo(req.TaskID, StatusArchived); err != nil {
			return nil, err
		}
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusArchived); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported transition to %s", toStatus)
	}

	return &TransitionOutcome{
		Task:    task,
		From:    fromStatus,
		Effects: effects,
	}, nil
}

// StartPending moves a pending task into in-progress and clears execution fields.
// Caller is responsible for launching the actual execution.
func (lc *Lifecycle) StartPending(taskID string, ctx TransitionContext) (*TaskItem, error) {
	if lc == nil || lc.Store == nil {
		return nil, fmt.Errorf("lifecycle store unavailable")
	}

	task, status, err := lc.Store.GetTaskByID(taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	if status != StatusPending {
		return nil, fmt.Errorf("task %s not pending (status=%s)", taskID, status)
	}
	if !task.ProcessorAutoRequeue && !ctx.ForceOverride {
		return nil, fmt.Errorf("auto-requeue disabled; cannot start task %s", taskID)
	}

	updated, _, err := lc.Store.MoveTaskTo(taskID, StatusInProgress)
	if err != nil {
		return nil, err
	}
	if updated != nil {
		task = updated
	}

	now := defaultNow(ctx).Format(time.RFC3339)
	task.Status = StatusInProgress
	task.CurrentPhase = StatusInProgress
	task.Results = nil
	task.CompletedAt = ""
	if task.StartedAt == "" {
		task.StartedAt = now
	}
	task.UpdatedAt = now

	if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusInProgress); err != nil {
		return nil, err
	}
	return task, nil
}

// StopActive moves an in-progress task out of active, sets cancellation info, and disables auto-requeue lock only per rules above.
func (lc *Lifecycle) StopActive(taskID, toStatus string, ctx TransitionContext) (*TaskItem, string, error) {
	if lc == nil || lc.Store == nil {
		return nil, "", fmt.Errorf("lifecycle store unavailable")
	}
	if toStatus == "" {
		toStatus = StatusPending
	}

	task, status, err := lc.Store.GetTaskByID(taskID)
	if err != nil {
		return nil, "", err
	}
	if task == nil {
		return nil, "", fmt.Errorf("task %s not found", taskID)
	}
	if status != StatusInProgress {
		return task, status, nil
	}

	now := defaultNow(ctx).Format(time.RFC3339)
	task.Status = toStatus
	task.CurrentPhase = "cancelled"
	task.Results = map[string]any{
		"success":      false,
		"error":        fmt.Sprintf("Task execution was cancelled (moved to %s)", toStatus),
		"cancelled_at": now,
	}
	task.UpdatedAt = now
	task.StartedAt = ""
	task.CompletedAt = ""
	task.CooldownUntil = ""
	task.ConsecutiveCompletionClaims = 0
	task.ConsecutiveFailures = 0
	if toStatus == StatusPending && task.ProcessorAutoRequeue {
		// Stay enabled
	} else if toStatus == StatusPending {
		// Respect lock
	} else {
		task.ProcessorAutoRequeue = false
	}

	if _, _, err := lc.Store.MoveTaskTo(taskID, toStatus); err != nil {
		return nil, "", err
	}
	if err := lc.Store.SaveQueueItem(*task, toStatus); err != nil {
		return nil, "", err
	}
	return task, status, nil
}

// Complete marks a task as completed, applies cooldown, and disables auto-requeue when manual.
func (lc *Lifecycle) Complete(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	return lc.finish(taskID, StatusCompleted, ctx)
}

// Fail marks a task as failed, applies cooldown, and disables auto-requeue when manual blocked.
func (lc *Lifecycle) Fail(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	return lc.finish(taskID, StatusFailed, ctx)
}

// Block marks a task as failed-blocked with cooldown and lock.
func (lc *Lifecycle) Block(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	return lc.finish(taskID, StatusFailedBlocked, ctx)
}

// Finalize marks a task as completed-finalized and locks auto-requeue off.
func (lc *Lifecycle) Finalize(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	return lc.finish(taskID, StatusCompletedFinalized, ctx)
}

// Recycle moves a completed/failed task back to pending if allowed (auto-requeue true, cooldown expired).
func (lc *Lifecycle) Recycle(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:            taskID,
		ToStatus:          StatusPending,
		TransitionContext: ctx,
	})
	if err != nil {
		return nil, "", err
	}
	return outcome.Task, outcome.From, nil
}

// finish applies terminal transitions and cooldown behavior.
func (lc *Lifecycle) finish(taskID, toStatus string, ctx TransitionContext) (*TaskItem, string, error) {
	if lc == nil || lc.Store == nil {
		return nil, "", fmt.Errorf("lifecycle store unavailable")
	}

	task, status, err := lc.Store.GetTaskByID(taskID)
	if err != nil {
		return nil, "", err
	}
	if task == nil {
		return nil, "", fmt.Errorf("task %s not found", taskID)
	}

	now := defaultNow(ctx).Format(time.RFC3339)
	task.Status = toStatus
	task.CurrentPhase = toStatus
	task.CompletedAt = now
	if task.CompletionCount <= 0 {
		task.CompletionCount = 1
	} else {
		task.CompletionCount++
	}
	task.LastCompletedAt = now

	// Apply cooldown regardless of manual/auto; manual moves also get cooldown.
	cooldownSeconds := settings.GetSettings().CooldownSeconds
	if cooldownSeconds > 0 {
		task.CooldownUntil = defaultNow(ctx).Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
	} else {
		task.CooldownUntil = ""
	}

	// Lock off auto-requeue when entering terminal states unless automated continuation later re-enables it explicitly.
	if toStatus == StatusCompleted || toStatus == StatusFailed || toStatus == StatusFailedBlocked || toStatus == StatusCompletedFinalized {
		task.ProcessorAutoRequeue = false
	}

	task.UpdatedAt = now

	if _, _, err := lc.Store.MoveTaskTo(taskID, toStatus); err != nil {
		return nil, status, err
	}
	if err := lc.Store.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
		return nil, status, err
	}
	return task, status, nil
}

// cooldownRemaining is duplicated from recycler for now to keep lifecycle self-contained.
func cooldownRemaining(task *TaskItem) time.Duration {
	if task == nil {
		return 0
	}
	if strings.TrimSpace(task.CooldownUntil) == "" {
		return 0
	}
	ts, err := time.Parse(time.RFC3339, task.CooldownUntil)
	if err != nil {
		return 0
	}
	remaining := time.Until(ts)
	if remaining <= 0 {
		return 0
	}
	return remaining
}
