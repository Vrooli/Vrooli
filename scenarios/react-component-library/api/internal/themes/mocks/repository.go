// Package mocks holds themes-domain test fakes.
package mocks

import (
	"context"
	"sort"
	"sync"

	"react-component-library/internal/themes"
)

// FakeRepository satisfies themes.Repository for service and handler
// tests.
type FakeRepository struct {
	mu        sync.Mutex
	items     map[string]themes.Theme
	UpsertErr error
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{items: map[string]themes.Theme{}}
}

var _ themes.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) UpsertBuiltin(_ context.Context, t themes.Theme) error {
	if f.UpsertErr != nil {
		return f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := themes.Theme{ID: t.ID, Name: t.Name, Source: "builtin", Tokens: map[string]string{}}
	for k, v := range t.Tokens {
		cp.Tokens[k] = v
	}
	f.items[t.ID] = cp
	return nil
}

func (f *FakeRepository) ReplaceBuiltins(ctx context.Context, items []themes.Theme) error {
	f.mu.Lock()
	f.items = map[string]themes.Theme{}
	f.mu.Unlock()
	for _, item := range items {
		if err := f.UpsertBuiltin(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (f *FakeRepository) GetBuiltin(_ context.Context, id string) (themes.Theme, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.items[id]
	if !ok {
		return themes.Theme{}, themes.ErrThemeNotFound{ID: id}
	}
	return t, nil
}

func (f *FakeRepository) ListBuiltins(_ context.Context) ([]themes.Theme, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.items))
	for id := range f.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]themes.Theme, 0, len(ids))
	for _, id := range ids {
		out = append(out, f.items[id])
	}
	return out, nil
}

func (f *FakeRepository) CountBuiltins(_ context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.items), nil
}
