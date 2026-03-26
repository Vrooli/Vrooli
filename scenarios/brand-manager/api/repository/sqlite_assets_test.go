package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"brand-manager/domain"
	"brand-manager/internal/testutil"
	"brand-manager/repository"
)

// [REQ:BM-REQ-STORE-ASSETS]

func TestAssetCreateAndGetByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)
	ctx := context.Background()

	asset := &domain.Asset{
		ID:       "asset-1",
		BrandID:  "brand-1",
		Filename: "logo.png",
		MimeType: "image/png",
		FilePath: "/tmp/assets/brand-1/logo.png",
		Size:     1024,
	}

	// Need a brand for the foreign key
	seedBrand(t, db, "brand-1")

	if err := repo.Create(ctx, asset); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, "asset-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Filename != "logo.png" {
		t.Errorf("Filename = %q, want %q", got.Filename, "logo.png")
	}
	if got.MimeType != "image/png" {
		t.Errorf("MimeType = %q, want %q", got.MimeType, "image/png")
	}
	if got.Size != 1024 {
		t.Errorf("Size = %d, want 1024", got.Size)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestAssetGetByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("error = %v, want sql.ErrNoRows", err)
	}
}

func TestAssetListByBrandID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)
	ctx := context.Background()

	seedBrand(t, db, "brand-1")
	seedBrand(t, db, "brand-2")

	for _, a := range []*domain.Asset{
		{ID: "a1", BrandID: "brand-1", Filename: "logo.png", MimeType: "image/png", FilePath: "/tmp/1", Size: 100},
		{ID: "a2", BrandID: "brand-1", Filename: "icon.svg", MimeType: "image/svg+xml", FilePath: "/tmp/2", Size: 200},
		{ID: "a3", BrandID: "brand-2", Filename: "other.png", MimeType: "image/png", FilePath: "/tmp/3", Size: 300},
	} {
		if err := repo.Create(ctx, a); err != nil {
			t.Fatalf("Create %s: %v", a.ID, err)
		}
	}

	assets, err := repo.ListByBrandID(ctx, "brand-1")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(assets) != 2 {
		t.Errorf("got %d assets, want 2", len(assets))
	}
}

func TestAssetListByBrandID_Empty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)

	assets, err := repo.ListByBrandID(context.Background(), "no-brand")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if assets != nil && len(assets) != 0 {
		t.Errorf("got %d assets, want 0", len(assets))
	}
}

func TestAssetDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)
	ctx := context.Background()

	seedBrand(t, db, "brand-1")

	asset := &domain.Asset{
		ID: "asset-del", BrandID: "brand-1", Filename: "x.png",
		MimeType: "image/png", FilePath: "/tmp/x", Size: 10,
	}
	if err := repo.Create(ctx, asset); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, "asset-del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.GetByID(ctx, "asset-del")
	if err != sql.ErrNoRows {
		t.Errorf("after delete: error = %v, want sql.ErrNoRows", err)
	}
}

func TestAssetDelete_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteAssetRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("error = %v, want sql.ErrNoRows", err)
	}
}

// seedBrand inserts a minimal brand row for foreign key satisfaction.
func seedBrand(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO brands (id, name, version, created_at, updated_at) VALUES (?, ?, 1, datetime('now'), datetime('now'))`,
		id, "Brand "+id,
	)
	if err != nil {
		t.Fatalf("seed brand %s: %v", id, err)
	}
}
