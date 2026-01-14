package steering

import (
	"github.com/ecosystem-manager/api/pkg/autosteer"
)

// SteeringStrategy identifies which strategy controls a task's steering behavior.
type SteeringStrategy string

const (
	// StrategyManual uses a single, manually-selected mode that repeats indefinitely.
	StrategyManual SteeringStrategy = "manual"

	// StrategyNone applies no explicit steering, defaulting to Progress mode.
	StrategyNone SteeringStrategy = "none"

	// StrategyProfile uses an Auto Steer profile with phases, stop conditions, and metrics.
	StrategyProfile SteeringStrategy = "profile"

	// StrategyQueue uses an ordered list of modes, executing each once before advancing.
	StrategyQueue SteeringStrategy = "queue"
)

// SteeringDecision represents the outcome of a steering provider's evaluation
// after a task execution completes.
type SteeringDecision struct {
	// Mode is the current (or next) steering mode to apply.
	Mode autosteer.SteerMode

	// ShouldRequeue indicates whether the task should be requeued for another iteration.
	// This is combined with ProcessorAutoRequeue on the task - both must be true to requeue.
	ShouldRequeue bool

	// Exhausted is true when the steering strategy has completed all its iterations
	// (e.g., queue is empty, all profile phases done).
	Exhausted bool

	// Reason provides a human-readable explanation for the decision.
	Reason string
}

// PromptEnhancement contains the steering content to inject into agent prompts.
type PromptEnhancement struct {
	// Section is the markdown/text content to inject into the prompt.
	Section string

	// Source identifies where the enhancement came from for logging purposes.
	// Examples: "manual:progress", "queue:test[2/5]", "profile:mvp-polish"
	Source string
}

// SteeringConfig carries strategy-specific configuration for initializing a provider.
type SteeringConfig struct {
	// Strategy indicates which steering strategy to use.
	Strategy SteeringStrategy

	// Mode is the steering mode for StrategyManual.
	Mode autosteer.SteerMode

	// ProfileID is the Auto Steer profile ID for StrategyProfile.
	ProfileID string

	// Queue is the ordered list of modes for StrategyQueue.
	Queue []autosteer.SteerMode
}

// QueueState tracks progress through a steering queue.
type QueueState struct {
	// TaskID is the unique identifier for the task.
	TaskID string `json:"task_id"`

	// Queue is the ordered list of steering modes.
	Queue []autosteer.SteerMode `json:"queue"`

	// CurrentIndex is the position in the queue (0-indexed).
	CurrentIndex int `json:"current_index"`

	// CreatedAt is when the queue state was first created.
	CreatedAt string `json:"created_at"`

	// UpdatedAt is when the queue state was last modified.
	UpdatedAt string `json:"updated_at"`
}

// IsExhausted returns true if the queue has been fully processed.
func (q *QueueState) IsExhausted() bool {
	return q.CurrentIndex >= len(q.Queue)
}

// CurrentMode returns the current mode in the queue, or empty string if exhausted.
func (q *QueueState) CurrentMode() autosteer.SteerMode {
	if q.IsExhausted() {
		return ""
	}
	return q.Queue[q.CurrentIndex]
}

// Remaining returns the number of items left in the queue (including current).
func (q *QueueState) Remaining() int {
	remaining := len(q.Queue) - q.CurrentIndex
	if remaining < 0 {
		return 0
	}
	return remaining
}
