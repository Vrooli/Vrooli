package redisstate

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// Memory is an in-memory Store for tests and the documented in-process fallback
// when Redis is unavailable. It honors TTLs (checked lazily on access) and is
// safe for concurrent use. It is NOT a production substitute across replicas —
// each process holds its own map (the whole reason Redis is required for HA).
type Memory struct {
	mu     sync.Mutex
	values map[string]memEntry
	sets   map[string]map[string]struct{}
	now    func() time.Time
}

type memEntry struct {
	value   string
	expires time.Time // zero = no expiry
}

// NewMemory constructs an empty in-memory Store using the real clock.
func NewMemory() *Memory {
	return &Memory{
		values: map[string]memEntry{},
		sets:   map[string]map[string]struct{}{},
		now:    time.Now,
	}
}

// NewMemoryWithClock is the test variant with an injectable clock so TTL
// expiry is deterministic.
func NewMemoryWithClock(now func() time.Time) *Memory {
	m := NewMemory()
	m.now = now
	return m
}

var _ Store = (*Memory)(nil)

func (m *Memory) expired(e memEntry) bool {
	return !e.expires.IsZero() && m.now().After(e.expires)
}

func (m *Memory) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := memEntry{value: value}
	if ttl > 0 {
		e.expires = m.now().Add(ttl)
	}
	m.values[key] = e
	return nil
}

func (m *Memory) Get(_ context.Context, key string) (string, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.values[key]
	if !ok || m.expired(e) {
		if ok {
			delete(m.values, key)
		}
		return "", false, nil
	}
	return e.value, true, nil
}

func (m *Memory) Del(_ context.Context, keys ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, k := range keys {
		delete(m.values, k)
		delete(m.sets, k)
	}
	return nil
}

func (m *Memory) Exists(_ context.Context, key string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.values[key]
	if !ok || m.expired(e) {
		if ok {
			delete(m.values, key)
		}
		return false, nil
	}
	return true, nil
}

func (m *Memory) SAdd(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.sets[key]
	if !ok {
		set = map[string]struct{}{}
		m.sets[key] = set
	}
	for _, mem := range members {
		set[mem] = struct{}{}
	}
	return nil
}

func (m *Memory) SRem(_ context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	set, ok := m.sets[key]
	if !ok {
		return nil
	}
	for _, mem := range members {
		delete(set, mem)
	}
	if len(set) == 0 {
		delete(m.sets, key)
	}
	return nil
}

func (m *Memory) SMembers(_ context.Context, key string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	set := m.sets[key]
	out := make([]string, 0, len(set))
	for mem := range set {
		out = append(out, mem)
	}
	return out, nil
}

func (m *Memory) Incr(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.values[key]
	var n int64
	if ok && !m.expired(e) {
		n, _ = strconv.ParseInt(e.value, 10, 64)
	}
	n++
	expires := time.Time{}
	if ok && !m.expired(e) {
		expires = e.expires
	}
	m.values[key] = memEntry{value: strconv.FormatInt(n, 10), expires: expires}
	return n, nil
}

func (m *Memory) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.values[key]
	if !ok || m.expired(e) {
		return nil
	}
	if ttl > 0 {
		e.expires = m.now().Add(ttl)
	} else {
		e.expires = time.Time{}
	}
	m.values[key] = e
	return nil
}
