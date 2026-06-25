package stats

import (
	"context"
	"sync"

	"swarm-manager/internal/eventlog"
)

// indexByteFast returns the index of the first occurrence of c in s,
// or -1 if absent. Used to parse kind from entity IDs of the form
// "<kind>/<name>" without dragging in the strings package for one call.
func indexByteFast(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

const refreshBatchSize = 5000

// Engine incrementally aggregates events into metrics using a watermark pattern.
type Engine struct {
	mu        sync.RWMutex
	watermark int64
	repo      eventlog.Repository
	state     *aggregateState
}

// NewEngine creates a stats engine backed by the given event repository.
func NewEngine(repo eventlog.Repository) *Engine {
	return &Engine{
		repo:  repo,
		state: newAggregateState(),
	}
}

// Rebuild replays all events from scratch. Called once at startup.
func (e *Engine) Rebuild(ctx context.Context) error {
	events, err := e.repo.All(ctx)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state = newAggregateState()
	for i := range events {
		e.state.processEvent(&events[i])
	}

	maxID, err := e.repo.MaxID(ctx)
	if err != nil {
		return err
	}
	e.watermark = maxID
	return nil
}

// Refresh incrementally processes events appended since the last watermark.
func (e *Engine) Refresh(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for {
		events, err := e.repo.Since(ctx, e.watermark, refreshBatchSize)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}
		for i := range events {
			e.state.processEvent(&events[i])
		}
		e.watermark = events[len(events)-1].ID
		if len(events) < refreshBatchSize {
			return nil
		}
	}
}

// GetStats returns the current computed metrics. Callers should call Refresh first.
func (e *Engine) GetStats() StatsResponse {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state.buildResponse()
}
