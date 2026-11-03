package aimoneyfixtures

import (
	"context"
	"errors"
	"sync"
)

// LifecycleSnapshot is a test observation, not the production three-fact
// schema. It lets later tests model which backend answered and an opaque state.
type LifecycleSnapshot struct {
	Backend string
	State   string
	Version string
}
type LifecycleAction struct {
	Harness string
	Action  string
}

// LifecycleBackend is the intentionally small two-backend test seam from the
// parallel plan review. Production state remains owned by harness-lifecycle.
type LifecycleBackend interface {
	Observe(context.Context, string) (LifecycleSnapshot, error)
	Apply(context.Context, LifecycleAction) error
}

type FakeLifecycleBackend struct {
	Name       string
	mu         sync.Mutex
	Snapshots  map[string]LifecycleSnapshot
	Actions    []LifecycleAction
	ApplyError error
}

func (f *FakeLifecycleBackend) Observe(_ context.Context, harness string) (LifecycleSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	snapshot, ok := f.Snapshots[harness]
	if !ok {
		return LifecycleSnapshot{}, errors.New("harness not found")
	}
	snapshot.Backend = f.Name
	return snapshot, nil
}
func (f *FakeLifecycleBackend) Apply(_ context.Context, action LifecycleAction) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if action.Harness == "" || action.Action == "" {
		return errors.New("harness and action required")
	}
	if f.ApplyError != nil {
		return f.ApplyError
	}
	f.Actions = append(f.Actions, action)
	return nil
}

// TwoBackends selects the local or server-side fake explicitly; it never
// silently falls through from one authority to another.
type TwoBackends struct {
	Local  LifecycleBackend
	Server LifecycleBackend
}

func (b TwoBackends) Backend(local bool) LifecycleBackend {
	if local {
		return b.Local
	}
	return b.Server
}
