package tasks

import (
	"fmt"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/settings"
)

// TransitionIntent encodes who/what is initiating a move so rules are consistent.
type TransitionIntent string

const (
	IntentAuto      TransitionIntent = "auto"      // scheduler/processor
	IntentManual    TransitionIntent = "manual"    // user-initiated move
	IntentRecycler  TransitionIntent = "recycler"  // background recycler
	IntentReconcile TransitionIntent = "reconcile" // safety recovery that can override locks/cooldowns
)

// TransitionContext captures caller intent and knobs for lifecycle operations.
type TransitionContext struct {
	Intent TransitionIntent
	Now    func() time.Time
}

func (ctx TransitionContext) intent() TransitionIntent {
	if ctx.Intent != "" {
		return ctx.Intent
	}
	return IntentAuto
}

func (ctx TransitionContext) isManual() bool {
	return ctx.intent() == IntentManual
}

func (ctx TransitionContext) allowsLockOverride() bool {
	return ctx.intent() == IntentReconcile
}

func (ctx TransitionContext) allowsCooldownOverride() bool {
	return ctx.allowsLockOverride()
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
	ResetSteeringQueue     bool // caller should reset steering queue position to 0
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

// transitionRule expresses an allowed destination and its mutation/side effects.
type transitionRule struct {
	allowedFrom map[string]struct{}
	apply       func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error)
}

// Lifecycle rules (authoritative):
// - Manual movement intent wins for starting: moving to in-progress emits ForceStart even if slots are full; leaving in-progress emits TerminateProcess.
// - Terminal buckets (completed/failed/blocked/finalized/archived) always apply cooldown on entry; auto-requeue is disabled for manual moves and hard-stop buckets.
// - Leaving terminal buckets re-enables auto-requeue, clears cooldown, and pending moves may opportunistically start if a slot exists.
// - Auto-requeue=false is a lock: pending moves are rejected unless leaving a terminal state or using a reconcile intent (lock override).
// - Recycler/automation only recycle from completed/failed with auto-requeue true and expired cooldown; Pending starts are signaled via StartIfSlotAvailable.
// - Side effects are surfaced (terminate, force start, wake), never executed here; callers must act on TransitionEffects.
//
// The goal: a single, declarative state machine that callers delegate to, rather than re-encoding rules in handlers/processors/recycler.

// StorageAPI abstracts task storage for testability.
// Used by Lifecycle, Processor, Recycler, and handlers.
type StorageAPI interface {
	// Core CRUD operations
	GetTaskByID(taskID string) (*TaskItem, string, error)
	SaveQueueItem(task TaskItem, status string) error
	SaveQueueItemSkipCleanup(task TaskItem, status string) error
	MoveTaskTo(taskID, status string) (*TaskItem, string, error)

	// Query operations
	GetQueueItems(status string) ([]TaskItem, error)
	CurrentStatus(taskID string) (string, error)
	FindActiveTargetTask(taskType, operation, target string) (*TaskItem, string, error)

	// Maintenance operations
	CleanupDuplicates() error
	DeleteTask(taskID string) (string, error)

	// Configuration
	GetQueueDir() string
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
// 2) Terminal buckets (completed/failed/blocked/finalized/archived) always apply cooldown; auto-requeue is disabled on manual moves and hard-stop buckets.
// 3) Leaving terminal buckets re-enables auto-requeue and clears cooldown; pending moves from terminal may be started immediately if capacity exists.
// 4) Auto-requeue=false is a lock: pending moves are rejected unless leaving a terminal state or using a reconcile intent.
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

	// No-op transition, just update timestamp.
	if toStatus == fromStatus {
		task.UpdatedAt = now
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
			return nil, err
		}
		return &TransitionOutcome{Task: task, From: fromStatus}, nil
	}

	rules := lifecycleRules(lc)
	rule, ok := rules[toStatus]
	if !ok {
		return nil, newTransitionError(TransitionErrorCodeUnsupported, "unsupported transition to %s", toStatus)
	}
	if len(rule.allowedFrom) > 0 {
		if _, allowed := rule.allowedFrom[fromStatus]; !allowed {
			return nil, newTransitionError(TransitionErrorCodeConflict, "cannot move %s from %s", toStatus, fromStatus)
		}
	}

	effects, err := rule.apply(lc, task, fromStatus, req, now)
	if err != nil {
		return nil, err
	}

	return &TransitionOutcome{
		Task:    task,
		From:    fromStatus,
		Effects: effects,
	}, nil
}

