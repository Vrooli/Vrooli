package graph

import (
	"container/list"
	"sync"
)

// DefaultPathMutexCapacity is the default upper bound on distinct path
// entries the PathMutex registry retains. The bound is advisory: held
// entries (refs > 0) are never evicted. Only idle entries (refs == 0)
// participate in LRU eviction.
const DefaultPathMutexCapacity = 10_000

// PathMutex is a registry of per-absolute-path locks. It implements
// OT-P0-006: two concurrent operations targeting the same module_path
// serialize; two targeting different paths run in parallel.
//
// The registry is bounded: idle entries are tracked in an LRU list and
// evicted when the entry count exceeds the configured capacity. Held
// entries (refs > 0) are exempt — eviction never invalidates a lock a
// goroutine is currently waiting on or holding. When the registry is
// saturated with held entries, capacity is exceeded transiently; idle
// eviction resumes on the next release.
type PathMutex struct {
	mu      sync.Mutex
	cap     int
	entries map[string]*pathEntry
	idle    *list.List // values: string keys; front = LRU, back = MRU
}

// pathEntry tracks a single absolute path's mutex along with the
// reference count and its position in the idle list (nil when held).
type pathEntry struct {
	mu   *sync.Mutex
	refs int
	el   *list.Element
}

// NewPathMutex returns a PathMutex with the default capacity bound.
func NewPathMutex() *PathMutex {
	return NewPathMutexWithCapacity(DefaultPathMutexCapacity)
}

// NewPathMutexWithCapacity returns a PathMutex bounded to roughly cap
// idle entries. A non-positive cap is silently coerced to the default.
// Exposed primarily so tests can exercise eviction on a small registry.
func NewPathMutexWithCapacity(cap int) *PathMutex {
	if cap <= 0 {
		cap = DefaultPathMutexCapacity
	}
	return &PathMutex{
		cap:     cap,
		entries: make(map[string]*pathEntry),
		idle:    list.New(),
	}
}

// Lock acquires the per-path mutex for absPath and returns the unlock
// closure. The caller MUST defer the returned function. Lock blocks
// while another goroutine holds the same path; goroutines locking
// different paths proceed in parallel.
//
// The function never returns nil — even an empty absPath gets its own
// mutex (so callers can defer fearlessly).
func (m *PathMutex) Lock(absPath string) func() {
	m.mu.Lock()
	e, ok := m.entries[absPath]
	if !ok {
		e = &pathEntry{mu: &sync.Mutex{}}
		m.entries[absPath] = e
	}
	if e.el != nil {
		m.idle.Remove(e.el)
		e.el = nil
	}
	e.refs++
	m.mu.Unlock()

	e.mu.Lock()
	return func() { m.release(absPath, e) }
}

// release decrements the refcount and, when an entry goes idle, parks
// it at the MRU end of the idle list. If the registry is over capacity
// it evicts LRU idle entries until it is back under the bound or the
// idle list is empty.
func (m *PathMutex) release(absPath string, e *pathEntry) {
	e.mu.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	e.refs--
	if e.refs > 0 {
		return
	}
	e.el = m.idle.PushBack(absPath)
	for len(m.entries) > m.cap && m.idle.Len() > 0 {
		front := m.idle.Front()
		evictKey, _ := front.Value.(string)
		m.idle.Remove(front)
		delete(m.entries, evictKey)
	}
}

// Len reports the number of tracked entries (held + idle). Intended for
// tests and instrumentation; not part of the production API contract.
func (m *PathMutex) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}
