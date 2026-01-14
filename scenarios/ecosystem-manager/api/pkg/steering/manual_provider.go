package steering

import (
	"strings"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// ManualProvider implements steering for tasks with a manually-selected mode.
// The mode is stored on the task's SteerMode field and repeats indefinitely.
type ManualProvider struct {
	promptEnhancer autosteer.PromptEnhancerAPI
}

// Compile-time interface assertion
var _ SteeringProvider = (*ManualProvider)(nil)

// NewManualProvider creates a new ManualProvider.
func NewManualProvider(promptEnhancer autosteer.PromptEnhancerAPI) *ManualProvider {
	return &ManualProvider{
		promptEnhancer: promptEnhancer,
	}
}

// Strategy returns StrategyManual.
func (p *ManualProvider) Strategy() SteeringStrategy {
	return StrategyManual
}

// GetCurrentMode returns the mode from the task's SteerMode field.
// Falls back to Progress if the mode is invalid.
func (p *ManualProvider) GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error) {
	return p.getModeFromTask(task), nil
}

// getModeFromTask extracts and validates the steering mode from a task.
func (p *ManualProvider) getModeFromTask(task *tasks.TaskItem) autosteer.SteerMode {
	if task == nil {
		return autosteer.ModeProgress
	}

	mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(task.SteerMode)))
	if !mode.IsValid() {
		return autosteer.ModeProgress
	}
	return mode
}

// EnhancePrompt generates a mode section for the task's configured SteerMode.
func (p *ManualProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	if p.promptEnhancer == nil {
		return nil, nil
	}

	mode := p.getModeFromTask(task)
	section := p.promptEnhancer.GenerateModeSection(mode)
	if section == "" {
		return nil, nil
	}

	return &PromptEnhancement{
		Section: section,
		Source:  "manual:" + string(mode),
	}, nil
}

// AfterExecution always indicates the task can continue with the same mode.
// Manual mode never exhausts - it repeats indefinitely.
func (p *ManualProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	mode := p.getModeFromTask(task)
	return &SteeringDecision{
		Mode:          mode,
		ShouldRequeue: true,
		Exhausted:     false,
		Reason:        "manual_mode_continues",
	}, nil
}

// Initialize is a no-op for the manual strategy.
func (p *ManualProvider) Initialize(task *tasks.TaskItem) error {
	return nil
}

// Reset is a no-op for the manual strategy.
func (p *ManualProvider) Reset(taskID string) error {
	return nil
}
