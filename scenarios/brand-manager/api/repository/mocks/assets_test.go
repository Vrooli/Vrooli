package mocks

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"brand-manager/domain"
)

func TestAssetRepository_CreateAndGetByID(t *testing.T) {
	repo := NewAssetRepository()
	ctx := context.Background()

	asset := &domain.Asset{ID: "a1", BrandID: "b1", Filename: "logo.png"}
	if err := repo.Create(ctx, asset); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Filename != "logo.png" {
		t.Errorf("Filename = %q, want %q", got.Filename, "logo.png")
	}
}

func TestAssetRepository_GetByID_NotFound(t *testing.T) {
	repo := NewAssetRepository()
	_, err := repo.GetByID(context.Background(), "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestAssetRepository_ListByBrandID(t *testing.T) {
	repo := NewAssetRepository()
	ctx := context.Background()

	repo.Seed(&domain.Asset{ID: "a1", BrandID: "b1"})
	repo.Seed(&domain.Asset{ID: "a2", BrandID: "b1"})
	repo.Seed(&domain.Asset{ID: "a3", BrandID: "b2"})

	list, err := repo.ListByBrandID(ctx, "b1")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 assets for b1, got %d", len(list))
	}
}

func TestAssetRepository_ListByBrandID_Empty(t *testing.T) {
	repo := NewAssetRepository()
	list, err := repo.ListByBrandID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 assets, got %d", len(list))
	}
}

func TestAssetRepository_Delete(t *testing.T) {
	repo := NewAssetRepository()
	ctx := context.Background()

	repo.Seed(&domain.Asset{ID: "a1", BrandID: "b1"})

	if err := repo.Delete(ctx, "a1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(ctx, "a1")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Error("expected asset to be deleted")
	}
}

func TestAssetRepository_Delete_NotFound(t *testing.T) {
	repo := NewAssetRepository()
	err := repo.Delete(context.Background(), "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestAssetRepository_ErrorOverrides(t *testing.T) {
	ctx := context.Background()
	forced := errors.New("forced")

	t.Run("CreateErr", func(t *testing.T) {
		repo := NewAssetRepository()
		repo.CreateErr = forced
		if err := repo.Create(ctx, &domain.Asset{ID: "a1"}); !errors.Is(err, forced) {
			t.Errorf("expected forced error, got %v", err)
		}
	})

	t.Run("GetByIDErr", func(t *testing.T) {
		repo := NewAssetRepository()
		repo.Seed(&domain.Asset{ID: "a1"})
		repo.GetByIDErr = forced
		_, err := repo.GetByID(ctx, "a1")
		if !errors.Is(err, forced) {
			t.Errorf("expected forced error, got %v", err)
		}
	})

	t.Run("ListByBrandErr", func(t *testing.T) {
		repo := NewAssetRepository()
		repo.ListByBrandErr = forced
		_, err := repo.ListByBrandID(ctx, "b1")
		if !errors.Is(err, forced) {
			t.Errorf("expected forced error, got %v", err)
		}
	})

	t.Run("DeleteErr", func(t *testing.T) {
		repo := NewAssetRepository()
		repo.DeleteErr = forced
		if err := repo.Delete(ctx, "a1"); !errors.Is(err, forced) {
			t.Errorf("expected forced error, got %v", err)
		}
	})
}

func TestAssetRepository_Seed_CopiesData(t *testing.T) {
	repo := NewAssetRepository()
	original := &domain.Asset{ID: "a1", Filename: "original.png"}
	repo.Seed(original)

	// Modify the original after seeding
	original.Filename = "modified.png"

	got, _ := repo.GetByID(context.Background(), "a1")
	if got.Filename != "original.png" {
		t.Error("Seed should copy data, not store reference")
	}
}
