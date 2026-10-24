package orchestration

import (
	"context"
	"errors"
	"sync"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

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

func (s failingEventStore) Summarize(context.Context, uuid.UUID, event.GetOptions) (*event.Summary, error) {
	return &event.Summary{Counts: map[string]int64{}}, nil
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

func TestNewRunEventSinkSharesPersistenceBroadcastAndObserverRouting(t *testing.T) {
	for _, tc := range []struct {
		name             string
		withStore        bool
		withBroadcaster  bool
		wantStored       int
		wantBroadcasts   int
		wantObservations int
	}{
		{name: "no configured outputs"},
		{name: "broadcaster without store", withBroadcaster: true},
		{name: "store only", withStore: true, wantStored: 2, wantObservations: 1},
		{name: "store and broadcaster", withStore: true, withBroadcaster: true, wantStored: 2, wantBroadcasts: 1, wantObservations: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runID := uuid.New()
			var store event.Store
			var fakeStore *mocks.FakeEventStore
			if tc.withStore {
				fakeStore = mocks.NewFakeEventStore()
				store = fakeStore
			}
			var broadcaster EventBroadcaster
			var recording *recordingBroadcaster
			if tc.withBroadcaster {
				recording = &recordingBroadcaster{}
				broadcaster = recording
			}

			observations := 0
			observerSawPersistedEvent := false
			sink := newRunEventSink(store, broadcaster, runID, func(*domain.RunEvent) {
				observations++
				persisted, err := fakeStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
				if err == nil && len(persisted) == observations {
					observerSawPersistedEvent = true
				}
			})
			first := domain.NewLogEvent(runID, "info", "first")
			if err := sink.Emit(first); err != nil {
				t.Fatalf("emit first event: %v", err)
			}
			replay := domain.NewLogEvent(runID, "info", "replayed")
			replay.PersistenceReplay = true
			if err := sink.Emit(replay); err != nil {
				t.Fatalf("emit replay event: %v", err)
			}

			stored := 0
			if fakeStore != nil {
				persisted, err := fakeStore.Get(t.Context(), runID, event.GetOptions{AfterSequence: -1})
				if err != nil {
					t.Fatalf("read persisted events: %v", err)
				}
				stored = len(persisted)
			}
			if stored != tc.wantStored {
				t.Errorf("persisted events = %d, want %d", stored, tc.wantStored)
			}
			if got := observations; got != tc.wantObservations {
				t.Errorf("after-persist observations = %d, want %d", got, tc.wantObservations)
			}
			if tc.wantObservations > 0 && !observerSawPersistedEvent {
				t.Error("after-persist callback ran before the event was durable")
			}
			if recording != nil {
				if got := recording.eventCount(); got != tc.wantBroadcasts {
					t.Errorf("broadcast events = %d, want %d", got, tc.wantBroadcasts)
				}
			}
		})
	}
}
