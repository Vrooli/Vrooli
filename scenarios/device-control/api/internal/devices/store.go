// Package devices owns the durable-in-memory identity map for physical targets.
// It deliberately knows nothing about how a strategy drives a device.
package devices

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"device-control/internal/identity"
	"device-control/strategy"
)

type Record struct {
	ID, Name, Kind, Serial, Model, OSVersion, StrategyID string
	IdentityKey                                          string
	IdentityKind                                         string
	Claims                                               []identity.IdentityClaim
	IdentityReason                                       string
	Endpoint                                             string
	Status, Health, HealthReason, HostNodeID, Transport  string
	Capabilities                                         []strategy.Capability
	Properties                                           []strategy.PropertyDescriptor
	FirstSeenAt, LastSeenAt                              time.Time
	ObservedAt                                           time.Time
	Transports                                           []strategy.DeviceTransport
}

type Store struct {
	mu      sync.RWMutex
	records map[string]Record
	merges  map[string]MergeSnapshot
}

type MergeSnapshot struct {
	CanonicalBefore Record
	Members         map[string]Record
	Claim           identity.IdentityClaim
}

func NewStore() *Store {
	return &Store{records: map[string]Record{}, merges: map[string]MergeSnapshot{}}
}

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
	if len(record.Properties) == 0 {
		record.Properties = old.Properties
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
	record.Claims = normalizedClaims(record)
	if len(record.Claims) > 0 && record.IdentityKey == "" {
		record.IdentityKey = record.Claims[0].Value
	}
	canonicalID := record.ID
	var existing Record
	var hasExisting bool
	for id, candidate := range s.records {
		if candidate.Kind != "physical" || id == record.ID {
			continue
		}
		candidateClaims := candidate.Claims
		if len(candidateClaims) == 0 {
			candidateClaims = normalizedClaims(candidate)
		}
		if identity.ClaimsMatch(candidateClaims, record.Claims) {
			canonicalID, existing, hasExisting = id, candidate, true
			existing.Claims = candidateClaims
			break
		}
		if sameEndpoint(candidate.Endpoint, record.Endpoint) && record.Endpoint != "" {
			record.IdentityReason = "address-only-correlation-refused"
			candidate.IdentityReason = "address-only-correlation-refused"
			s.records[id] = clone(candidate)
		}
	}
	if current, ok := s.records[canonicalID]; ok {
		if len(current.Claims) > 0 && len(record.Claims) > 0 && !identity.ClaimsMatch(current.Claims, record.Claims) {
			// A reused transport id with a conflicting hardware claim is a
			// distinct identity, never an overwrite of the old audit owner.
			canonicalID = record.ID + "#conflict-" + string(record.Claims[0].Kind)
		} else {
			existing, hasExisting = current, true
		}
	}
	if hasExisting {
		record.ID = canonicalID
		record.FirstSeenAt = existing.FirstSeenAt
		record.Transports = mergeTransports(existing.Transports, record)
		record.Claims = mergeClaims(existing.Claims, record.Claims)
		if record.StrategyID == "" {
			record.StrategyID = existing.StrategyID
		}
		if record.Transport == "" {
			record.Transport = existing.Transport
		}
		if record.Serial == "" {
			record.Serial = existing.Serial
		}
		if record.IdentityKey == "" {
			record.IdentityKey = existing.IdentityKey
		}
		if record.IdentityKind == "" {
			record.IdentityKind = existing.IdentityKind
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
	if len(record.Properties) == 0 {
		record.Properties = old.Properties
	}
	s.records[record.ID] = clone(record)
	return clone(record)
}

func normalizedClaims(record Record) []identity.IdentityClaim {
	claims := make([]identity.IdentityClaim, 0, len(record.Claims)+1)
	for _, claim := range record.Claims {
		if claim.Valid() {
			claims = appendUniqueClaim(claims, claim)
		}
	}
	if len(claims) == 0 {
		kind := strings.TrimSpace(record.IdentityKind)
		if kind == "" {
			switch record.StrategyID {
			case "android-adb":
				kind = string(identity.ADBSerial)
			case "android-tv-remote":
				kind = string(identity.BluetoothMAC)
			case "google-cast":
				kind = string(identity.CastID)
			}
		}
		value := strings.TrimSpace(record.IdentityKey)
		if value == "" && kind == string(identity.ADBSerial) {
			value = strings.TrimSpace(record.Serial)
		}
		if value != "" {
			if claim, err := identity.NewClaim(kind, value, record.StrategyID, "observed"); err == nil {
				claims = append(claims, claim)
			}
		}
	}
	return claims
}

func appendUniqueClaim(claims []identity.IdentityClaim, claim identity.IdentityClaim) []identity.IdentityClaim {
	for _, existing := range claims {
		if existing.Key() == claim.Key() && existing.Evidence == claim.Evidence {
			return claims
		}
	}
	return append(claims, claim)
}

func mergeClaims(left, right []identity.IdentityClaim) []identity.IdentityClaim {
	merged := append([]identity.IdentityClaim(nil), left...)
	for _, claim := range right {
		merged = appendUniqueClaim(merged, claim)
	}
	return merged
}

func sameEndpoint(left, right string) bool {
	return left != "" && right != "" && strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
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
	transport := strategy.DeviceTransport{StrategyID: record.StrategyID, Name: name, Endpoint: record.Endpoint, Health: record.Health, HealthReason: record.HealthReason, Capabilities: map[string]strategy.Capability{}, Properties: append([]strategy.PropertyDescriptor(nil), record.Properties...), ObservedAt: record.ObservedAt}
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
	for canonical, snapshot := range s.merges {
		if canonical == id {
			delete(s.merges, canonical)
			continue
		}
		if _, ok := snapshot.Members[id]; ok {
			delete(s.merges, canonical)
		}
	}
	return true
}

// Merge combines two identities only after the caller has supplied a valid
// shared observed claim or an explicit owner assertion. The pre-merge records
// are retained so split can restore both identities without losing transport
// attribution.
func (s *Store) Merge(canonicalID, memberID string, claim identity.IdentityClaim) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := identity.ValidateClaim(claim); err != nil {
		return Record{}, err
	}
	canonical, ok := s.records[canonicalID]
	if !ok {
		return Record{}, fmt.Errorf("identity %q not found", canonicalID)
	}
	member, ok := s.records[memberID]
	if !ok {
		return Record{}, fmt.Errorf("identity %q not found", memberID)
	}
	if canonicalID == memberID {
		return Record{}, fmt.Errorf("cannot merge an identity with itself")
	}
	if claim.Evidence != "owner-asserted" && !identity.ClaimsMatch([]identity.IdentityClaim{claim}, canonical.Claims) {
		return Record{}, fmt.Errorf("canonical identity %q does not carry claim %s=%s", canonicalID, claim.Kind, claim.Value)
	}
	if claim.Evidence != "owner-asserted" && !identity.ClaimsMatch([]identity.IdentityClaim{claim}, member.Claims) {
		return Record{}, fmt.Errorf("member identity %q does not carry claim %s=%s", memberID, claim.Kind, claim.Value)
	}
	canonicalBefore := clone(canonical)
	merged := canonical
	merged.Claims = mergeClaims(canonical.Claims, member.Claims)
	merged.Claims = appendUniqueClaim(merged.Claims, claim)
	merged.Transports = mergeTransports(canonical.Transports, Record{StrategyID: member.StrategyID, Transport: member.Transport, Endpoint: member.Endpoint, Health: member.Health, HealthReason: member.HealthReason, Capabilities: member.Capabilities, Properties: member.Properties, ObservedAt: member.ObservedAt})
	for _, transport := range member.Transports {
		mergeTransport(&merged.Transports, transport)
	}
	if merged.Name == "" {
		merged.Name = member.Name
	}
	if merged.Model == "" {
		merged.Model = member.Model
	}
	merged.IdentityReason = "merged-on-" + string(claim.Kind)
	s.records[canonicalID] = clone(merged)
	delete(s.records, memberID)
	if s.merges == nil {
		s.merges = map[string]MergeSnapshot{}
	}
	s.merges[canonicalID] = MergeSnapshot{CanonicalBefore: canonicalBefore, Members: map[string]Record{memberID: clone(member)}, Claim: claim}
	return clone(merged), nil
}

// Split restores every pre-merge identity for canonicalID. The operation is
// intentionally all-or-nothing from the store's perspective.
func (s *Store) Split(canonicalID string) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.merges[canonicalID]
	if !ok {
		return nil, fmt.Errorf("identity %q has no merge history", canonicalID)
	}
	restored := []Record{clone(snapshot.CanonicalBefore)}
	s.records[canonicalID] = clone(snapshot.CanonicalBefore)
	for memberID, record := range snapshot.Members {
		s.records[memberID] = clone(record)
		restored = append(restored, clone(record))
	}
	delete(s.merges, canonicalID)
	sort.Slice(restored, func(i, j int) bool { return restored[i].ID < restored[j].ID })
	return restored, nil
}

