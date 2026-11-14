package orchestration

import (
	"testing"

	"github.com/google/uuid"
)

func TestAwaitRegistryOldWatcherExitCannotDeregisterReplacement(t *testing.T) {
	registry := NewAwaitRegistry(nil)
	defer registry.shutdownCancel()
	runID := uuid.New()
	oldCanceled := false
	old := &awaitRegistration{cancel: func() { oldCanceled = true }}
	registry.mu.Lock()
	registry.active[runID] = old
	registry.mu.Unlock()

	// Cancel removes and cancels the old registration. A new park for the same
	// Run can then register while the old watcher's deferred cleanup is pending.
	registry.Cancel(runID)
	if !oldCanceled {
		t.Fatal("Cancel did not cancel the old watcher")
	}
	newRegistration := &awaitRegistration{cancel: func() {}}
	registry.mu.Lock()
	registry.active[runID] = newRegistration
	registry.mu.Unlock()

	// Model the old watcher's deferred cleanup after the replacement is live.
	registry.deregister(runID, old)
	registry.mu.Lock()
	got := registry.active[runID]
	registry.mu.Unlock()
	if got != newRegistration {
		t.Fatal("old watcher cleanup removed the replacement registration")
	}

	// The replacement remains independently cancelable after stale cleanup.
	registry.Cancel(runID)
	registry.mu.Lock()
	got = registry.active[runID]
	registry.mu.Unlock()
	if got != nil {
		t.Fatal("replacement registration remained active after Cancel")
	}
}
