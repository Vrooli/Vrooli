// Package devices owns the durable-in-memory identity map for physical targets.
// It deliberately knows nothing about how a strategy drives a device.
package devices

import (
	"sort"
	"sync"
	"time"

	"device-control/strategy"
)

type Record struct {
	ID, Name, Kind, Serial, Model, OSVersion, StrategyID string
	Endpoint                                             string
	Status, Health, HealthReason, HostNodeID, Transport  string
	Capabilities                                         []strategy.Capability
	FirstSeenAt, LastSeenAt                              time.Time
	ObservedAt                                           time.Time
}

type Store struct {
	mu      sync.RWMutex
	records map[string]Record
}

func NewStore() *Store { return &Store{records: map[string]Record{}} }

func (s *Store) Upsert(record Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := record.ObservedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	old, existed := s.records[record.ID]
	if !existed || old.FirstSeenAt.IsZero() {
		record.FirstSeenAt = now
	} else {
		record.FirstSeenAt = old.FirstSeenAt
	}
	record.LastSeenAt = now
	record.ObservedAt = now
	if len(record.Capabilities) == 0 {
		record.Capabilities = old.Capabilities
	}
	s.records[record.ID] = clone(record)
	return clone(record)
}

// UpsertIdentity writes a device under its hardware identity and removes
// stale endpoint-keyed rows for the same physical target. A transport address
// is deliberately treated as an alias, never as a second device.
func (s *Store) UpsertIdentity(record Record) Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	if record.Kind == "physical" && record.Serial != "" {
		for id, existing := range s.records {
			if id == record.ID || existing.Kind != "physical" {
				continue
			}
			if existing.Serial == record.Serial || (record.Endpoint != "" && existing.Serial == record.Endpoint) {
				delete(s.records, id)
			}
		}
	}
	now := record.ObservedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	old, existed := s.records[record.ID]
	if !existed || old.FirstSeenAt.IsZero() {
		record.FirstSeenAt = now
	} else {
		record.FirstSeenAt = old.FirstSeenAt
	}
	record.LastSeenAt = now
	record.ObservedAt = now
	if len(record.Capabilities) == 0 {
		record.Capabilities = old.Capabilities
	}
	s.records[record.ID] = clone(record)
	return clone(record)
}

func (s *Store) MarkAbsentExcept(now time.Time, present map[string]bool, reason func(Record) string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, record := range s.records {
		if (record.Kind != "physical" && record.Kind != "emulator") || present[id] {
			continue
		}
		record.Status = "unreachable"
		record.Health = "unreachable"
		record.HealthReason = reason(record)
		record.ObservedAt = now
		s.records[id] = clone(record)
	}
}

func (s *Store) MarkAbsent(now time.Time, reason func(Record) string) {
	s.MarkAbsentExcept(now, nil, reason)
}

func (s *Store) Get(id string) (Record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.records[id]
	return clone(record), ok
}

// Forget removes an explicitly owner-forgotten device identity. Normal
// disappearance uses MarkAbsent so audit attribution survives reconnects;
// forgetting is the deliberate, irreversible removal of that retained row.
func (s *Store) Forget(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.records[id]; !ok {
		return false
	}
	delete(s.records, id)
	return true
}

func (s *Store) List() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		out = append(out, clone(record))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func clone(record Record) Record {
	capabilities := make([]strategy.Capability, len(record.Capabilities))
	copy(capabilities, record.Capabilities)
	record.Capabilities = capabilities
	return record
}
