package mocks

import (
	"context"
	"database/sql"
	"sync"

	"brand-manager/domain"
)

// VersionRepository is an in-memory mock implementing repository.VersionRepository.
type VersionRepository struct {
	mu       sync.RWMutex
	versions map[string][]*domain.BrandVersion // keyed by brand_id

	// Error overrides.
	CreateErr           error
	ListByBrandIDErr    error
	GetByBrandAndVerErr error
}

// NewVersionRepository creates a ready-to-use mock VersionRepository.
func NewVersionRepository() *VersionRepository {
	return &VersionRepository{versions: make(map[string][]*domain.BrandVersion)}
}

func (m *VersionRepository) Create(_ context.Context, v *domain.BrandVersion) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *v
	m.versions[v.BrandID] = append(m.versions[v.BrandID], &cp)
	return nil
}

func (m *VersionRepository) ListByBrandID(_ context.Context, brandID string) ([]*domain.BrandVersion, error) {
	if m.ListByBrandIDErr != nil {
		return nil, m.ListByBrandIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	vs := m.versions[brandID]
	result := make([]*domain.BrandVersion, len(vs))
	for i, v := range vs {
		cp := *v
		result[i] = &cp
	}
	return result, nil
}

func (m *VersionRepository) GetByBrandIDAndVersion(_ context.Context, brandID string, version int) (*domain.BrandVersion, error) {
	if m.GetByBrandAndVerErr != nil {
		return nil, m.GetByBrandAndVerErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.versions[brandID] {
		if v.Version == version {
			cp := *v
			return &cp, nil
		}
	}
	return nil, sql.ErrNoRows
}
