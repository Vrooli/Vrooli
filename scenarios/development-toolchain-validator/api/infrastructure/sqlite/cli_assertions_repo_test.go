package sqlite

import (
	"context"
	"testing"

	"development-toolchain-validator/domain/expectation"
)

func TestCLIAssertionsRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewCLIAssertionsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	// Create
	input := expectation.CreateCLIInput{
		ConnectionID:  conn.ID,
		Command:       "vrooli scenario audit --json",
		JSONPath:      "$.score",
		Operator:      expectation.OpGte,
		ExpectedValue: float64(80),
		Description:   "Score at least 80",
	}

	assertion, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if assertion.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if assertion.Operator != expectation.OpGte {
		t.Errorf("operator = %q, want %q", assertion.Operator, expectation.OpGte)
	}

	// Verify ExpectedValue round-trips
	if val, ok := assertion.ExpectedValue.(float64); !ok || val != 80 {
		t.Errorf("expected_value = %v (%T), want 80 (float64)", assertion.ExpectedValue, assertion.ExpectedValue)
	}

	// GetByID
	got, err := repo.GetByID(ctx, assertion.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Command != assertion.Command {
		t.Errorf("command = %q, want %q", got.Command, assertion.Command)
	}
	if got.Description != "Score at least 80" {
		t.Errorf("description = %q, want %q", got.Description, "Score at least 80")
	}

	// List
	assertions, err := repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(assertions) != 1 {
		t.Errorf("List count = %d, want 1", len(assertions))
	}

	// Delete
	if err := repo.Delete(ctx, assertion.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(ctx, assertion.ID)
	if err != expectation.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestCLIAssertionsRepository_NilExpectedValue(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewCLIAssertionsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	assertion, err := repo.Create(ctx, expectation.CreateCLIInput{
		ConnectionID: conn.ID,
		Command:      "test --json",
		JSONPath:     "$.exists",
		Operator:     expectation.OpExists,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, assertion.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ExpectedValue != nil {
		t.Errorf("expected nil ExpectedValue, got %v", got.ExpectedValue)
	}
}

func TestCLIAssertionsRepository_DeleteByConnection(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewCLIAssertionsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, expectation.CreateCLIInput{
			ConnectionID: conn.ID,
			Command:      "cmd" + string(rune('1'+i)),
			JSONPath:     "$.x",
			Operator:     expectation.OpEq,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	assertions, _ := repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if len(assertions) != 3 {
		t.Fatalf("expected 3, got %d", len(assertions))
	}

	if err := repo.DeleteByConnection(ctx, conn.ID); err != nil {
		t.Fatalf("DeleteByConnection: %v", err)
	}

	assertions, _ = repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if len(assertions) != 0 {
		t.Errorf("expected 0 after DeleteByConnection, got %d", len(assertions))
	}
}

func TestCLIAssertionsRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCLIAssertionsRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err != expectation.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = repo.Delete(ctx, "nonexistent")
	if err != expectation.ErrNotFound {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}
