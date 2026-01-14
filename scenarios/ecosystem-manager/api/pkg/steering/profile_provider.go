package steering

import (
	"fmt"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// AutoSteerIntegrationAPI defines the interface for Auto Steer integration.
// This breaks the circular dependency between steering and queue packages.
type AutoSteerIntegrationAPI interface {
	// InitializeAutoSteer initializes Auto Steer execution for a task.
	InitializeAutoSteer(task *tasks.TaskItem, scenarioName string) error

	// EnhancePrompt adds Auto Steer context to the task prompt.
	EnhancePrompt(task *tasks.TaskItem, basePrompt string) (string, error)

	// ShouldContinueTask determines if a task should continue (requeue) after execution.
	ShouldContinueTask(task *tasks.TaskItem, scenarioName string) (bool, error)

	// GetCurrentMode returns the current Auto Steer mode for a task.
	GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error)

	// ExecutionOrchestrator returns the underlying orchestrator for advanced operations.
	ExecutionOrchestrator() *autosteer.ExecutionOrchestrator
}

// ProfileProvider implements steering for tasks using Auto Steer profiles.
// It wraps the existing AutoSteerIntegration to maintain compatibility.
type ProfileProvider struct {
	integration AutoSteerIntegrationAPI
}

// Compile-time interface assertion
var _ SteeringProvider = (*ProfileProvider)(nil)

// NewProfileProvider creates a new ProfileProvider wrapping an AutoSteerIntegration.
func NewProfileProvider(integration AutoSteerIntegrationAPI) *ProfileProvider {
	return &ProfileProvider{
		integration: integration,
	}
}

// Strategy returns StrategyProfile.
func (p *ProfileProvider) Strategy() SteeringStrategy {
	return StrategyProfile
}

// GetCurrentMode returns the current mode from the Auto Steer profile.
func (p *ProfileProvider) GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error) {
	if p.integration == nil || task == nil {
		return "", nil
	}
	return p.integration.GetCurrentMode(task)
}

// EnhancePrompt delegates to AutoSteerIntegration for profile-based prompt enhancement.
// Returns a PromptEnhancement with the section to inject, or nil if not applicable.
func (p *ProfileProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	if p.integration == nil {
		return nil, nil
	}

	// Get the Auto Steer enhanced section
	orchestrator := p.integration.ExecutionOrchestrator()
	if orchestrator == nil {
		return nil, nil
	}

	section, err := orchestrator.GetEnhancedPrompt(task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile prompt enhancement: %w", err)
	}

	if section == "" {
		return nil, nil
	}

	// Get current mode for source attribution
	mode, _ := orchestrator.GetCurrentMode(task.ID)
	source := "profile"
	if mode != "" {
		source = "profile:" + string(mode)
	}

	return &PromptEnhancement{
		Section: section,
		Source:  source,
	}, nil
}

// AfterExecution delegates to AutoSteerIntegration for profile-based continuation decision.
func (p *ProfileProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	if p.integration == nil {
		return &SteeringDecision{
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "no_integration",
		}, nil
	}

	shouldContinue, err := p.integration.ShouldContinueTask(task, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("profile evaluation failed: %w", err)
	}

	mode, _ := p.integration.GetCurrentMode(task)

	if !shouldContinue {
		return &SteeringDecision{
			Mode:          mode,
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "profile_completed",
		}, nil
	}

	return &SteeringDecision{
		Mode:          mode,
		ShouldRequeue: true,
		Exhausted:     false,
		Reason:        "profile_continues",
	}, nil
}

// Initialize delegates to AutoSteerIntegration for profile initialization.
func (p *ProfileProvider) Initialize(task *tasks.TaskItem) error {
	if p.integration == nil {
		return nil
	}

	// Get scenario name from task target
	scenarioName := task.Target
	if scenarioName == "" && len(task.Targets) > 0 {
		scenarioName = task.Targets[0]
	}

	return p.integration.InitializeAutoSteer(task, scenarioName)
}

// Reset clears the Auto Steer execution state for a task.
func (p *ProfileProvider) Reset(taskID string) error {
	if p.integration == nil || p.integration.ExecutionOrchestrator() == nil {
		return nil
	}

	return p.integration.ExecutionOrchestrator().DeleteExecutionState(taskID)
}
