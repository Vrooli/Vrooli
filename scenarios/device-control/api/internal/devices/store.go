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
	IdentityKey                                          string
	Endpoint                                             string
	Status, Health, HealthReason, HostNodeID, Transport  string
	Capabilities                                         []strategy.Capability
	FirstSeenAt, LastSeenAt                              time.Time
	ObservedAt                                           time.Time
	Transports                                           []strategy.DeviceTransport
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
	canonicalID := record.ID
	if record.Kind == "physical" && record.Serial != "" {
		for id, existing := range s.records {
			if id == record.ID || existing.Kind != "physical" {
				continue
			}
			if existing.Serial == record.Serial {
				canonicalID = existing.ID
				// Keep the canonical row; only an alternate endpoint-keyed row
				// is removed when it is not the selected identity.
				if id != canonicalID {
					delete(s.records, id)
				}
			} else if record.Endpoint != "" && existing.Serial == record.Endpoint && record.Serial != record.Endpoint {
				// An endpoint-keyed pre-promotion row is replaced by the
				// stronger hardware identity contributed by the incoming strategy.
				delete(s.records, id)
				canonicalID = record.ID
			}
		}
	}
	if existing, ok := s.records[canonicalID]; ok {
		record.ID = canonicalID
		record.FirstSeenAt = existing.FirstSeenAt
		record.Transports = mergeTransports(existing.Transports, record)
		if record.StrategyID == "" {
			record.StrategyID = existing.StrategyID
		}
		if record.Transport == "" {
			record.Transport = existing.Transport
		}
		if record.Serial == "" {
			record.Serial = existing.Serial
		}
	} else {
		record.Transports = mergeTransports(nil, record)
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

func mergeTransports(existing []strategy.DeviceTransport, record Record) []strategy.DeviceTransport {
	merged := append([]strategy.DeviceTransport(nil), existing...)
	for _, candidate := range record.Transports {
		mergeTransport(&merged, candidate)
	}
	name := record.Transport
	if name == "" {
		name = record.StrategyID
	}
	transport := strategy.DeviceTransport{StrategyID: record.StrategyID, Name: name, Endpoint: record.Endpoint, Health: record.Health, HealthReason: record.HealthReason, Capabilities: map[string]strategy.Capability{}, ObservedAt: record.ObservedAt}
	for _, capability := range record.Capabilities {
		transport.Capabilities[capability.Name] = capability
	}
	for i := range merged {
		if merged[i].StrategyID == transport.StrategyID && merged[i].Name == transport.Name {
			merged[i] = transport
			return merged
		}
	}
	mergeTransport(&merged, transport)
	sort.Slice(merged, func(i, j int) bool { return merged[i].Name < merged[j].Name })
	return merged
}

func mergeTransport(merged *[]strategy.DeviceTransport, transport strategy.DeviceTransport) {
	if transport.StrategyID == "" && transport.Name == "" {
		return
	}
	for i := range *merged {
		if (*merged)[i].StrategyID == transport.StrategyID && (*merged)[i].Name == transport.Name {
			(*merged)[i] = transport
			return
		}
	}
	*merged = append(*merged, transport)
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
	record.Transports = append([]strategy.DeviceTransport(nil), record.Transports...)
	for i := range record.Transports {
		record.Transports[i].Capabilities = cloneCapabilities(record.Transports[i].Capabilities)
	}
	return record
}

func cloneCapabilities(input map[string]strategy.Capability) map[string]strategy.Capability {
	if input == nil {
		return nil
	}
	out := make(map[string]strategy.Capability, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
