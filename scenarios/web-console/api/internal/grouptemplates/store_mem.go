package grouptemplates

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemStore is an in-memory Store implementation for unit tests.
type MemStore struct {
	mu        sync.RWMutex
	templates map[string]*Template
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{templates: make(map[string]*Template)}
}

func (m *MemStore) List(_ context.Context) ([]Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Template, 0, len(m.templates))
	for _, t := range m.templates {
		out = append(out, cloneTemplate(*t))
	}
	sortTemplates(out)
	return out, nil
}

func (m *MemStore) Upsert(_ context.Context, req UpsertRequest) (Template, error) {
	if err := req.Validate(); err != nil {
		return Template{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := FormatTime(time.Now())
	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}

	t := Template{
		ID:        id,
		Name:      req.Name,
		Color:     req.Color,
		Roles:     append([]TemplateRole(nil), req.Roles...),
		UseCount:  req.UseCount,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if existing, ok := m.templates[id]; ok {
		t.CreatedAt = existing.CreatedAt
		if !req.HasUseCount {
			t.UseCount = existing.UseCount
		}
	}
	if t.Roles == nil {
		t.Roles = []TemplateRole{}
	}
	stored := cloneTemplate(t)
	m.templates[id] = &stored
	return t, nil
}

func (m *MemStore) Delete(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.templates[id]; !ok {
		return false, nil
	}
	delete(m.templates, id)
	return true, nil
}

// cloneTemplate copies the role slice so a caller mutating what it got back
// cannot reach into the store's own row.
func cloneTemplate(t Template) Template {
	t.Roles = append([]TemplateRole(nil), t.Roles...)
	if t.Roles == nil {
		t.Roles = []TemplateRole{}
	}
	return t
}

// sortTemplates orders by name then id, so the sequence is total and two
// templates sharing a name never swap between reads.
func sortTemplates(in []Template) {
	sort.Slice(in, func(i, j int) bool {
		if in[i].Name != in[j].Name {
			return in[i].Name < in[j].Name
		}
		return in[i].ID < in[j].ID
	})
}
