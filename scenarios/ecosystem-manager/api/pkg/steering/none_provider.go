package steering

import (
	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// NoneProvider implements steering for tasks with no explicit steering configuration.
// It defaults to Progress mode and always indicates the task should continue
// (actual continuation is controlled by ProcessorAutoRequeue).
type NoneProvider struct {
	promptEnhancer autosteer.PromptEnhancerAPI
}

// Compile-time interface assertion
var _ SteeringProvider = (*NoneProvider)(nil)

// NewNoneProvider creates a new NoneProvider.
func NewNoneProvider(promptEnhancer autosteer.PromptEnhancerAPI) *NoneProvider {
	return &NoneProvider{
		promptEnhancer: promptEnhancer,
	}
}

// Strategy returns StrategyNone.
func (p *NoneProvider) Strategy() SteeringStrategy {
	return StrategyNone
}

// GetCurrentMode returns Progress as the default mode.
func (p *NoneProvider) GetCurrentMode(taskID string) (autosteer.SteerMode, error) {
	return autosteer.ModeProgress, nil
}

// EnhancePrompt generates a Progress mode section for the prompt.
func (p *NoneProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	if p.promptEnhancer == nil {
		return nil, nil
	}

	section := p.promptEnhancer.GenerateModeSection(autosteer.ModeProgress)
	if section == "" {
		return nil, nil
	}

	return &PromptEnhancement{
		Section: section,
		Source:  "none:progress",
	}, nil
}

// AfterExecution always indicates the task can continue.
// Actual continuation is controlled by ProcessorAutoRequeue on the task.
func (p *NoneProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	return &SteeringDecision{
		Mode:          autosteer.ModeProgress,
		ShouldRequeue: true,
		Exhausted:     false,
		Reason:        "none_strategy_continues",
	}, nil
}

// Initialize is a no-op for the none strategy.
func (p *NoneProvider) Initialize(task *tasks.TaskItem) error {
	return nil
}

// Reset is a no-op for the none strategy.
func (p *NoneProvider) Reset(taskID string) error {
	return nil
}
