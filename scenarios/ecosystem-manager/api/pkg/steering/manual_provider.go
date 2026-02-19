package steering

import (
	"strings"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// ManualProvider implements steering for tasks with a manually-selected skill set.
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

// GetCurrentSet returns the skill set from task.SteerSet.
func (p *ManualProvider) GetCurrentSet(task *tasks.TaskItem) ([]string, error) {
	return p.getSkillSetFromTask(task), nil
}

func (p *ManualProvider) getSkillSetFromTask(task *tasks.TaskItem) []string {
	if task == nil {
		return []string{string(autosteer.ModeProgress)}
	}
	out := make([]string, 0, len(task.SteerSet))
	for _, raw := range task.SteerSet {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		if !autosteer.SteerMode(normalized).IsValid() {
			continue
		}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return []string{string(autosteer.ModeProgress)}
	}
	return out
}

// EnhancePrompt generates a steering section for the task's configured SteerSet.
func (p *ManualProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	if p.promptEnhancer == nil {
		return nil, nil
	}

	skillSet := p.getSkillSetFromTask(task)
	section := generateSectionFromSet(p.promptEnhancer, skillSet, false, "")
	if section == "" {
		return nil, nil
	}

	return &PromptEnhancement{
		Section: section,
		Source:  "manual:" + strings.Join(skillSet, ","),
	}, nil
}

// AfterExecution always indicates the task can continue with the same set.
func (p *ManualProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	skillSet := p.getSkillSetFromTask(task)
	return &SteeringDecision{
		SkillSet:      skillSet,
		ShouldRequeue: true,
		Exhausted:     false,
		Reason:        "manual_set_continues",
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
