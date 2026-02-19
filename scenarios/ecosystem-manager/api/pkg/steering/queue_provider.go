package steering

import (
	"fmt"
	"log"
	"strings"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// QueueProvider implements steering for tasks using an ordered queue of skill sets.
// Each queue step runs exactly once before advancing to the next.
// When the queue is exhausted, the task completes.
type QueueProvider struct {
	stateRepo      QueueStateRepository
	promptEnhancer autosteer.PromptEnhancerAPI
}

// Compile-time interface assertion
var _ SteeringProvider = (*QueueProvider)(nil)

// NewQueueProvider creates a new QueueProvider.
func NewQueueProvider(stateRepo QueueStateRepository, promptEnhancer autosteer.PromptEnhancerAPI) *QueueProvider {
	return &QueueProvider{
		stateRepo:      stateRepo,
		promptEnhancer: promptEnhancer,
	}
}

// Strategy returns StrategyQueue.
func (p *QueueProvider) Strategy() SteeringStrategy {
	return StrategyQueue
}

// GetCurrentSet returns the current skill set from the queue.
func (p *QueueProvider) GetCurrentSet(task *tasks.TaskItem) ([]string, error) {
	if p.stateRepo == nil || task == nil {
		return nil, nil
	}

	state, err := p.stateRepo.Get(task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue state: %w", err)
	}

	if state == nil {
		return nil, nil
	}

	state.QueueLength = len(task.SteeringQueue)
	if state.IsExhausted() || state.CurrentIndex < 0 || state.CurrentIndex >= len(task.SteeringQueue) {
		return nil, nil
	}

	return normalizedSkillSet(task.SteeringQueue[state.CurrentIndex]), nil
}

// GetQueueState returns the current queue state for a task.
// This is used for metadata enrichment to get actual queue position.
func (p *QueueProvider) GetQueueState(taskID string) (*QueueState, error) {
	if p.stateRepo == nil {
		return nil, nil
	}
	return p.stateRepo.Get(taskID)
}

// EnhancePrompt generates a steering section for the current queue skill set.
func (p *QueueProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	if p.stateRepo == nil || p.promptEnhancer == nil {
		return nil, nil
	}

	state, err := p.stateRepo.Get(task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue state: %w", err)
	}

	if state == nil {
		// No state yet - initialize it
		if err := p.Initialize(task); err != nil {
			return nil, fmt.Errorf("failed to initialize queue: %w", err)
		}
		state, err = p.stateRepo.Get(task.ID)
		if err != nil || state == nil {
			return nil, fmt.Errorf("failed to get queue state after init: %w", err)
		}
	}

	state.QueueLength = len(task.SteeringQueue)
	if state.IsExhausted() || state.CurrentIndex < 0 || state.CurrentIndex >= len(task.SteeringQueue) {
		return nil, nil
	}

	skillSet := normalizedSkillSet(task.SteeringQueue[state.CurrentIndex])
	section := generateSectionFromSet(p.promptEnhancer, skillSet, false, "")
	if section == "" {
		return nil, nil
	}

	// Add queue progress info to the section
	section = p.addQueueProgressInfo(section, state, task)

	return &PromptEnhancement{
		Section: section,
		Source:  fmt.Sprintf("queue:%s[%s]", strings.Join(skillSet, ","), state.Position()),
	}, nil
}

// addQueueProgressInfo appends queue progress information to the section.
func (p *QueueProvider) addQueueProgressInfo(section string, state *QueueState, task *tasks.TaskItem) string {
	var sb strings.Builder
	sb.WriteString(section)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("## Queue Progress\n\n")
	sb.WriteString(fmt.Sprintf("**Position:** %s\n", state.Position()))
	if state.CurrentIndex >= 0 && state.CurrentIndex < len(task.SteeringQueue) {
		sb.WriteString(fmt.Sprintf("**Current Focus:** %s\n", strings.Join(task.SteeringQueue[state.CurrentIndex], ", ")))
	}

	if state.Remaining() > 1 {
		sb.WriteString(fmt.Sprintf("**Remaining:** %d more items after this\n", state.Remaining()-1))
		sb.WriteString("\n**Upcoming:**\n")
		for i := state.CurrentIndex + 1; i < len(task.SteeringQueue) && i < state.CurrentIndex+4; i++ {
			sb.WriteString(fmt.Sprintf("- %s\n", strings.Join(task.SteeringQueue[i], ", ")))
		}
		if len(task.SteeringQueue)-state.CurrentIndex > 4 {
			sb.WriteString(fmt.Sprintf("- ... and %d more\n", len(task.SteeringQueue)-state.CurrentIndex-4))
		}
	}

	return sb.String()
}

// AfterExecution advances the queue and determines if the task should continue.
func (p *QueueProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	if p.stateRepo == nil {
		return &SteeringDecision{
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "no_state_repo",
		}, nil
	}

	state, err := p.stateRepo.Get(task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue state: %w", err)
	}

	if state == nil {
		log.Printf("Warning: Queue state not found for task %s - treating as exhausted", task.ID)
		return &SteeringDecision{
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "no_queue_state",
		}, nil
	}

	state.QueueLength = len(task.SteeringQueue)
	var currentSet []string
	if state.CurrentIndex >= 0 && state.CurrentIndex < len(task.SteeringQueue) {
		currentSet = normalizedSkillSet(task.SteeringQueue[state.CurrentIndex])
	}

	// Advance to next item in queue
	hasMore := state.Advance()
	if err := p.stateRepo.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save queue state: %w", err)
	}

	if !hasMore {
		// Queue exhausted
		log.Printf("Queue exhausted for task %s after skill set %v", task.ID, currentSet)
		return &SteeringDecision{
			SkillSet:      currentSet,
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "queue_exhausted",
		}, nil
	}

	// More items in queue
	var nextSet []string
	if state.CurrentIndex >= 0 && state.CurrentIndex < len(task.SteeringQueue) {
		nextSet = normalizedSkillSet(task.SteeringQueue[state.CurrentIndex])
	}
	log.Printf("Queue advanced for task %s: %v -> %v (%s)", task.ID, currentSet, nextSet, state.Position())

	return &SteeringDecision{
		SkillSet:      nextSet,
		ShouldRequeue: true,
		Exhausted:     false,
		Reason:        fmt.Sprintf("queue_advance_%s", state.Position()),
	}, nil
}

