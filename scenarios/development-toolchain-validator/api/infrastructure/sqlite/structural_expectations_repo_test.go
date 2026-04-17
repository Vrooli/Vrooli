package sqlite

import (
	"context"
	"development-toolchain-validator/domain/expectation"
	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
	"testing"
)

// createTestConnection creates a reference + skill connection for FK constraints.
func createTestConnection(t *testing.T, db *ReferenceRepository, skillRepo *SkillRepository, ctx context.Context) (*reference.Reference, *skill.Connection) {
	t.Helper()
	ref := createTestReference(t, db, ctx)
	conn, err := skillRepo.Connect(ctx, skill.ConnectInput{
		ReferenceID: ref.ID,
		SkillID:     "test-skill-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("create test connection: %v", err)
	}
	return ref, conn
}

func TestStructuralExpectationsRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewStructuralExpectationsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	// Create
	input := expectation.CreateStructuralInput{
		ConnectionID:    conn.ID,
		Type:            expectation.TypeFile,
		Pattern:         "src/main.go",
		Required:        true,
		ExpectedContent: "",
		Description:     "Main entry point exists",
	}

	exp, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if exp.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if exp.Type != expectation.TypeFile {
		t.Errorf("type = %q, want %q", exp.Type, expectation.TypeFile)
	}
	if !exp.Required {
		t.Error("expected Required=true")
	}

	// GetByID
	got, err := repo.GetByID(ctx, exp.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Pattern != exp.Pattern {
		t.Errorf("pattern = %q, want %q", got.Pattern, exp.Pattern)
	}

	// List
	exps, err := repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(exps) != 1 {
		t.Errorf("List count = %d, want 1", len(exps))
	}

	// Delete
	if err := repo.Delete(ctx, exp.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err = repo.GetByID(ctx, exp.ID)
	if err != expectation.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStructuralExpectationsRepository_ContentSnippet(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewStructuralExpectationsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	exp, err := repo.Create(ctx, expectation.CreateStructuralInput{
		ConnectionID:    conn.ID,
		Type:            expectation.TypeContentSnippet,
		Pattern:         "src/config.go",
		Required:        false,
		ExpectedContent: "func LoadConfig()",
		Description:     "Config loader exists",
	})
	if err != nil {
		t.Fatalf("Create content_snippet: %v", err)
	}

	got, err := repo.GetByID(ctx, exp.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ExpectedContent != "func LoadConfig()" {
		t.Errorf("expected_content = %q, want %q", got.ExpectedContent, "func LoadConfig()")
	}
	if got.Required {
		t.Error("expected Required=false")
	}
}

func TestStructuralExpectationsRepository_DeleteByConnection(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	skillRepo := NewSkillRepository(db)
	repo := NewStructuralExpectationsRepository(db)
	ctx := context.Background()

	_, conn := createTestConnection(t, refRepo, skillRepo, ctx)

	// Create two expectations
	for _, pattern := range []string{"file1.go", "file2.go"} {
		_, err := repo.Create(ctx, expectation.CreateStructuralInput{
			ConnectionID: conn.ID,
			Type:         expectation.TypeFile,
			Pattern:      pattern,
			Required:     true,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	exps, _ := repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if len(exps) != 2 {
		t.Fatalf("expected 2 expectations, got %d", len(exps))
	}

	// Delete by connection
	if err := repo.DeleteByConnection(ctx, conn.ID); err != nil {
		t.Fatalf("DeleteByConnection: %v", err)
	}

	exps, _ = repo.List(ctx, expectation.ListOptions{ConnectionID: conn.ID})
	if len(exps) != 0 {
		t.Errorf("expected 0 after DeleteByConnection, got %d", len(exps))
	}
}
