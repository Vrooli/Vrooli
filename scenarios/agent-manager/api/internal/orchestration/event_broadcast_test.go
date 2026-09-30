package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type failingEventStore struct {
	err error
}

func (s failingEventStore) Append(context.Context, uuid.UUID, ...*domain.RunEvent) error {
	return s.err
}

func (s failingEventStore) Get(context.Context, uuid.UUID, event.GetOptions) ([]*domain.RunEvent, error) {
	return nil, nil
}

func (s failingEventStore) Stream(context.Context, uuid.UUID, event.StreamOptions) (<-chan *domain.RunEvent, error) {
	return nil, nil
}

func (s failingEventStore) Count(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

func (s failingEventStore) Delete(context.Context, uuid.UUID) error {
	return nil
}

type recordingBroadcaster struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (b *recordingBroadcaster) BroadcastEvent(evt *domain.RunEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
}

func (b *recordingBroadcaster) BroadcastRunStatus(*domain.Run) {}

func (b *recordingBroadcaster) BroadcastProgress(uuid.UUID, domain.RunPhase, int, string) {}

func (b *recordingBroadcaster) eventCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}

func TestAppendAndBroadcastEvents_DoesNotBroadcastWhenAppendFails(t *testing.T) {
	runID := uuid.New()
	broadcaster := &recordingBroadcaster{}
	storeErr := errors.New("append failed")

	err := appendAndBroadcastEvents(
		context.Background(),
		failingEventStore{err: storeErr},
		broadcaster,
		runID,
		domain.NewLogEvent(runID, "info", "hello"),
	)

	if !errors.Is(err, storeErr) {
		t.Fatalf("expected append error, got %v", err)
	}
	if got := broadcaster.eventCount(); got != 0 {
		t.Fatalf("expected no broadcasts after append failure, got %d", got)
	}
}

func TestBroadcastingEventSink_DoesNotBroadcastWhenAppendFails(t *testing.T) {
	runID := uuid.New()
	broadcaster := &recordingBroadcaster{}
	storeErr := errors.New("append failed")
	sink := &broadcastingEventSink{
		store:       failingEventStore{err: storeErr},
		runID:       runID,
		broadcaster: broadcaster,
	}

	err := sink.Emit(domain.NewLogEvent(runID, "info", "hello"))

	if !errors.Is(err, storeErr) {
		t.Fatalf("expected append error, got %v", err)
	}
	if got := broadcaster.eventCount(); got != 0 {
		t.Fatalf("expected no broadcasts after append failure, got %d", got)
	}
}
