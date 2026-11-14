// Package faultinject is the failure-injection seam for certification tests.
//
// Boundaries in the cloud pipeline call Hit(ctx, point) at well-known points
// (transport send/reply, worker before-commit/after-effect, activation
// before/after switch, data before-backup/after-restore). A Registry armed
// with a behaviour makes Hit return a typed *Fault (or delay, or crash) at
// that point.
//
// The registry can only be armed on a context that carries the routed
// test-mode mark (database.IsTestMode). In production there is no armed
// registry, Arm returns ErrNotTestMode, and Hit never fires.
package faultinject

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/database"
)

// Point names an injection boundary.
type Point string

// Injection boundaries. Later phases call Hit(ctx, <point>) at these places.
const (
	TransportSend          Point = "transport_send"
	TransportReply         Point = "transport_reply"
	WorkerBeforeCommit     Point = "worker_before_commit"
	WorkerAfterEffect      Point = "worker_after_effect"
	ActivationBeforeSwitch Point = "activation_before_switch"
	ActivationAfterSwitch  Point = "activation_after_switch"
	DataBeforeBackup       Point = "data_before_backup"
	DataAfterRestore       Point = "data_after_restore"
)

// Points lists every boundary in a stable order.
func Points() []Point {
	return []Point{
		TransportSend, TransportReply,
		WorkerBeforeCommit, WorkerAfterEffect,
		ActivationBeforeSwitch, ActivationAfterSwitch,
		DataBeforeBackup, DataAfterRestore,
	}
}

// Kind is what happens when an armed point is hit.
type Kind string

// Behaviour kinds.
const (
	// KindFail returns a *Fault error from Hit.
	KindFail Kind = "fail"
	// KindDropReply returns a *Fault wrapping ErrReplyDropped: the effect is
	// assumed to have happened but the reply is lost.
	KindDropReply Kind = "drop_reply"
	// KindCrash invokes the registry's crash function (default: panic) to
	// simulate the process dying at the point.
	KindCrash Kind = "crash"
	// KindDelay blocks for Behaviour.Delay (or until ctx is done) then returns nil.
	KindDelay Kind = "delay"
)

// Behaviour describes an armed fault.
type Behaviour struct {
	Kind  Kind
	Delay time.Duration
	// Once disarms the point after its first hit.
	Once bool
}

// Fault is the typed error returned by Hit at an armed point.
type Fault struct {
	Point Point
	Kind  Kind
}

func (f *Fault) Error() string { return fmt.Sprintf("faultinject: %s at %s", f.Kind, f.Point) }

// Unwrap lets errors.Is(err, ErrReplyDropped) work for drop_reply faults.
func (f *Fault) Unwrap() error {
	if f.Kind == KindDropReply {
		return ErrReplyDropped
	}
	return nil
}

// Sentinel errors.
var (
	ErrNotTestMode  = errors.New("faultinject: registry can only be armed in routed test mode")
	ErrUnknownPoint = errors.New("faultinject: unknown injection point")
	ErrUnknownKind  = errors.New("faultinject: unknown behaviour kind")
	ErrReplyDropped = errors.New("faultinject: reply dropped")
)

// Injector is the seam boundaries depend on.
type Injector interface {
	// Hit reports whether an armed fault fires at point for ctx.
	Hit(ctx context.Context, point Point) error
}

// Registry holds armed points for one test scope. The zero value is safe and
// inert; use New.
type Registry struct {
	mu    sync.Mutex
	armed map[Point]Behaviour
	hits  map[Point]int
	crash func(*Fault)
}

// New returns an empty registry. Nothing fires until Arm succeeds.
func New() *Registry {
	return &Registry{armed: map[Point]Behaviour{}, hits: map[Point]int{}, crash: func(f *Fault) { panic(f) }}
}

// SetCrashFunc replaces the crash behaviour (tests substitute a recorder).
func (r *Registry) SetCrashFunc(fn func(*Fault)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.crash = fn
}

// Arm enables behaviour at point. It refuses unless ctx carries the routed
// test-mode mark, so production requests can never arm a fault.
func (r *Registry) Arm(ctx context.Context, point Point, b Behaviour) error {
	if !database.IsTestMode(ctx) {
		return ErrNotTestMode
	}
	if !knownPoint(point) {
		return fmt.Errorf("%w: %q", ErrUnknownPoint, point)
	}
	switch b.Kind {
	case KindFail, KindDropReply, KindCrash, KindDelay:
	default:
		return fmt.Errorf("%w: %q", ErrUnknownKind, b.Kind)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.armed == nil {
		r.armed = map[Point]Behaviour{}
		r.hits = map[Point]int{}
	}
	r.armed[point] = b
	return nil
}

// Disarm removes any behaviour at point.
func (r *Registry) Disarm(point Point) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.armed, point)
}

// Hits returns how many times point fired.
func (r *Registry) Hits(point Point) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.hits[point]
}

// Armed reports whether point currently has a behaviour.
func (r *Registry) Armed(point Point) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.armed[point]
	return ok
}

// Hit implements Injector. It never fires outside test mode, even if the
// registry was armed through a test-mode context earlier.
func (r *Registry) Hit(ctx context.Context, point Point) error {
	if r == nil || !database.IsTestMode(ctx) {
		return nil
	}
	r.mu.Lock()
	b, ok := r.armed[point]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	r.hits[point]++
	if b.Once {
		delete(r.armed, point)
	}
	crash := r.crash
	r.mu.Unlock()

	f := &Fault{Point: point, Kind: b.Kind}
	switch b.Kind {
	case KindDelay:
		select {
		case <-time.After(b.Delay):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	case KindCrash:
		crash(f)
		return f
	default:
		return f
	}
}

func knownPoint(p Point) bool {
	for _, known := range Points() {
		if known == p {
			return true
		}
	}
	return false
}

type registryKey struct{}

// WithRegistry attaches r to ctx so boundaries reached through ctx consult it.
// A request-scoped carrier keeps injection out of process-global state.
func WithRegistry(ctx context.Context, r *Registry) context.Context {
	return context.WithValue(ctx, registryKey{}, r)
}

// FromContext returns the registry attached to ctx, if any.
func FromContext(ctx context.Context) (*Registry, bool) {
	if ctx == nil {
		return nil, false
	}
	r, ok := ctx.Value(registryKey{}).(*Registry)
	return r, ok && r != nil
}

// Hit is the call boundaries make. With no registry on ctx, or outside test
// mode, it returns nil and costs one context lookup.
func Hit(ctx context.Context, point Point) error {
	if !database.IsTestMode(ctx) {
		return nil
	}
	r, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	return r.Hit(ctx, point)
}
