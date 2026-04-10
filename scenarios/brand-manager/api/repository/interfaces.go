// Package repository defines storage contracts for brand-manager entities.
//
// Interfaces here decouple business logic from the underlying database,
// allowing SQLite in production and in-memory fakes in tests.
// DOC: docs/internal/SEAMS.md#1-handler--repository-primary-seam
// DOC: docs/concepts/ARCHITECTURE.md#key-patterns
package repository

import (
	"context"
	"time"

	"brand-manager/domain"
)

// nowUTC returns the current UTC time truncated to second precision and its
// RFC3339 string representation for database storage. This replaces the
// repeated format-then-parse pattern across repository implementations.
func nowUTC() (time.Time, string) {
	t := time.Now().UTC().Truncate(time.Second)
	return t, t.Format(time.RFC3339)
}

// BrandRepository manages persistence of Brand entities.
type BrandRepository interface {
	Create(ctx context.Context, brand *domain.Brand) error
	GetByID(ctx context.Context, id string) (*domain.Brand, error)
	List(ctx context.Context, filter domain.BrandFilter) ([]*domain.Brand, error)
	Update(ctx context.Context, brand *domain.Brand) error
	Delete(ctx context.Context, id string) error
}

// VersionRepository manages persistence of BrandVersion snapshots.
type VersionRepository interface {
	Create(ctx context.Context, version *domain.BrandVersion) error
	ListByBrandID(ctx context.Context, brandID string) ([]*domain.BrandVersion, error)
	GetByBrandIDAndVersion(ctx context.Context, brandID string, version int) (*domain.BrandVersion, error)
}

// AssignmentRepository manages persistence of brand-to-scenario assignments.
type AssignmentRepository interface {
	Create(ctx context.Context, assignment *domain.Assignment) error
	GetByScenario(ctx context.Context, scenarioName string) (*domain.Assignment, error)
	ListByBrandID(ctx context.Context, brandID string) ([]*domain.Assignment, error)
	Delete(ctx context.Context, id string) error
}

// AssetRepository manages persistence of brand asset file records.
type AssetRepository interface {
	Create(ctx context.Context, asset *domain.Asset) error
	GetByID(ctx context.Context, id string) (*domain.Asset, error)
	ListByBrandID(ctx context.Context, brandID string) ([]*domain.Asset, error)
	Delete(ctx context.Context, id string) error
}
