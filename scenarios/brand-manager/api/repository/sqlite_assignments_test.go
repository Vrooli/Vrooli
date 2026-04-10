package repository_test

import (
	"context"
	"database/sql"
	"testing"

	"brand-manager/domain"
	"brand-manager/internal/testutil"
	"brand-manager/repository"
)

// TestAssignmentCreate verifies assignment insertion and that applied_at is tracked.
// [REQ:BM-REQ-ASSIGN-LINK] [REQ:BM-REQ-ASSIGN-TRACK]
func TestAssignmentCreate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})

	a := &domain.Assignment{
		ID:           "a1",
		BrandID:      "b1",
		ScenarioName: "test-scenario",
		BrandVersion: 1,
		Elements:     []string{"logo", "colors"},
	}
	if err := assignRepo.Create(ctx, a); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.AppliedAt.IsZero() {
		t.Error("expected AppliedAt to be set")
	}
}

// TestAssignmentGetByScenario verifies retrieval by scenario name.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentGetByScenario(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})
	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a1", BrandID: "b1", ScenarioName: "my-scenario", BrandVersion: 1,
		Elements: []string{"logo"},
	})

	got, err := assignRepo.GetByScenario(ctx, "my-scenario")
	if err != nil {
		t.Fatalf("GetByScenario: %v", err)
	}
	if got.BrandID != "b1" {
		t.Errorf("BrandID = %q, want %q", got.BrandID, "b1")
	}
	if len(got.Elements) != 1 || got.Elements[0] != "logo" {
		t.Errorf("Elements = %v, want [logo]", got.Elements)
	}
}

// TestAssignmentGetByScenarioNotFound verifies ErrNoRows for missing scenario.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentGetByScenarioNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)

	_, err := assignRepo.GetByScenario(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

// TestAssignmentReplaceOnSameScenario verifies INSERT OR REPLACE behaviour.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentReplaceOnSameScenario(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand1"})
	brandRepo.Create(ctx, &domain.Brand{ID: "b2", Name: "Brand2"})

	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a1", BrandID: "b1", ScenarioName: "same-scenario", BrandVersion: 1,
	})
	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a2", BrandID: "b2", ScenarioName: "same-scenario", BrandVersion: 1,
	})

	got, err := assignRepo.GetByScenario(ctx, "same-scenario")
	if err != nil {
		t.Fatalf("GetByScenario: %v", err)
	}
	if got.BrandID != "b2" {
		t.Errorf("BrandID = %q, want %q (latest)", got.BrandID, "b2")
	}
}

// TestAssignmentListByBrandID verifies one brand can be assigned to multiple scenarios.
// [REQ:BM-REQ-ASSIGN-MULTI] [REQ:BM-REQ-ASSIGN-TRACK]
func TestAssignmentListByBrandID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})
	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a1", BrandID: "b1", ScenarioName: "scenario-a", BrandVersion: 1,
	})
	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a2", BrandID: "b1", ScenarioName: "scenario-b", BrandVersion: 1,
	})

	list, err := assignRepo.ListByBrandID(ctx, "b1")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("got %d assignments, want 2", len(list))
	}
}

// TestAssignmentDelete verifies assignment deletion.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentDelete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})
	assignRepo.Create(ctx, &domain.Assignment{
		ID: "a1", BrandID: "b1", ScenarioName: "scenario-a", BrandVersion: 1,
	})

	if err := assignRepo.Delete(ctx, "a1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := assignRepo.GetByScenario(ctx, "scenario-a")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows after delete, got %v", err)
	}
}

// TestAssignmentDeleteNotFound verifies ErrNoRows for missing assignment.
// [REQ:BM-REQ-ASSIGN-LINK]
func TestAssignmentDeleteNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)

	err := assignRepo.Delete(context.Background(), "nonexistent")
	if err != sql.ErrNoRows {
		t.Errorf("expected ErrNoRows, got %v", err)
	}
}

// TestAssignmentTrackAppliedAt verifies that applied_at timestamps are recorded
// and updated when a brand is reassigned to a scenario.
// [REQ:BM-REQ-ASSIGN-TRACK]
func TestAssignmentTrackAppliedAt(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Brand"})

	// First assignment
	a1 := &domain.Assignment{
		ID: "a1", BrandID: "b1", ScenarioName: "tracked-scenario", BrandVersion: 1,
		Elements: []string{"colors"},
	}
	if err := assignRepo.Create(ctx, a1); err != nil {
		t.Fatalf("Create a1: %v", err)
	}
	if a1.AppliedAt.IsZero() {
		t.Fatal("AppliedAt should be set after creation")
	}
	firstAppliedAt := a1.AppliedAt

	// Reassign same scenario (replace)
	a2 := &domain.Assignment{
		ID: "a2", BrandID: "b1", ScenarioName: "tracked-scenario", BrandVersion: 2,
		Elements: []string{"colors", "typography"},
	}
	if err := assignRepo.Create(ctx, a2); err != nil {
		t.Fatalf("Create a2: %v", err)
	}

	got, err := assignRepo.GetByScenario(ctx, "tracked-scenario")
	if err != nil {
		t.Fatalf("GetByScenario: %v", err)
	}
	if got.BrandVersion != 2 {
		t.Errorf("BrandVersion = %d, want 2", got.BrandVersion)
	}
	if got.AppliedAt.IsZero() {
		t.Error("replaced assignment should have AppliedAt set")
	}
	if !got.AppliedAt.After(firstAppliedAt) && !got.AppliedAt.Equal(firstAppliedAt) {
		t.Error("replaced assignment AppliedAt should be >= first")
	}
}

// TestAssignmentMultiScenario verifies one brand can be assigned to multiple
// scenarios simultaneously and all tracked via ListByBrandID.
// [REQ:BM-REQ-ASSIGN-MULTI] [REQ:BM-REQ-ASSIGN-TRACK]
func TestAssignmentMultiScenario(t *testing.T) {
	db := testutil.SetupTestDB(t)
	brandRepo := repository.NewSQLiteBrandRepository(db)
	assignRepo := repository.NewSQLiteAssignmentRepository(db)
	ctx := context.Background()

	brandRepo.Create(ctx, &domain.Brand{ID: "b1", Name: "Shared Brand"})

	scenarios := []string{"scenario-a", "scenario-b", "scenario-c"}
	for i, s := range scenarios {
		assignRepo.Create(ctx, &domain.Assignment{
			ID: "multi-" + s, BrandID: "b1", ScenarioName: s,
			BrandVersion: 1, Elements: []string{"colors"},
		})
		_ = i
	}

	// All three scenarios should be listed
	list, err := assignRepo.ListByBrandID(ctx, "b1")
	if err != nil {
		t.Fatalf("ListByBrandID: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("got %d assignments, want 3", len(list))
	}

	// Each scenario should be independently retrievable
	for _, s := range scenarios {
		got, err := assignRepo.GetByScenario(ctx, s)
		if err != nil {
			t.Errorf("GetByScenario(%s): %v", s, err)
			continue
		}
		if got.BrandID != "b1" {
			t.Errorf("scenario %s: BrandID = %q, want b1", s, got.BrandID)
		}
		if got.AppliedAt.IsZero() {
			t.Errorf("scenario %s: AppliedAt not tracked", s)
		}
	}
}
