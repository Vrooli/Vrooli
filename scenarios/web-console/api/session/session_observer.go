package session

import (
	"sync"
	"sync/atomic"
)

// ObserverFrame is the payload delivered to registered observers for every
// completed PTY output frame. In Phase 1 it carries only the decoded byte
// stream; Phase 2 adds parsed control events and a screen view.
//
// DOC: docs/internal/SEAMS.md (Observer seam)
type ObserverFrame struct {
	// Decoded is the post-UTF-8-reassembly byte stream broadcast to clients.
	// Observers must not mutate this slice.
	Decoded []byte
}

// Observer is the seam through which non-client consumers (idle detector,
// prompt detector, ANSI responder, adapter dispatchers) tap the PTY output
// stream without growing readLoop.
//
// Implementations MUST be non-blocking. The dispatcher invokes OnFrame
// synchronously from the read-loop goroutine after broadcast; observers
// that need to do work should hand the frame to their own goroutine.
type Observer interface {
	OnFrame(frame ObserverFrame)
}

// UnregisterFunc removes a previously registered observer. Safe to call
// multiple times; subsequent calls are no-ops.
type UnregisterFunc func()

// observerRegistry holds the set of observers attached to a Session.
// Lazily initialized by RegisterObserver to avoid touching every Session
// construction site.
type observerRegistry struct {
	mu      sync.RWMutex
	nextID  uint64
	entries map[uint64]Observer
}

func (r *observerRegistry) register(o Observer) UnregisterFunc {
	r.mu.Lock()
	id := atomic.AddUint64(&r.nextID, 1)
	if r.entries == nil {
		r.entries = make(map[uint64]Observer)
	}
	r.entries[id] = o
	r.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			r.mu.Lock()
			delete(r.entries, id)
			r.mu.Unlock()
		})
	}
}

func (r *observerRegistry) dispatch(frame ObserverFrame) {
	r.mu.RLock()
	if len(r.entries) == 0 {
		r.mu.RUnlock()
		return
	}
	// Snapshot observers under RLock so OnFrame runs without holding it
	// (callers must not block, but we still avoid lock-ordering risk).
	obs := make([]Observer, 0, len(r.entries))
	for _, o := range r.entries {
		obs = append(obs, o)
	}
	r.mu.RUnlock()
	for _, o := range obs {
		o.OnFrame(frame)
	}
}

// RegisterObserver attaches an Observer to this session. The returned
// function removes it. Observers see frames AFTER they are broadcast to
// WebSocket clients — clients have priority; observers must not block.
func (s *Session) RegisterObserver(o Observer) UnregisterFunc {
	s.observersOnce.Do(func() { s.observers = &observerRegistry{} })
	return s.observers.register(o)
}

func (s *Session) dispatchObservers(frame ObserverFrame) {
	if s.observers == nil {
		return
	}
	s.observers.dispatch(frame)
}
