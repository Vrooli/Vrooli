package effectiveness

import (
	"sort"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/dimensions"
)

// MemoryStore is an in-memory effectiveness ledger. It is the test double for
// Store and is concurrency-safe (the production controller can steer multiple
// tasks at once).
type MemoryStore struct {
	mu   sync.Mutex
	rows map[key]*Stat
	// now is injectable so tests can assert recency without a real clock.
	now func() time.Time
}

type key struct {
	skill string
	dim   dimensions.Dimension
}

var _ Store = (*MemoryStore)(nil)

// NewMemoryStore creates an empty in-memory ledger.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{rows: make(map[key]*Stat), now: time.Now}
}

// WithClock overrides the store's clock (for deterministic recency in tests).
func (m *MemoryStore) WithClock(now func() time.Time) *MemoryStore {
	m.now = now
	return m
}

// Seed pre-populates a stat (test convenience).
func (m *MemoryStore) Seed(s Stat) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := s
	m.rows[key{s.SkillID, s.Dimension}] = &cp
}

// Get implements Store.
func (m *MemoryStore) Get(skillID string, dim dimensions.Dimension) (Stat, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.rows[key{skillID, dim}]; ok {
		return *s, true, nil
	}
	return Stat{}, false, nil
}

// Bulk implements Store.
func (m *MemoryStore) Bulk(dim dimensions.Dimension) (map[string]Stat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]Stat)
	for k, s := range m.rows {
		if k.dim == dim {
			out[k.skill] = *s
		}
	}
	return out, nil
}

// Record implements Store with commutative increments.
func (m *MemoryStore) Record(ev CreditEvent) error {
	if ev.SkillID == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()

	touch := func(dim dimensions.Dimension) *Stat {
		k := key{ev.SkillID, dim}
		s, ok := m.rows[k]
		if !ok {
			s = &Stat{SkillID: ev.SkillID, Dimension: dim}
			m.rows[k] = s
		}
		s.UpdatedAt = now
		return s
	}

	// The target dimension earns the run count, token cost, and recency.
	target := touch(ev.TargetDimension)
	target.TotalRuns++
	target.TotalTokens += ev.Tokens
	target.LastRunAt = now

	// Closed/introduced are recorded for every observed dimension (collateral
	// dimensions earn their debt even though the run did not target them).
	for dim, n := range ev.ClosedByDimension {
		touch(dim).ClosedCount += int64(n)
	}
	for dim, n := range ev.IntroducedByDimension {
		touch(dim).IntroducedCount += int64(n)
	}
	return nil
}

// List implements Store.
func (m *MemoryStore) List(skillID string, dim dimensions.Dimension) ([]Stat, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Stat, 0, len(m.rows))
	for k, s := range m.rows {
		if skillID != "" && k.skill != skillID {
			continue
		}
		if dim != "" && k.dim != dim {
			continue
		}
		out = append(out, *s)
	}
	sortStats(out)
	return out, nil
}

// sortStats orders rows deterministically (dimension, then skill).
func sortStats(rows []Stat) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Dimension != rows[j].Dimension {
			return rows[i].Dimension < rows[j].Dimension
		}
		return rows[i].SkillID < rows[j].SkillID
	})
}