func lifecycleRules(lc *Lifecycle) map[string]transitionRule {
	return map[string]transitionRule{
		StatusInProgress: {
			apply: func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error) {
				effects := TransitionEffects{}

				// Leaving terminal buckets lifts the lock and clears cooldown.
				if isTerminalStatus(fromStatus) {
					task.ProcessorAutoRequeue = true
					task.CooldownUntil = ""
					effects.ResetSteeringQueue = true
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
					return effects, err
				}
				if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusInProgress); err != nil {
					return effects, err
				}

				if req.isManual() {
					effects.ForceStart = true
				} else {
					effects.StartIfSlotAvailable = true
				}
				effects.WakeProcessorAfterSave = true

				return effects, nil
			},
		},
		StatusPending: {
			apply: func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error) {
				effects := TransitionEffects{}

				// Enforce lock unless leaving terminal state (which re-enables auto-requeue) or explicitly overridden.
				if !isTerminalStatus(fromStatus) && !task.ProcessorAutoRequeue && !req.allowsLockOverride() {
					return effects, newTransitionError(TransitionErrorCodeConflict, "auto-requeue disabled; cannot move %s to pending", req.TaskID)
				}
				if remaining := CooldownRemaining(task); remaining > 0 && !req.allowsCooldownOverride() {
					return effects, newTransitionError(TransitionErrorCodeConflict, "task %s still cooling down (%v remaining)", req.TaskID, remaining)
				}
				if isTerminalStatus(fromStatus) {
					task.ProcessorAutoRequeue = true
					task.CooldownUntil = ""
					effects.ResetSteeringQueue = true
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
					return effects, err
				}
				if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusPending); err != nil {
					return effects, err
				}

				effects.StartIfSlotAvailable = true
				effects.WakeProcessorAfterSave = true
				return effects, nil
			},
		},
		StatusCompleted:          {apply: finalizeRule(StatusCompleted)},
		StatusFailed:             {apply: finalizeRule(StatusFailed)},
		StatusFailedBlocked:      {apply: finalizeRule(StatusFailedBlocked)},
		StatusCompletedFinalized: {apply: finalizeRule(StatusCompletedFinalized)},
		StatusArchived: {
			apply: func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error) {
				effects := TransitionEffects{}
				if fromStatus == StatusInProgress {
					effects.TerminateProcess = true
				}
				task.Status = StatusArchived
				task.CurrentPhase = StatusArchived
				task.ProcessorAutoRequeue = false
				task.UpdatedAt = now
				if _, _, err := lc.Store.MoveTaskTo(req.TaskID, StatusArchived); err != nil {
					return effects, err
				}
				if err := lc.Store.SaveQueueItemSkipCleanup(*task, StatusArchived); err != nil {
					return effects, err
				}
				return effects, nil
			},
		},
	}
}

func finalizeRule(toStatus string) func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error) {
	return func(lc *Lifecycle, task *TaskItem, fromStatus string, req TransitionRequest, now string) (TransitionEffects, error) {
		effects := TransitionEffects{}
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

		cooldownSeconds := settings.GetSettings().CooldownSeconds
		if cooldownSeconds > 0 {
			task.CooldownUntil = defaultNow(req.TransitionContext).Add(time.Duration(cooldownSeconds) * time.Second).Format(time.RFC3339)
		} else {
			task.CooldownUntil = ""
		}

		// Moving into terminal buckets disables auto-requeue when manual intent is present,
		// or when entering hard-stop buckets.
		if req.isManual() || toStatus == StatusFailedBlocked || toStatus == StatusCompletedFinalized {
			task.ProcessorAutoRequeue = false
		}
		task.UpdatedAt = now

		if _, _, err := lc.Store.MoveTaskTo(req.TaskID, toStatus); err != nil {
			return effects, err
		}
		if err := lc.Store.SaveQueueItemSkipCleanup(*task, toStatus); err != nil {
			return effects, err
		}
		return effects, nil
	}
}


// Complete marks a task as completed using canonical rules.
func (lc *Lifecycle) Complete(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:            taskID,
		ToStatus:          StatusCompleted,
		TransitionContext: ctx,
	})
	if err != nil {
		return nil, "", err
	}
	return outcome.Task, outcome.From, nil
}

// Fail marks a task as failed using canonical rules.
func (lc *Lifecycle) Fail(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:            taskID,
		ToStatus:          StatusFailed,
		TransitionContext: ctx,
	})
	if err != nil {
		return nil, "", err
	}
	return outcome.Task, outcome.From, nil
}

// Block marks a task as failed-blocked using canonical rules.
func (lc *Lifecycle) Block(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:            taskID,
		ToStatus:          StatusFailedBlocked,
		TransitionContext: ctx,
	})
	if err != nil {
		return nil, "", err
	}
	return outcome.Task, outcome.From, nil
}

// Finalize marks a task as completed-finalized using canonical rules.
func (lc *Lifecycle) Finalize(taskID string, ctx TransitionContext) (*TaskItem, string, error) {
	outcome, err := lc.ApplyTransition(TransitionRequest{
		TaskID:            taskID,
		ToStatus:          StatusCompletedFinalized,
		TransitionContext: ctx,
	})
	if err != nil {
		return nil, "", err
	}
	return outcome.Task, outcome.From, nil
}

// Recycle moves a task back to pending via canonical rules.
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

