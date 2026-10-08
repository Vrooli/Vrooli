// Package maintenance owns the admission fence for planned Agent Manager maintenance.
// The orchestration owner must hold an admission until its accepted work is durable.
// Existing work is observed by that owner; this package never cancels or schedules it.
//
// Integration contract installed by the composition root:
//   - Bootstrap Schema through EnsureSchemas for each routed store, then share one
//     Gate among the corresponding orchestration admission paths and HTTP handler.
//   - Reconcile accepted idempotency replays first. Acquire Admit before a new
//     reservation/dispatch in CreateRun, ContinueRun and StartWorkflowExecution,
//     including their alternate entry paths. Release after durable acceptance or
//     rejection, never by cancelling the admitted run.
//   - The orchestration owner distinguishes new work from already-admitted pending
//     work and workflow descendants using durable identity, not caller force flags.
//     The latter must keep progressing and remain covered by the drain observer.
//   - The observer includes pending/starting/running work and admitted workflow
//     lifetimes, plus any execution state a restart would interrupt. Unrecovered or
//     partial inventories are errors, not empty drain results.
//   - The control plane retains the closed revision through its planned lifecycle
//     operation, including restart. Only an authorized explicit Resume with that
//     revision reopens admission. A timeout ends observation, not maintenance.
//
// Operator protocol (adoption of a new build belongs to the lifecycle owner):
//
//	agent-manager maintenance status --json
//	agent-manager maintenance begin --reason "planned AM rollout" --local-owner --json
//	agent-manager maintenance drain --timeout 120s --local-owner --json
//	agent-manager maintenance resume --revision <closed-revision> --local-owner --json
//
// Local-owner exchange is an explicit human operation; identified agents cannot
// use it to elevate. Drain timeout leaves work and the durable fence unchanged.
// A closed fence must survive the owner's lifecycle operation. Never infer a
// safe restart from a terminal run row: InventoryReader includes recorded physical
// executors. Missing descendant/group evidence remains unknown and requires the
// control-plane owner, as does indirect restart impact through dependencies.
package maintenance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ErrClosed   = errors.New("agent-manager admission is closed for maintenance")
	ErrConflict = errors.New("maintenance revision changed")
)

type State struct {
	Closed   bool   `json:"closed"`
	Revision int64  `json:"revision"`
	Owner    string `json:"owner"`
	Reason   string `json:"reason"`
}

type Standing struct {
	State
	Admitting          int        `json:"admitting"`
	Remaining          *int       `json:"remaining"`
	Drained            bool       `json:"drained"`
	LifecycleInterlock string     `json:"lifecycleInterlock,omitempty"`
	Inventory          *Inventory `json:"inventory,omitempty"`
}

type Store interface {
	Load(context.Context) (State, error)
	Save(context.Context, State, int64) error
}

// Gate is shared by every admission path in one serving AM process. The lifecycle
// owner must ensure there is only one serving process for this store. Persistence
// is checked on admission, so failures refuse work and restarts retain the fence.
type Gate struct {
	mu        sync.Mutex
	store     Store
	admitting int
	changed   chan struct{}
}

func NewGate(store Store) *Gate { return &Gate{store: store, changed: make(chan struct{})} }

func (g *Gate) signal() { close(g.changed); g.changed = make(chan struct{}) }

// Admit linearizes with Enter. Release only after the admission is persisted or
// rejected; it is not a run cancellation function. The drain observer owns the
// durable pending/running/continuing work after release, including after restart.
func (g *Gate) Admit(ctx context.Context) (func(), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	if state.Closed {
		return nil, ErrClosed
	}
	g.admitting++
	var once sync.Once
	return func() { once.Do(func() { g.mu.Lock(); defer g.mu.Unlock(); g.admitting--; g.signal() }) }, nil
}

func (g *Gate) Enter(ctx context.Context, owner, reason string) (State, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(reason) == "" {
		return State{}, fmt.Errorf("maintenance requires an authenticated owner and reason")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.store.Load(ctx)
	if err != nil || state.Closed {
		return state, err
	}
	next := State{Closed: true, Revision: state.Revision + 1, Owner: owner, Reason: reason}
	if err := g.store.Save(ctx, next, state.Revision); err != nil {
		return State{}, err
	}
	g.signal()
	return next, nil
}

func (g *Gate) Resume(ctx context.Context, owner string, revision int64) (State, error) {
	if strings.TrimSpace(owner) == "" {
		return State{}, fmt.Errorf("maintenance requires an authenticated owner")
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.store.Load(ctx)
	if err != nil {
		return State{}, err
	}
	if state.Revision != revision {
		return state, ErrConflict
	}
	if !state.Closed {
		return state, nil
	}
	next := State{Revision: state.Revision + 1, Owner: owner, Reason: state.Reason}
	if err := g.store.Save(ctx, next, state.Revision); err != nil {
		return State{}, err
	}
	g.signal()
	return next, nil
}

func (g *Gate) Status(ctx context.Context) (Standing, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	state, err := g.store.Load(ctx)
	return Standing{State: state, Admitting: g.admitting}, err
}

// Wait observes admitted work without owning its lifecycle. Cancellation and
// timeout end only this attachment; they leave the fence and work unchanged.
// The observer must include all admitted work that a restart could interrupt.
func (g *Gate) Wait(ctx context.Context, remaining func(context.Context) (int, error), interval time.Duration) (Standing, error) {
	if remaining == nil {
		return Standing{}, fmt.Errorf("maintenance drain observer is unavailable")
	}
	if interval <= 0 {
		interval = time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return Standing{}, err
		}
		g.mu.Lock()
		changed := g.changed
		g.mu.Unlock()
		before, err := g.Status(ctx)
		if err != nil {
			return before, err
		}
		if !before.Closed {
			return before, ErrConflict
		}
		count, err := remaining(ctx)
		if err != nil {
			return before, err
		}
		if count < 0 {
			return before, fmt.Errorf("owner returned unknown work count")
		}
		after, err := g.Status(ctx)
		if err != nil {
			return after, err
		}
		after.Remaining = &count
		if !after.Closed || before.Revision != after.Revision {
			return after, ErrConflict
		}
		if before.Admitting == 0 && after.Admitting == 0 && count == 0 {
			after.Drained = true
			return after, nil
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return after, ctx.Err()
		case <-changed:
			timer.Stop()
		case <-timer.C:
		}
	}
}
