package mocks

import (
	"context"
	"sync"

	"network-manager/internal/adapters"
)

type Repository struct {
	mu      sync.Mutex
	reports []adapters.Report
}

func NewRepository() *Repository { return &Repository{} }

func (r *Repository) SaveReport(_ context.Context, report adapters.Report) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = append([]adapters.Report{report}, r.reports...)
	return nil
}

func (r *Repository) LatestCapabilities(context.Context) ([]adapters.Capability, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.reports) == 0 {
		return nil, adapters.ErrNotFound
	}
	out := make([]adapters.Capability, len(r.reports[0].Capabilities))
	copy(out, r.reports[0].Capabilities)
	return out, nil
}

func (r *Repository) LatestPlatformSummary(context.Context) (adapters.PlatformSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.reports) == 0 {
		return adapters.PlatformSummary{}, adapters.ErrNotFound
	}
	return r.reports[0].Platform, nil
}

func (r *Repository) Reports() []adapters.Report {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]adapters.Report, len(r.reports))
	copy(out, r.reports)
	return out
}

var _ adapters.Repository = (*Repository)(nil)
