package steering

import (
	"fmt"
	"time"
)

// InMemoryQueueStateRepository implements QueueStateRepository using an in-memory map.
// Useful for testing and development.
type InMemoryQueueStateRepository struct {
	states map[string]*QueueState
}

// Compile-time interface assertion
var _ QueueStateRepository = (*InMemoryQueueStateRepository)(nil)

// NewInMemoryQueueStateRepository creates a new in-memory queue state repository.
func NewInMemoryQueueStateRepository() *InMemoryQueueStateRepository {
	return &InMemoryQueueStateRepository{
		states: make(map[string]*QueueState),
	}
}

// Get retrieves the queue state for a task from memory.
func (r *InMemoryQueueStateRepository) Get(taskID string) (*QueueState, error) {
	state, ok := r.states[taskID]
	if !ok {
		return nil, nil
	}
	// Return a copy to prevent mutation
	cp := *state
	return &cp, nil
}

// Save persists the queue state to memory.
func (r *InMemoryQueueStateRepository) Save(state *QueueState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}
	// Store a copy to prevent mutation
	cp := *state
	r.states[state.TaskID] = &cp
	return nil
}

// Delete removes the queue state for a task from memory.
func (r *InMemoryQueueStateRepository) Delete(taskID string) error {
	delete(r.states, taskID)
	return nil
}

// ResetPosition resets the queue position to 0 without deleting the state.
func (r *InMemoryQueueStateRepository) ResetPosition(taskID string) error {
	if state, ok := r.states[taskID]; ok {
		state.Reset() // Uses existing QueueState.Reset() method
	}
	return nil
}

// SetPosition sets the queue position to a specific index.
func (r *InMemoryQueueStateRepository) SetPosition(taskID string, position int) error {
	if position < 0 {
		return fmt.Errorf("position must be non-negative, got %d", position)
	}

	state, ok := r.states[taskID]
	if !ok {
		return fmt.Errorf("no queue state found for task %s", taskID)
	}

	state.CurrentIndex = position
	state.UpdatedAt = time.Now().Format(time.RFC3339)
	return nil
}
