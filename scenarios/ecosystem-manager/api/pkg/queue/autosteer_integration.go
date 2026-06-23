package queue

import (
	"fmt"
	"log"
	"sync"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/systemlog"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// AutoSteerIntegration handles Auto Steer integration with the task processor
type AutoSteerIntegration struct {
	executionOrchestrator *autosteer.ExecutionOrchestrator

	// Baseline Modes engagement (plan P6 §192). baselineRunner shells
	// git-control-tower; engagements holds per-task engagement state between
	// EvaluateStart and the controller's terminal decision. selfScenario is the
	// loop's own scenario, which is never engaged (self-promote is externalized).
	baselineRunner BaselineEngagementRunner
	selfScenario   string
	engMu          sync.Mutex
	engagements    map[string]*taskEngagement
}

// NewAutoSteerIntegration creates a new Auto Steer integration handler. projectRoot
// is the repo root the default git-control-tower baseline runner shells from.
func NewAutoSteerIntegration(executionOrchestrator *autosteer.ExecutionOrchestrator, projectRoot string) *AutoSteerIntegration {
	return &AutoSteerIntegration{
		executionOrchestrator: executionOrchestrator,
		baselineRunner:        &GCTBaselineRunner{ProjectRoot: projectRoot},
		selfScenario:          defaultSelfScenario,
		engagements:           make(map[string]*taskEngagement),
	}
}

// SetBaselineRunner overrides the Baseline Modes engagement runner (test seam).
func (a *AutoSteerIntegration) SetBaselineRunner(r BaselineEngagementRunner) *AutoSteerIntegration {
	a.baselineRunner = r
	return a
}

// isEligible constrains Auto Steer usage to the task shapes it was designed for.
func (a *AutoSteerIntegration) isEligible(task *tasks.TaskItem) bool {
	if task == nil {
		return false
	}
	return task.AutoSteerProfileID != "" && task.Type == "scenario" && task.Operation == "improver"
}

// ExecutionOrchestrator exposes the underlying execution orchestrator for advanced workflows.
func (a *AutoSteerIntegration) ExecutionOrchestrator() *autosteer.ExecutionOrchestrator {
	return a.executionOrchestrator
}

// InitializeAutoSteer initializes Auto Steer execution for a task if needed
// Should be called before executing a task for the first time
func (a *AutoSteerIntegration) InitializeAutoSteer(task *tasks.TaskItem, scenarioName string) error {
	if !a.isEligible(task) {
		return nil // No Auto Steer profile configured
	}

	// Check if already initialized
	existingState, err := a.executionOrchestrator.GetExecutionState(task.ID)
	if err != nil {
		return fmt.Errorf("failed to check Auto Steer state: %w", err)
	}

	if existingState != nil {
		// Check if the profile has changed since the execution was initialized.
		// If the task's profile ID differs from the persisted state, the user
		// switched profiles between runs — reset and re-initialize so the new
		// profile's phases and conditions take effect.
		if existingState.ProfileID != task.AutoSteerProfileID {
			log.Printf("Auto Steer: Profile changed for task %s (%s → %s); resetting execution state",
				task.ID, existingState.ProfileID, task.AutoSteerProfileID)
			systemlog.Infof("Auto Steer: Resetting task %s execution state due to profile change (%s → %s)",
				task.ID, existingState.ProfileID, task.AutoSteerProfileID)

			if err := a.executionOrchestrator.DeleteExecutionState(task.ID); err != nil {
				return fmt.Errorf("failed to delete stale Auto Steer state: %w", err)
			}
			// Fall through to create a fresh execution below.
		} else {
			// Already initialized with the correct profile.
			log.Printf("Auto Steer already initialized for task %s (profile: %s, iteration: %d, skill: %s)",
				task.ID, task.AutoSteerProfileID, existingState.Iteration, existingState.CurrentSkill)
			return nil
		}
	}

	// Initialize new execution
	log.Printf("Initializing Auto Steer for task %s with profile %s", task.ID, task.AutoSteerProfileID)
	systemlog.Infof("Auto Steer: Initializing task %s with profile %s for scenario %s",
		task.ID, task.AutoSteerProfileID, scenarioName)

	state, err := a.executionOrchestrator.StartExecution(task.ID, task.AutoSteerProfileID, scenarioName)
	if err != nil {
		return fmt.Errorf("failed to start Auto Steer execution: %w", err)
	}

	log.Printf("Auto Steer initialized successfully for task %s - starting at iteration %d",
		task.ID, state.Iteration)
	systemlog.Infof("Auto Steer: Task %s initialized - iteration: %d",
		task.ID, state.Iteration)

	return nil
}

// EvaluateStart asks the controller whether the first agent run is warranted,
// after InitializeAutoSteer has run the initial DIAGNOSE + SELECT. It returns
// proceed=false (with the controller's halt reason) when the objective is
// already met or there is nothing to steer — in which case the caller should
// finalize the task instead of launching a blind agent pass. Tasks without an
// Auto Steer profile always proceed.
func (a *AutoSteerIntegration) EvaluateStart(task *tasks.TaskItem, scenarioName string) (proceed bool, reason string, err error) {
	if !a.isEligible(task) {
		return true, "", nil
	}
	proceed, reason, err = a.executionOrchestrator.EvaluateStart(task.ID, scenarioName)
	if err == nil && proceed {
		// About to run an agent: open a Baseline Modes engagement if the profile
		// enables it (idempotent + best-effort — never blocks the run).
		a.maybeStartEngagement(task, scenarioName)
	}
	return proceed, reason, err
}

// EnhancePrompt adds Auto Steer context to the task prompt
// Returns the enhanced prompt or the original prompt if no Auto Steer is active
func (a *AutoSteerIntegration) EnhancePrompt(task *tasks.TaskItem, basePrompt string) (string, error) {
	if !a.isEligible(task) {
		return basePrompt, nil // No enhancement needed
	}

	// Get Auto Steer prompt section
	autoSteerSection, err := a.executionOrchestrator.GetEnhancedPrompt(task.ID)
	if err != nil {
		return "", fmt.Errorf("failed to get Auto Steer prompt enhancement: %w", err)
	}

	if autoSteerSection == "" {
		// No active Auto Steer state (shouldn't happen if profile is configured)
		log.Printf("Warning: Task %s has Auto Steer profile but no active state", task.ID)
		return basePrompt, nil
	}

	// Insert Auto Steer section into prompt (placeholder-aware)
	enhancedPrompt := autosteer.InjectSteeringSection(basePrompt, autoSteerSection)

	// Log for debugging
	currentSet, _ := a.executionOrchestrator.GetCurrentSet(task.ID)
	log.Printf("Enhanced prompt with Auto Steer (%v skill set) for task %s", currentSet, task.ID)

	return enhancedPrompt, nil
}

// EvaluateIteration runs one controller MEASURE+TERMINATE step after a task's
// agent run. The orchestrator re-audits, records the realized delta, and either
// finalizes (stop) or selects the next skill (continue). Returns whether the
// loop should continue.
func (a *AutoSteerIntegration) EvaluateIteration(task *tasks.TaskItem, scenarioName string) (bool, error) {
	if !a.isEligible(task) {
		// No Auto Steer - task should continue normally based on ProcessorAutoRequeue
		return true, nil
	}

	log.Printf("Evaluating Auto Steer iteration for task %s", task.ID)

	evaluation, err := a.executionOrchestrator.EvaluateIteration(task.ID, scenarioName)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate Auto Steer iteration: %w", err)
	}

	if evaluation.ShouldStop {
		log.Printf("Auto Steer: Task %s controller stopping (reason: %s)", task.ID, evaluation.Reason)
		systemlog.Infof("Auto Steer: Task %s run complete - reason: %s", task.ID, evaluation.Reason)
		// Close any Baseline Modes engagement: promote (objective met) or abandon.
		a.maybeFinishEngagement(task.ID, evaluation.Reason)
		return false, nil
	}

	// Continuing: a checkpoint_on_green engagement banks a validated win early,
	// ending the loop rather than risking a later regression.
	if a.maybeCheckpointPromote(task.ID, evaluation) {
		log.Printf("Auto Steer: Task %s promoted at green checkpoint — ending engagement", task.ID)
		systemlog.Infof("Auto Steer: Task %s promoted at green checkpoint", task.ID)
		return false, nil
	}

	log.Printf("Auto Steer: Task %s continuing — next skill %q", task.ID, evaluation.ChosenSkill)
	systemlog.Infof("Auto Steer: Task %s continuing with skill %s", task.ID, evaluation.ChosenSkill)
	return true, nil
}

