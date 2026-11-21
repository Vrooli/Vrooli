package sessions

import (
	"context"
	"fmt"
	"sync"
)

type RestorationEvent struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

type Undo func(context.Context) error

// StateManager owns session-scoped undo entries and always applies them in
// reverse order. It is deliberately independent of a particular strategy so
// lease cleanup can run the same way for USB, wireless, and test adapters.
type StateManager struct {
	mu    sync.Mutex
	stack []struct {
		name string
		undo Undo
	}
}

func (m *StateManager) Push(name string, undo Undo) {
	if m == nil || undo == nil {
		return
	}
	m.mu.Lock()
	m.stack = append(m.stack, struct {
		name string
		undo Undo
	}{name: name, undo: undo})
	m.mu.Unlock()
}

func (m *StateManager) Restore(ctx context.Context) []RestorationEvent {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	stack := append([]struct {
		name string
		undo Undo
	}{}, m.stack...)
	m.stack = nil
	m.mu.Unlock()
	events := make([]RestorationEvent, 0, len(stack))
	for i := len(stack) - 1; i >= 0; i-- {
		err := stack[i].undo(ctx)
		event := RestorationEvent{Name: stack[i].name, Status: "restored"}
		if err != nil {
			event.Status = "failed"
			event.Reason = fmt.Sprint(err)
		}
		events = append(events, event)
	}
	return events
}
