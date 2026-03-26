package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"brand-manager/domain"
	"brand-manager/internal/testutil"
	"brand-manager/repository"
)

// TestCreateAndGetBrand verifies brand creation and retrieval.
// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-CRUD-READ]
func TestCreateAndGetBrand(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	brand := &domain.Brand{
		ID:          "test-id-1",
		Name:        "Test Brand",
		Description: "A test brand",
		Identity:    &domain.Identity{DisplayName: "Test Display"},
		Colors:      &domain.Colors{Primary: "#ff0000"},
		Notes:       "some notes",
	}

	// Create
	if err := repo.Create(ctx, brand); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Get
	got, err := repo.GetByID(ctx, "test-id-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Name != "Test Brand" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Brand")
	}
	if got.Identity == nil || got.Identity.DisplayName != "Test Display" {
		t.Errorf("Identity.DisplayName mismatch")
	}
	if got.Colors == nil || got.Colors.Primary != "#ff0000" {
		t.Errorf("Colors.Primary mismatch")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
}

// TestListBrands verifies brand listing with filters.
// [REQ:BM-REQ-CRUD-READ]
func TestListBrands(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	// Create two brands
	repo.Create(ctx, &domain.Brand{ID: "b1", Name: "Alpha Brand"})
	repo.Create(ctx, &domain.Brand{ID: "b2", Name: "Beta Brand"})

	// List all
	brands, err := repo.List(ctx, domain.BrandFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(brands) != 2 {
		t.Errorf("List returned %d brands, want 2", len(brands))
	}

	// Filter by name
	brands, err = repo.List(ctx, domain.BrandFilter{NameContains: "Alpha"})
	if err != nil {
		t.Fatalf("List with filter: %v", err)
	}
	if len(brands) != 1 {
		t.Errorf("Filtered list returned %d brands, want 1", len(brands))
	}

	// Limit
	brands, err = repo.List(ctx, domain.BrandFilter{Limit: 1})
	if err != nil {
		t.Fatalf("List with limit: %v", err)
	}
	if len(brands) != 1 {
		t.Errorf("Limited list returned %d brands, want 1", len(brands))
	}
}

// TestUpdateBrand verifies brand update with version increment.
// [REQ:BM-REQ-CRUD-UPDATE]
func TestUpdateBrand(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "u1", Name: "Original"})

	brand, _ := repo.GetByID(ctx, "u1")
	brand.Name = "Updated"
	if err := repo.Update(ctx, brand); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, "u1")
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}
}

// TestDeleteBrand verifies brand deletion.
// [REQ:BM-REQ-CRUD-READ]
func TestDeleteBrand(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "d1", Name: "ToDelete"})

	if err := repo.Delete(ctx, "d1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(ctx, "d1")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows after delete, got %v", err)
	}
}

// TestEmptyFacetsHydrateAsNil verifies that brands created without optional facets
// return nil pointers (not empty structs) on retrieval. This is a key decision boundary:
// nil means "not set", which the API serializes as absent rather than empty objects.
// [REQ:BM-REQ-CRUD-READ]
func TestEmptyFacetsHydrateAsNil(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	// Create brand with no optional facets
	repo.Create(ctx, &domain.Brand{ID: "empty-1", Name: "Minimal"})

	got, err := repo.GetByID(ctx, "empty-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Identity != nil {
		t.Error("Identity should be nil for brand without identity data")
	}
	if got.Colors != nil {
		t.Error("Colors should be nil for brand without color data")
	}
	if got.Typography != nil {
		t.Error("Typography should be nil for brand without typography data")
	}
	if got.Voice != nil {
		t.Error("Voice should be nil for brand without voice data")
	}
}

// TestDeleteBrandNotFound verifies deleting a non-existent brand returns ErrNoRows.
// [REQ:BM-REQ-CRUD-READ]
func TestDeleteBrandNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}