// RestoreMerge rehydrates durable merge history before inventory resumes.
func (s *Store) RestoreMerge(canonicalID string, snapshot MergeSnapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.merges == nil {
		s.merges = map[string]MergeSnapshot{}
	}
	current, ok := s.merges[canonicalID]
	if !ok {
		current = MergeSnapshot{CanonicalBefore: clone(snapshot.CanonicalBefore), Members: map[string]Record{}, Claim: snapshot.Claim}
	}
	for id, record := range snapshot.Members {
		current.Members[id] = clone(record)
	}
	s.merges[canonicalID] = current
}

func (s *Store) MergeSnapshot(canonicalID string) (MergeSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, ok := s.merges[canonicalID]
	if !ok {
		return MergeSnapshot{}, false
	}
	copySnapshot := MergeSnapshot{CanonicalBefore: clone(snapshot.CanonicalBefore), Members: map[string]Record{}, Claim: snapshot.Claim}
	for id, record := range snapshot.Members {
		copySnapshot.Members[id] = clone(record)
	}
	return copySnapshot, true
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
	record.Properties = append([]strategy.PropertyDescriptor(nil), record.Properties...)
	record.Claims = append([]identity.IdentityClaim(nil), record.Claims...)
	record.Transports = append([]strategy.DeviceTransport(nil), record.Transports...)
	for i := range record.Transports {
		record.Transports[i].Capabilities = cloneCapabilities(record.Transports[i].Capabilities)
		record.Transports[i].Properties = append([]strategy.PropertyDescriptor(nil), record.Transports[i].Properties...)
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
