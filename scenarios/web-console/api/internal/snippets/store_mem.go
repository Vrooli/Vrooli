package snippets

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemStore struct {
	mu       sync.RWMutex
	snippets map[string]Snippet
}

func NewMemStore() *MemStore { return &MemStore{snippets: make(map[string]Snippet)} }

func (m *MemStore) List(context.Context) ([]Snippet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Snippet, 0, len(m.snippets))
	for _, snippet := range m.snippets {
		out = append(out, snippet)
	}
	sortSnippets(out)
	return out, nil
}

func (m *MemStore) Upsert(_ context.Context, req UpsertRequest) (Snippet, error) {
	if err := req.Validate(); err != nil {
		return Snippet{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	id := req.ID
	if id == "" {
		id = uuid.NewString()
	}
	now := FormatTime(time.Now())
	snippet := Snippet{
		ID: id, Name: req.Name, Body: req.Body, Color: req.Color,
		Pinned: req.Pinned, SortOrder: req.SortOrder,
		CreatedAt: now, UpdatedAt: now,
	}
	if existing, ok := m.snippets[id]; ok {
		snippet.CreatedAt = existing.CreatedAt
		snippet.UseCount = existing.UseCount
		snippet.LastUsedAt = existing.LastUsedAt
		if !req.HasPinned {
			snippet.Pinned = existing.Pinned
		}
	}
	m.snippets[id] = snippet
	return snippet, nil
}

func (m *MemStore) Delete(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.snippets[id]; !ok {
		return false, nil
	}
	delete(m.snippets, id)
	return true, nil
}

func (m *MemStore) Touch(_ context.Context, id string, now time.Time) (Snippet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snippet, ok := m.snippets[id]
	if !ok {
		return Snippet{}, ErrSnippetNotFound
	}
	snippet.UseCount++
	snippet.LastUsedAt = FormatTime(now)
	snippet.UpdatedAt = snippet.LastUsedAt
	m.snippets[id] = snippet
	return snippet, nil
}

func sortSnippets(snippets []Snippet) {
	sort.Slice(snippets, func(i, j int) bool {
		a, b := snippets[i], snippets[j]
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		if a.LastUsedAt != b.LastUsedAt {
			return a.LastUsedAt > b.LastUsedAt
		}
		if a.UseCount != b.UseCount {
			return a.UseCount > b.UseCount
		}
		return a.ID < b.ID
	})
}
