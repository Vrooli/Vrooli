package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"brand-manager/domain"
	"brand-manager/internal/testutil"
	"brand-manager/repository"
)

// TestVersionCreate verifies version snapshot insertion.
// [REQ:BM-REQ-CRUD-VERSION]
func TestVersionCreate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	versionRepo := repository.NewSQLiteVersionRepository(db)
	ctx := context.Background()

	// Create parent brand first (FK constraint)
	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})

	v := &domain.BrandVersion{
		ID:       "v1",
		BrandID:  "b1",
		Version:  1,
		Snapshot: `{"name":"Brand"}`,
	}
	if err := versionRepo.Create(ctx, v); err != nil {
		t.Fatalf("Create version: %v", err)
	}
	if v.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

// TestVersionListByBrandID verifies listing versions in descending order.
// [REQ:BM-REQ-CRUD-VERSION]
func TestVersionListByBrandID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	versionRepo := repository.NewSQLiteVersionRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})

	for i := 1; i <= 3; i++ {
		versionRepo.Create(ctx, &domain.BrandVersion{
			ID:       "v" + string(rune('0'+i)),
			BrandID:  "b1",
			Version:  i,
			Snapshot: `{}`,
		})
	}

	versions, err := versionRepo.ListByBrandID(ctx, "b1")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(versions) != 3 {
		t.Fatalf("got %d versions, want 3", len(versions))
	}
	// Should be DESC order
	if versions[0].Version != 3 {
		t.Errorf("first version = %d, want 3 (DESC)", versions[0].Version)
	}
	if versions[2].Version != 1 {
		t.Errorf("last version = %d, want 1 (DESC)", versions[2].Version)
	}
}

// TestVersionListByBrandIDEmpty verifies empty result for unknown brand.
// [REQ:BM-REQ-CRUD-VERSION]
func TestVersionListByBrandIDEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	versionRepo := repository.NewSQLiteVersionRepository(db)

	versions, err := versionRepo.ListByBrandID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(versions) != 0 {
		t.Errorf("got %d versions, want 0", len(versions))
	}
}

// TestVersionGetByBrandIDAndVersion verifies retrieval of a specific version.
// [REQ:BM-REQ-CRUD-VERSION]
func TestVersionGetByBrandIDAndVersion(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	versionRepo := repository.NewSQLiteVersionRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})
	versionRepo.Create(ctx, &domain.BrandVersion{
		ID: "v1", BrandID: "b1", Version: 1, Snapshot: `{"v":1}`,
	})
	versionRepo.Create(ctx, &domain.BrandVersion{
		ID: "v2", BrandID: "b1", Version: 2, Snapshot: `{"v":2}`,
	})

	got, err := versionRepo.GetByBrandIDAndVersion(ctx, "b1", 2)
	if err != nil {
		t.Fatalf("GetByBrandIDAndVersion: %v", err)
	}
	if got.Snapshot != `{"v":2}` {
		t.Errorf("Snapshot = %q, want %q", got.Snapshot, `{"v":2}`)
	}
}

// TestVersionGetByBrandIDAndVersionNotFound verifies ErrNoRows for missing version.
// [REQ:BM-REQ-CRUD-VERSION]
func TestVersionGetByBrandIDAndVersionNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	versionRepo := repository.NewSQLiteVersionRepository(db)

	_, err := versionRepo.GetByBrandIDAndVersion(context.Background(), "b1", 99)
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}
