package mocks

import (
	"context"
	"strings"
	"sync"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

var _ event.Store = (*FakeEventStore)(nil)

// CapturedLogEvent is the log-focused view most phase tests assert on.
type CapturedLogEvent struct {
	Level   string
	Message string
}

// FakeEventStore is an in-memory event.Store for tests that need to inspect
// emitted events without bringing up SQLite.
type FakeEventStore struct {
	mu sync.Mutex

	EventsByRun map[uuid.UUID][]*domain.RunEvent
	Logs        []CapturedLogEvent

	AppendErr error
	GetErr    error
	StreamErr error
	CountErr  error
	DeleteErr error
}

func NewFakeEventStore() *FakeEventStore {
	return &FakeEventStore{EventsByRun: map[uuid.UUID][]*domain.RunEvent{}}
}

func (s *FakeEventStore) Append(_ context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if s.AppendErr != nil {
		return s.AppendErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	for _, evt := range events {
		if evt == nil {
			continue
		}
		s.EventsByRun[runID] = append(s.EventsByRun[runID], evt)
		log, ok := evt.Data.(*domain.LogEventData)
		if ok && log != nil {
			s.Logs = append(s.Logs, CapturedLogEvent{Level: log.Level, Message: log.Message})
		}
	}
	return nil
}

func (s *FakeEventStore) Get(_ context.Context, runID uuid.UUID, _ event.GetOptions) ([]*domain.RunEvent, error) {
	if s.GetErr != nil {
		return nil, s.GetErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	return append([]*domain.RunEvent(nil), s.EventsByRun[runID]...), nil
}

func (s *FakeEventStore) Stream(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error) {
	if s.StreamErr != nil {
		return nil, s.StreamErr
	}
	ch := make(chan *domain.RunEvent)
	close(ch)
	return ch, nil
}

func (s *FakeEventStore) Count(_ context.Context, runID uuid.UUID) (int64, error) {
	if s.CountErr != nil {
		return 0, s.CountErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	return int64(len(s.EventsByRun[runID])), nil
}

func (s *FakeEventStore) Delete(_ context.Context, runID uuid.UUID) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	delete(s.EventsByRun, runID)
	return nil
}

func (s *FakeEventStore) FindLogMessage(substr string) (CapturedLogEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, evt := range s.Logs {
		if strings.Contains(evt.Message, substr) {
			return evt, true
		}
	}
	return CapturedLogEvent{}, false
}

// TypedEvents returns every typed-operational event captured for runID,
// in arrival order. Tests use this to assert that phases emit the right
// typed payload after a fallback / sandbox op / heartbeat miss.
func (s *FakeEventStore) TypedEvents(runID uuid.UUID, eventType domain.RunEventType) []*domain.RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensure()
	var out []*domain.RunEvent
	for _, evt := range s.EventsByRun[runID] {
		if evt == nil {
			continue
		}
		if evt.EventType != eventType {
			continue
		}
		out = append(out, evt)
	}
	return out
}

func (s *FakeEventStore) ensure() {
	if s.EventsByRun == nil {
		s.EventsByRun = map[uuid.UUID][]*domain.RunEvent{}
	}
}
