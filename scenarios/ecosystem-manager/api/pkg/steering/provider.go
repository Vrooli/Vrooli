package steering

import (
	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// SteeringProvider is the unified interface that all steering strategies implement.
// It abstracts the decision-making for prompt enhancement and task continuation.
type SteeringProvider interface {
	// Strategy returns which steering strategy this provider implements.
	Strategy() SteeringStrategy

	// GetCurrentMode returns the mode to use for the current execution.
	// Returns empty string if the strategy has no mode to provide.
	GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error)

	// EnhancePrompt generates the steering content to inject into the agent prompt.
	// Returns nil if no enhancement is needed.
	EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error)

	// AfterExecution is called after a task execution completes successfully.
	// It returns a decision about whether to continue/requeue the task.
	AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error)

	// Initialize sets up any state needed before the first execution.
	// For stateless strategies (manual, none), this may be a no-op.
	Initialize(task *tasks.TaskItem) error

	// Reset clears any persisted state for the task.
	// Used when a task is deleted or needs to start fresh.
	Reset(taskID string) error
}

// RegistryAPI provides access to steering providers based on task configuration.
type RegistryAPI interface {
	// GetProvider returns the appropriate steering provider for a task.
	GetProvider(task *tasks.TaskItem) SteeringProvider

	// DetermineStrategy inspects task fields to determine which strategy applies.
	DetermineStrategy(task *tasks.TaskItem) SteeringStrategy
}