// Initialize creates queue state from the task's SteeringQueue field.
func (p *QueueProvider) Initialize(task *tasks.TaskItem) error {
	if p.stateRepo == nil {
		return fmt.Errorf("state repository not available")
	}

	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// Check if state already exists
	existing, err := p.stateRepo.Get(task.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing queue state: %w", err)
	}

	if existing != nil {
		// Already initialized
		log.Printf("Queue state already exists for task %s (position: %s)", task.ID, existing.Position())
		return nil
	}

	if len(task.SteeringQueue) == 0 {
		return fmt.Errorf("steering queue is empty")
	}

	state := NewQueueState(task.ID, len(task.SteeringQueue))
	if err := p.stateRepo.Save(state); err != nil {
		return fmt.Errorf("failed to save queue state: %w", err)
	}

	log.Printf("Queue initialized for task %s with %d items", task.ID, len(task.SteeringQueue))
	return nil
}

// Reset removes the queue state for a task.
func (p *QueueProvider) Reset(taskID string) error {
	if p.stateRepo == nil {
		return nil
	}

	return p.stateRepo.Delete(taskID)
}

func normalizedSkillSet(set []string) []string {
	out := make([]string, 0, len(set))
	for _, raw := range set {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		if normalized == "" {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func generateSectionFromSet(enhancer autosteer.PromptEnhancerAPI, skillSet []string, withScope bool, scope string) string {
	if enhancer == nil {
		return ""
	}
	if len(skillSet) == 0 {
		return ""
	}
	return enhancer.GenerateSkillSetSection(skillSet, withScope, scope)
}
