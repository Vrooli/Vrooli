package mocks

import (
	"context"
	"sync"

	"network-manager/internal/resolver"
)

type Repository struct {
	mu        sync.Mutex
	backends  map[string]resolver.BackendConfig
	upstreams map[string][]string
}

func NewRepository() *Repository {
	return &Repository{backends: map[string]resolver.BackendConfig{}, upstreams: map[string][]string{}}
}

func (r *Repository) SaveBackend(_ context.Context, cfg resolver.BackendConfig) (resolver.BackendConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[cfg.Backend] = cfg
	return cfg, nil
}

func (r *Repository) GetBackend(_ context.Context, backend string) (resolver.BackendConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, ok := r.backends[backend]
	if !ok {
		return resolver.BackendConfig{}, resolver.ErrNotFound
	}
	return cfg, nil
}

func (r *Repository) UpdateUpstreams(_ context.Context, backend string, upstreams []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upstreams[backend] = append([]string(nil), upstreams...)
	return nil
}

func (r *Repository) GetUpstreams(_ context.Context, backend string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.upstreams[backend]...), nil
}

var _ resolver.Repository = (*Repository)(nil)
