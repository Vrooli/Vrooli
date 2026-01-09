// Package event provides event storage and streaming implementations.
//
// This file contains an in-memory implementation of the event Store interface.
package event

import (
	"context"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// =============================================================================
// In-Memory Event Store
// =============================================================================

// MemoryStore is an in-memory implementation of the event Store interface.
type MemoryStore struct {
	mu          sync.RWMutex
	events      map[uuid.UUID][]*domain.RunEvent // runID -> events
	sequences   map[uuid.UUID]int64              // runID -> next sequence
	subscribers map[uuid.UUID][]chan *domain.RunEvent
}

// NewMemoryStore creates a new in-memory event store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:      make(map[uuid.UUID][]*domain.RunEvent),
		sequences:   make(map[uuid.UUID]int64),
		subscribers: make(map[uuid.UUID][]chan *domain.RunEvent),
	}
}

// Append adds events to a run's event stream.
func (s *MemoryStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, evt := range events {
		// Assign sequence number
		seq := s.sequences[runID]
		evt.RunID = runID
		evt.Sequence = seq
		if evt.Timestamp.IsZero() {
			evt.Timestamp = time.Now()
		}
		if evt.ID == uuid.Nil {
			evt.ID = uuid.New()
		}
		s.sequences[runID] = seq + 1

		// Store the event
		copy := *evt
		s.events[runID] = append(s.events[runID], &copy)

		// Notify subscribers
		for _, ch := range s.subscribers[runID] {
			select {
			case ch <- &copy:
			default:
				// Channel full, skip
			}
		}
	}

	return nil
}

// Get retrieves events for a run with optional filtering.
func (s *MemoryStore) Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[runID]
	var result []*domain.RunEvent

	// Build type filter set
	typeFilter := make(map[domain.RunEventType]bool)
	for _, t := range opts.EventTypes {
		typeFilter[t] = true
	}

	for _, evt := range events {
		// Apply sequence filter
		if evt.Sequence <= opts.AfterSequence {
			continue
		}

		// Apply type filter
		if len(typeFilter) > 0 && !typeFilter[evt.EventType] {
			continue
		}

		// Apply timestamp filter
		if opts.Since != nil && evt.Timestamp.Before(*opts.Since) {
			continue
		}

		copy := *evt
		result = append(result, &copy)
	}

	// Apply offset
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return []*domain.RunEvent{}, nil
	}

	// Apply limit
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

// Stream returns a channel that receives events in real-time.
func (s *MemoryStore) Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 100
	}

	ch := make(chan *domain.RunEvent, bufSize)
	s.subscribers[runID] = append(s.subscribers[runID], ch)

	// Send existing events from the starting sequence
	go func() {
		s.mu.RLock()
		events := s.events[runID]
		s.mu.RUnlock()

		for _, evt := range events {
			if evt.Sequence >= opts.FromSequence {
				// Apply type filter
				if len(opts.EventTypes) > 0 {
					found := false
					for _, t := range opts.EventTypes {
						if evt.EventType == t {
							found = true
							break
						}
					}
					if !found {
						continue
					}
				}

				select {
				case ch <- evt:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Clean up on context cancellation
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()

		// Remove from subscribers
		subs := s.subscribers[runID]
		for i, sub := range subs {
			if sub == ch {
				s.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}

// Count returns the number of events for a run.
func (s *MemoryStore) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return int64(len(s.events[runID])), nil
}

// Delete removes all events for a run.
func (s *MemoryStore) Delete(ctx context.Context, runID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.events, runID)
	delete(s.sequences, runID)

	// Close and remove subscribers
	for _, ch := range s.subscribers[runID] {
		close(ch)
	}
	delete(s.subscribers, runID)

	return nil
}

// Verify interface compliance
var _ Store = (*MemoryStore)(nil)
