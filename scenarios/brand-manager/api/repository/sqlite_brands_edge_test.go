package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"brand-manager/domain"
	"brand-manager/internal/testutil"
	"brand-manager/repository"
)

// Edge case tests for SQLiteBrandRepository — filtering, pagination, update boundaries.
// [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-CRUD-UPDATE]

func TestListBrands_NameFilter_CaseInsensitive(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "b1", Name: "Alpha Brand"})
	repo.Create(ctx, &domain.Brand{ID: "b2", Name: "ALPHA UPPERCASE"})
	repo.Create(ctx, &domain.Brand{ID: "b3", Name: "Beta Brand"})

	// SQLite LIKE is case-insensitive by default for ASCII
	brands, err := repo.List(ctx, domain.BrandFilter{NameContains: "alpha"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(brands) != 2 {
		t.Errorf("expected 2 brands matching 'alpha', got %d", len(brands))
	}
}

func TestListBrands_NameFilter_NoMatch(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "b1", Name: "Alpha Brand"})

	brands, err := repo.List(ctx, domain.BrandFilter{NameContains: "zzzzz"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(brands) != 0 {
		t.Errorf("expected 0 brands, got %d", len(brands))
	}
}

func TestListBrands_Pagination(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		repo.Create(ctx, &domain.Brand{
			ID:   "b" + string(rune('a'+i)),
			Name: "Brand " + string(rune('A'+i)),
		})
	}

	// Page 1: limit 2, offset 0
	page1, err := repo.List(ctx, domain.BrandFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("page1: expected 2, got %d", len(page1))
	}

	// Page 2: limit 2, offset 2
	page2, err := repo.List(ctx, domain.BrandFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("page2: expected 2, got %d", len(page2))
	}

	// Page 3: limit 2, offset 4 (only 1 remaining)
	page3, err := repo.List(ctx, domain.BrandFilter{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("List page3: %v", err)
	}
	if len(page3) != 1 {
		t.Errorf("page3: expected 1, got %d", len(page3))
	}

	// Ensure no overlap between pages
	page1IDs := map[string]bool{}
	for _, b := range page1 {
		page1IDs[b.ID] = true
	}
	for _, b := range page2 {
		if page1IDs[b.ID] {
			t.Errorf("page2 contains page1 brand: %s", b.ID)
		}
	}
}

func TestListBrands_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)

	brands, err := repo.List(context.Background(), domain.BrandFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(brands) != 0 {
		t.Errorf("expected 0 brands in empty DB, got %d", len(brands))
	}
}

func TestUpdateBrand_MultipleVersionIncrements(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "v1", Name: "Version Test"})

	brand, _ := repo.GetByID(ctx, "v1")
	if brand.Version != 1 {
		t.Fatalf("initial version = %d, want 1", brand.Version)
	}

	// Update 3 times
	for i := 0; i < 3; i++ {
		brand.Name = "Updated " + string(rune('A'+i))
		if err := repo.Update(ctx, brand); err != nil {
			t.Fatalf("Update %d: %v", i+1, err)
		}
	}

	got, _ := repo.GetByID(ctx, "v1")
	if got.Version != 4 {
		t.Errorf("version after 3 updates = %d, want 4", got.Version)
	}
	if got.Name != "Updated C" {
		t.Errorf("name = %q, want 'Updated C'", got.Name)
	}
}

func TestUpdateBrand_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)

	err := repo.Update(context.Background(), &domain.Brand{ID: "nonexistent", Name: "X"})
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows for missing brand update, got %v", err)
	}
}

func TestCreateBrand_SetsTimestamps(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	brand := &domain.Brand{ID: "ts1", Name: "Timestamp Test"}
	repo.Create(ctx, brand)

	if brand.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set after Create")
	}
	if brand.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set after Create")
	}
	if !brand.CreatedAt.Equal(brand.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal on creation")
	}
}

func TestUpdateBrand_PreservesAllFacets(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	original := &domain.Brand{
		ID:          "full1",
		Name:        "Full Brand",
		Description: "Original description",
		Identity: &domain.Identity{
			DisplayName: "Display",
			Tagline:     "Tag",
			LogoPath:    "/logo.png",
			FaviconPath: "/fav.ico",
		},
		Colors: &domain.Colors{
			Primary:    "#111",
			Secondary:  "#222",
			Background: "#fff",
			Text:       "#000",
		},
		Typography: &domain.Typography{
			HeadingFont: "Inter",
			BodyFont:    "Roboto",
		},
		Voice: &domain.Voice{
			Tone:     "professional",
			Style:    "concise",
			Keywords: []string{"reliable"},
		},
		Notes: "Important notes",
	}
	repo.Create(ctx, original)

	// Update only the name
	brand, _ := repo.GetByID(ctx, "full1")
	brand.Name = "Renamed"
	repo.Update(ctx, brand)

	got, _ := repo.GetByID(ctx, "full1")
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", got.Name)
	}
	// All other facets should be preserved
	if got.Identity == nil || got.Identity.DisplayName != "Display" {
		t.Error("Identity lost after update")
	}
	if got.Colors == nil || got.Colors.Primary != "#111" {
		t.Error("Colors lost after update")
	}
	if got.Typography == nil || got.Typography.HeadingFont != "Inter" {
		t.Error("Typography lost after update")
	}
	if got.Voice == nil || got.Voice.Tone != "professional" {
		t.Error("Voice lost after update")
	}
	if got.Notes != "Important notes" {
		t.Error("Notes lost after update")
	}
}

func TestCreateBrand_DuplicateID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteBrandRepository(db)
	ctx := context.Background()

	repo.Create(ctx, &domain.Brand{ID: "dup1", Name: "First"})
	err := repo.Create(ctx, &domain.Brand{ID: "dup1", Name: "Second"})
	if err == nil {
		t.Error("expected error when creating brand with duplicate ID")
	}
}
