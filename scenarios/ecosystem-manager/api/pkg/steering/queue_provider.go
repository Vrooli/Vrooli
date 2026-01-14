package steering

import (
	"fmt"
	"log"
	"strings"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

// QueueProvider implements steering for tasks using an ordered queue of modes.
// Each mode in the queue runs exactly once before advancing to the next.
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

// GetCurrentMode returns the current mode from the queue.
func (p *QueueProvider) GetCurrentMode(taskID string) (autosteer.SteerMode, error) {
	if p.stateRepo == nil {
		return "", nil
	}

	state, err := p.stateRepo.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get queue state: %w", err)
	}

	if state == nil || state.IsExhausted() {
		return "", nil
	}

	return state.CurrentMode(), nil
}

// EnhancePrompt generates a steering section for the current queue mode.
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

	if state.IsExhausted() {
		return nil, nil
	}

	mode := state.CurrentMode()
	section := p.promptEnhancer.GenerateModeSection(mode)
	if section == "" {
		return nil, nil
	}

	// Add queue progress info to the section
	section = p.addQueueProgressInfo(section, state)

	return &PromptEnhancement{
		Section: section,
		Source:  fmt.Sprintf("queue:%s[%s]", mode, state.Position()),
	}, nil
}

// addQueueProgressInfo appends queue progress information to the section.
func (p *QueueProvider) addQueueProgressInfo(section string, state *QueueState) string {
	var sb strings.Builder
	sb.WriteString(section)
	sb.WriteString("\n\n---\n\n")
	sb.WriteString("## Queue Progress\n\n")
	sb.WriteString(fmt.Sprintf("**Position:** %s\n", state.Position()))
	sb.WriteString(fmt.Sprintf("**Current Focus:** %s\n", state.CurrentMode()))

	if state.Remaining() > 1 {
		sb.WriteString(fmt.Sprintf("**Remaining:** %d more items after this\n", state.Remaining()-1))
		sb.WriteString("\n**Upcoming:**\n")
		for i := state.CurrentIndex + 1; i < len(state.Queue) && i < state.CurrentIndex+4; i++ {
			sb.WriteString(fmt.Sprintf("- %s\n", state.Queue[i]))
		}
		if len(state.Queue)-state.CurrentIndex > 4 {
			sb.WriteString(fmt.Sprintf("- ... and %d more\n", len(state.Queue)-state.CurrentIndex-4))
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

	currentMode := state.CurrentMode()

	// Advance to next item in queue
	hasMore := state.Advance()
	if err := p.stateRepo.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save queue state: %w", err)
	}

	if !hasMore {
		// Queue exhausted
		log.Printf("Queue exhausted for task %s after mode %s", task.ID, currentMode)
		return &SteeringDecision{
			Mode:          currentMode,
			ShouldRequeue: false,
			Exhausted:     true,
			Reason:        "queue_exhausted",
		}, nil
	}

	// More items in queue
	nextMode := state.CurrentMode()
	log.Printf("Queue advanced for task %s: %s -> %s (%s)", task.ID, currentMode, nextMode, state.Position())

	return &SteeringDecision{
		Mode:          nextMode,
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

	// Convert string queue to SteerMode queue
	queue := make([]autosteer.SteerMode, 0, len(task.SteeringQueue))
	for _, s := range task.SteeringQueue {
		mode := autosteer.SteerMode(strings.ToLower(strings.TrimSpace(s)))
		if mode != "" {
			queue = append(queue, mode)
		}
	}

	if len(queue) == 0 {
		return fmt.Errorf("steering queue is empty")
	}

	state := NewQueueState(task.ID, queue)
	if err := p.stateRepo.Save(state); err != nil {
		return fmt.Errorf("failed to save queue state: %w", err)
	}

	log.Printf("Queue initialized for task %s with %d items: %v", task.ID, len(queue), queue)
	return nil
}

// Reset removes the queue state for a task.
func (p *QueueProvider) Reset(taskID string) error {
	if p.stateRepo == nil {
		return nil
	}

	return p.stateRepo.Delete(taskID)
}