// ShouldContinueTask determines if a task should continue (requeue) after
// execution, driving the controller loop one iteration.
func (a *AutoSteerIntegration) ShouldContinueTask(task *tasks.TaskItem, scenarioName string) (bool, error) {
	if task == nil {
		return false, fmt.Errorf("task is nil")
	}

	// Honor explicit opt-outs even when Auto Steer is configured.
	if !task.ProcessorAutoRequeue {
		log.Printf("Auto Steer: Task %s has auto-enqueue disabled; skipping recycle", task.ID)
		return false, nil
	}

	if task.AutoSteerProfileID == "" {
		// No Auto Steer - use normal ProcessorAutoRequeue behavior
		return task.ProcessorAutoRequeue, nil
	}

	shouldContinue, err := a.EvaluateIteration(task, scenarioName)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate iteration: %w", err)
	}
	if !shouldContinue {
		log.Printf("Auto Steer: Task %s controller finished - will not requeue", task.ID)
	}
	return shouldContinue, nil
}

// GetCurrentSet returns the current Auto Steer skill set for a task
func (a *AutoSteerIntegration) GetCurrentSet(task *tasks.TaskItem) ([]string, error) {
	if !a.isEligible(task) {
		return nil, nil
	}

	return a.executionOrchestrator.GetCurrentSet(task.ID)
}

// IsActive checks if Auto Steer is active for a task
func (a *AutoSteerIntegration) IsActive(task *tasks.TaskItem) bool {
	if !a.isEligible(task) {
		return false
	}

	active, err := a.executionOrchestrator.IsAutoSteerActive(task.ID)
	if err != nil {
		log.Printf("Warning: Failed to check Auto Steer status for task %s: %v", task.ID, err)
		return false
	}

	return active
}
