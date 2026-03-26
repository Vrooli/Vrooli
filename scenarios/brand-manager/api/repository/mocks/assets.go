package mocks

import (
	"context"
	"database/sql"
	"sync"

	"brand-manager/domain"
)

// AssetRepository is an in-memory mock implementing repository.AssetRepository.
type AssetRepository struct {
	mu     sync.RWMutex
	assets map[string]*domain.Asset // keyed by ID

	// Error overrides.
	CreateErr      error
	GetByIDErr     error
	ListByBrandErr error
	DeleteErr      error
}

// NewAssetRepository creates a ready-to-use mock AssetRepository.
func NewAssetRepository() *AssetRepository {
	return &AssetRepository{
		assets: make(map[string]*domain.Asset),
	}
}

// Seed adds an asset to the mock store (for test setup).
func (m *AssetRepository) Seed(a *domain.Asset) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *a
	m.assets[a.ID] = &cp
}

func (m *AssetRepository) Create(_ context.Context, a *domain.Asset) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *a
	m.assets[a.ID] = &cp
	return nil
}

func (m *AssetRepository) GetByID(_ context.Context, id string) (*domain.Asset, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assets[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *a
	return &cp, nil
}

func (m *AssetRepository) ListByBrandID(_ context.Context, brandID string) ([]*domain.Asset, error) {
	if m.ListByBrandErr != nil {
		return nil, m.ListByBrandErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Asset
	for _, a := range m.assets {
		if a.BrandID == brandID {
			cp := *a
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *AssetRepository) Delete(_ context.Context, id string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.assets[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.assets, id)
	return nil
}
