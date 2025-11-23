package queue

import (
	"fmt"
	"log"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// AutoSteerIntegration handles Auto Steer integration with the task processor
type AutoSteerIntegration struct {
	executionEngine *autosteer.ExecutionEngine
}

// NewAutoSteerIntegration creates a new Auto Steer integration handler
func NewAutoSteerIntegration(executionEngine *autosteer.ExecutionEngine) *AutoSteerIntegration {
	return &AutoSteerIntegration{
		executionEngine: executionEngine,
	}
}

// InitializeAutoSteer initializes Auto Steer execution for a task if needed
// Should be called before executing a task for the first time
func (a *AutoSteerIntegration) InitializeAutoSteer(task *tasks.TaskItem, scenarioName string) error {
	if task.AutoSteerProfileID == "" {
		return nil // No Auto Steer profile configured
	}

	// Check if already initialized
	existingState, err := a.executionEngine.GetExecutionState(task.ID)
	if err != nil {
		return fmt.Errorf("failed to check Auto Steer state: %w", err)
	}

	if existingState != nil {
		// Already initialized
		log.Printf("Auto Steer already initialized for task %s (profile: %s, phase: %d/%d)",
			task.ID, task.AutoSteerProfileID, existingState.CurrentPhaseIndex+1, len(existingState.PhaseHistory)+1)
		return nil
	}

	// Initialize new execution
	log.Printf("Initializing Auto Steer for task %s with profile %s", task.ID, task.AutoSteerProfileID)
	systemlog.Infof("Auto Steer: Initializing task %s with profile %s for scenario %s",
		task.ID, task.AutoSteerProfileID, scenarioName)

	state, err := a.executionEngine.StartExecution(task.ID, task.AutoSteerProfileID, scenarioName)
	if err != nil {
		return fmt.Errorf("failed to start Auto Steer execution: %w", err)
	}

	log.Printf("Auto Steer initialized successfully for task %s - starting in %s mode",
		task.ID, state.Metrics.Loops)
	systemlog.Infof("Auto Steer: Task %s initialized - Phase 1: %s mode",
		task.ID, state.Metrics.Loops)

	return nil
}

// EnhancePrompt adds Auto Steer context to the task prompt
// Returns the enhanced prompt or the original prompt if no Auto Steer is active
func (a *AutoSteerIntegration) EnhancePrompt(task *tasks.TaskItem, basePrompt string) (string, error) {
	if task.AutoSteerProfileID == "" {
		return basePrompt, nil // No enhancement needed
	}

	// Get Auto Steer prompt section
	autoSteerSection, err := a.executionEngine.GetEnhancedPrompt(task.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get Auto Steer prompt enhancement: %w", err)
	}

	if autoSteerSection == "" {
		// No active Auto Steer state (shouldn't happen if profile is configured)
		log.Printf("Warning: Task %s has Auto Steer profile but no active state", task.ID)
		return basePrompt, nil
	}

	// Append Auto Steer section to prompt
	enhancedPrompt := basePrompt + "\n\n" + autoSteerSection

	// Log for debugging
	currentMode, _ := a.executionEngine.GetCurrentMode(task.ID)
	log.Printf("Enhanced prompt with Auto Steer (%s mode) for task %s", currentMode, task.ID)

	return enhancedPrompt, nil
}

// EvaluateIteration evaluates the current iteration and determines next steps
// Should be called after successful task execution
// Returns: shouldContinue, needsPhaseAdvance, error
func (a *AutoSteerIntegration) EvaluateIteration(task *tasks.TaskItem, scenarioName string) (bool, bool, error) {
	if task.AutoSteerProfileID == "" {
		// No Auto Steer - task should continue normally based on ProcessorAutoRequeue
		return true, false, nil
	}

	// Use CompletionCount as the loop counter
	loops := task.CompletionCount

	log.Printf("Evaluating Auto Steer iteration for task %s (loop %d)", task.ID, loops)

	// Evaluate iteration
	evaluation, err := a.executionEngine.EvaluateIteration(task.ID, scenarioName, loops)
	if err != nil {
		return false, false, fmt.Errorf("failed to evaluate Auto Steer iteration: %w", err)
	}

	if !evaluation.ShouldStop {
		// Continue in current phase
		log.Printf("Auto Steer: Task %s continuing in current phase (loop %d)", task.ID, loops)
		return true, false, nil
	}

	// Phase should stop
	log.Printf("Auto Steer: Task %s phase stopping (reason: %s)", task.ID, evaluation.Reason)
	systemlog.Infof("Auto Steer: Task %s phase complete - reason: %s", task.ID, evaluation.Reason)

	return false, true, nil
}

// AdvancePhase advances to the next Auto Steer phase
// Returns: allPhasesComplete, error
func (a *AutoSteerIntegration) AdvancePhase(task *tasks.TaskItem, scenarioName string) (bool, error) {
	log.Printf("Advancing Auto Steer phase for task %s", task.ID)
	systemlog.Infof("Auto Steer: Advancing phase for task %s", task.ID)

	result, err := a.executionEngine.AdvancePhase(task.ID, scenarioName)
	if err != nil {
		return false, fmt.Errorf("failed to advance Auto Steer phase: %w", err)
	}

	if !result.Success {
		return false, fmt.Errorf("phase advance failed: %s", result.Message)
	}

	if result.Completed {
		// All phases completed
		log.Printf("Auto Steer: All phases completed for task %s", task.ID)
		systemlog.Infof("Auto Steer: Task %s completed all phases successfully", task.ID)
		return true, nil
	}

	// Advanced to next phase
	log.Printf("Auto Steer: Task %s advanced to phase %d", task.ID, result.NextPhaseIndex+1)
	systemlog.Infof("Auto Steer: Task %s advanced to phase %d", task.ID, result.NextPhaseIndex+1)

	return false, nil
}

// ShouldContinueTask determines if a task should continue (requeue) after execution
// Takes into account Auto Steer state and phase progression
func (a *AutoSteerIntegration) ShouldContinueTask(task *tasks.TaskItem, scenarioName string) (bool, error) {
	if task.AutoSteerProfileID == "" {
		// No Auto Steer - use normal ProcessorAutoRequeue behavior
		return task.ProcessorAutoRequeue, nil
	}

	// Evaluate the current iteration
	shouldContinue, needsPhaseAdvance, err := a.EvaluateIteration(task, scenarioName)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate iteration: %w", err)
	}

	if shouldContinue && !needsPhaseAdvance {
		// Continue in current phase
		return true, nil
	}

	if needsPhaseAdvance {
		// Advance to next phase
		allComplete, err := a.AdvancePhase(task, scenarioName)
		if err != nil {
			return false, fmt.Errorf("failed to advance phase: %w", err)
		}

		if allComplete {
			// All phases done - don't continue
			log.Printf("Auto Steer: Task %s fully completed - will not requeue", task.ID)
			return false, nil
		}

		// Advanced to next phase - continue
		return true, nil
	}

	// Shouldn't get here, but default to not continuing
	return false, nil
}

// GetCurrentMode returns the current Auto Steer mode for a task
func (a *AutoSteerIntegration) GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error) {
	if task.AutoSteerProfileID == "" {
		return "", nil
	}

	return a.executionEngine.GetCurrentMode(task.ID)
}

// IsActive checks if Auto Steer is active for a task
func (a *AutoSteerIntegration) IsActive(task *tasks.TaskItem) bool {
	if task.AutoSteerProfileID == "" {
		return false
	}

	active, err := a.executionEngine.IsAutoSteerActive(task.ID)
	if err != nil {
		log.Printf("Warning: Failed to check Auto Steer status for task %s: %v", task.ID, err)
		return false
	}

	return active
}
