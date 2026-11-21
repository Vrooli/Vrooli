package checks

import (
	"context"
	"sync"
	"testing"
	"time"
)

type orderedTrackerStore struct {
	mu       sync.Mutex
	once     sync.Once
	entered  chan struct{}
	release  chan struct{}
	trackers map[string]*HealTracker
}

func (s *orderedTrackerStore) SaveHealTracker(ctx context.Context, id string, t *HealTracker) error {
	s.once.Do(func() { close(s.entered) })
	select {
	case <-s.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := *t
	s.trackers[id] = &copy
	return nil
}
func (s *orderedTrackerStore) DeleteHealTracker(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.trackers, id)
	return nil
}
func (s *orderedTrackerStore) GetAllHealTrackers(context.Context) (map[string]*HealTracker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]*HealTracker{}
	for id, t := range s.trackers {
		copy := *t
		out[id] = &copy
	}
	return out, nil
}

func TestTrackerMutationWaitsForPersistenceWithoutBlockingStatus(t *testing.T) {
	reg := newTestRegistry()
	store := &orderedTrackerStore{entered: make(chan struct{}), release: make(chan struct{}), trackers: map[string]*HealTracker{}}
	var unblock sync.Once
	release := func() { unblock.Do(func() { close(store.release) }) }
	defer release()
	reg.SetHealTrackerStore(store)
	done := make(chan struct{})
	go func() { reg.updateHealTracker("scenario-app", outcomeFailure); close(done) }()
	select {
	case <-store.entered:
	case <-time.After(time.Second):
		t.Fatal("save did not start")
	}
	select {
	case <-done:
		t.Fatal("mutation returned before durable save finished")
	case <-time.After(20 * time.Millisecond):
	}
	readDone := make(chan struct{})
	go func() { reg.GetAllHealTrackers(); close(readDone) }()
	select {
	case <-readDone:
	case <-time.After(time.Second):
		t.Fatal("slow save blocked status")
	}
	release()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("mutation did not complete")
	}
	reg.observeScenarioHealth(Result{CheckID: "scenario-app", Status: StatusOK})
	loaded, err := store.GetAllHealTrackers(t.Context())
	if err != nil || loaded["scenario-app"] == nil || loaded["scenario-app"].ConsecutiveFailures != 0 || !loaded["scenario-app"].UnhealthySince.IsZero() {
		t.Fatalf("newer recovery was not durable: %+v %v", loaded, err)
	}
	reg.ResetHealTracker("scenario-app")
	restored := newTestRegistry()
	restored.SetHealTrackerStore(store)
	if err := restored.LoadHealTrackers(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, ok := restored.GetHealTracker("scenario-app"); ok {
		t.Fatal("reset tracker resurrected on reload")
	}
}

func TestConcurrentTrackerMutationsPersistTheFinalState(t *testing.T) {
	reg := newTestRegistry()
	store := &orderedTrackerStore{entered: make(chan struct{}), release: make(chan struct{}), trackers: map[string]*HealTracker{}}
	close(store.release)
	reg.SetHealTrackerStore(store)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); reg.updateHealTracker("scenario-app", outcomeFailure) }()
	}
	wg.Wait()
	memory, _ := reg.GetHealTracker("scenario-app")
	stored, err := store.GetAllHealTrackers(t.Context())
	if err != nil || stored["scenario-app"] == nil || memory.TotalAttempts != 20 || *stored["scenario-app"] != *memory {
		t.Fatalf("durable state differs from completed mutations: memory=%+v stored=%+v err=%v", memory, stored["scenario-app"], err)
	}
}
