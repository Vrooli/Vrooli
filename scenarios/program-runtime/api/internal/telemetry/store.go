package telemetry

import (
	"context"
	"sort"
	"sync"

	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
)

type Store struct {
	mu        sync.RWMutex
	events    []*telemetryv1.ProgramEvent
	publisher Publisher
}

func NewStore() *Store { return &Store{} }

func NewStoreWithPublisher(publisher Publisher) *Store {
	return &Store{publisher: publisher}
}

func (s *Store) Append(event *telemetryv1.ProgramEvent) {
	if event == nil {
		return
	}
	s.mu.Lock()
	s.events = append(s.events, event)
	publisher := s.publisher
	s.mu.Unlock()
	if publisher != nil {
		// Event delivery is best-effort and must never turn a successful program
		// into a failure merely because the optional platform bus is down.
		go func() { _ = publisher.Publish(context.Background(), event) }()
	}
}

func (s *Store) List(sessionID string, kind telemetryv1.EventKind) []*telemetryv1.ProgramEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*telemetryv1.ProgramEvent, 0)
	for _, event := range s.events {
		if sessionID != "" && event.SessionId != sessionID {
			continue
		}
		if kind != telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED && event.Kind != kind {
			continue
		}
		copy := *event
		out = append(out, &copy)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OccurredAt < out[j].OccurredAt })
	return out
}
