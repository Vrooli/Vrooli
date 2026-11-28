package handoffrules

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemStore is an in-memory Store implementation for unit tests.
type MemStore struct {
	mu    sync.RWMutex
	rules map[string]*Rule
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore { return &MemStore{rules: make(map[string]*Rule)} }

func (m *MemStore) List(_ context.Context) ([]Rule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Rule, 0, len(m.rules))
	for _, r := range m.rules {
		out = append(out, cloneRule(*r))
	}
	sortRules(out)
	return out, nil
}

func (m *MemStore) Upsert(_ context.Context, req UpsertRequest) (Rule, error) {
	if err := req.Validate(); err != nil {
		return Rule{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := FormatTime(time.Now())
	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}

	r := Rule{
		ID:        id,
		Name:      req.Name,
		Enabled:   req.Enabled,
		Source:    req.Source,
		Pattern:   req.Pattern,
		Surfaces:  append([]string(nil), req.Surfaces...),
		SortOrder: req.SortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if existing, ok := m.rules[id]; ok {
		r.CreatedAt = existing.CreatedAt
	}
	if r.Surfaces == nil {
		r.Surfaces = []string{}
	}
	stored := cloneRule(r)
	m.rules[id] = &stored
	return r, nil
}

func (m *MemStore) Delete(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.rules[id]; !ok {
		return false, nil
	}
	delete(m.rules, id)
	return true, nil
}

// cloneRule copies the surfaces slice so a caller mutating what it got back
// cannot reach into the store's own row.
func cloneRule(r Rule) Rule {
	r.Surfaces = append([]string(nil), r.Surfaces...)
	if r.Surfaces == nil {
		r.Surfaces = []string{}
	}
	return r
}

// sortRules orders by position then id, so the sequence is total and two
// rules sharing a sort_order never swap between reads.
func sortRules(in []Rule) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].SortOrder != in[j].SortOrder {
			return in[i].SortOrder < in[j].SortOrder
		}
		return in[i].ID < in[j].ID
	})
}
