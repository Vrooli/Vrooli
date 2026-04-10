// Package mocks provides in-memory mock implementations of repository interfaces
// for unit testing handlers without a database dependency.
package mocks

import (
	"context"
	"database/sql"
	"strings"
	"sync"

	"brand-manager/domain"
)

// BrandRepository is an in-memory mock implementing repository.BrandRepository.
type BrandRepository struct {
	mu     sync.RWMutex
	brands map[string]*domain.Brand

	// Error overrides — set these to force specific error returns.
	CreateErr error
	GetErr    error
	ListErr   error
	UpdateErr error
	DeleteErr error
}

// NewBrandRepository creates a ready-to-use mock BrandRepository.
func NewBrandRepository() *BrandRepository {
	return &BrandRepository{brands: make(map[string]*domain.Brand)}
}

func (m *BrandRepository) Create(_ context.Context, brand *domain.Brand) error {
	if m.CreateErr != nil {
		return m.CreateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	brand.Version = 1
	cp := *brand
	m.brands[brand.ID] = &cp
	return nil
}

func (m *BrandRepository) GetByID(_ context.Context, id string) (*domain.Brand, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.brands[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	cp := *b
	return &cp, nil
}

func (m *BrandRepository) List(_ context.Context, filter domain.BrandFilter) ([]*domain.Brand, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*domain.Brand
	for _, b := range m.brands {
		if filter.NameContains != "" && !strings.Contains(b.Name, filter.NameContains) {
			continue
		}
		cp := *b
		result = append(result, &cp)
	}
	if filter.Offset > 0 && filter.Offset < len(result) {
		result = result[filter.Offset:]
	} else if filter.Offset >= len(result) {
		result = nil
	}
	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}
	return result, nil
}

func (m *BrandRepository) Update(_ context.Context, brand *domain.Brand) error {
	if m.UpdateErr != nil {
		return m.UpdateErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.brands[brand.ID]; !ok {
		return sql.ErrNoRows
	}
	brand.Version++
	cp := *brand
	m.brands[brand.ID] = &cp
	return nil
}

func (m *BrandRepository) Delete(_ context.Context, id string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.brands[id]; !ok {
		return sql.ErrNoRows
	}
	delete(m.brands, id)
	return nil
}

// Seed adds a brand directly to the mock store (bypasses Create logic).
func (m *BrandRepository) Seed(brand *domain.Brand) *BrandRepository {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *brand
	m.brands[brand.ID] = &cp
	return m
}

// SeedBrand is a convenience for seeding a brand with common fields.
func (m *BrandRepository) SeedBrand(id, name, primary, secondary string) *BrandRepository {
	return m.Seed(&domain.Brand{
		ID:   id,
		Name: name,
		Colors: &domain.Colors{
			Primary:    primary,
			Secondary:  secondary,
			Background: "#ffffff",
			Text:       "#000000",
		},
		Identity: &domain.Identity{
			DisplayName: name,
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Inter",
		},
		Version: 1,
	})
}
