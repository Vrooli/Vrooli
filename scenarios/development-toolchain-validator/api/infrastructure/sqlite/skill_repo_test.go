package sqlite

import (
	"context"
	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
	"testing"
)

// createTestReference creates a reference for FK constraints in skill tests.
func createTestReference(t *testing.T, db *ReferenceRepository, ctx context.Context) *reference.Reference {
	t.Helper()
	ref, err := db.Create(ctx, reference.CreateInput{
		Slug:     "test-ref-" + t.Name(),
		Name:     "Test",
		Template: "t1",
		Path:     "/tmp/test",
	})
	if err != nil {
		t.Fatalf("create test reference: %v", err)
	}
	return ref
}

func TestSkillRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	repo := NewSkillRepository(db)
	ctx := context.Background()

	ref := createTestReference(t, refRepo, ctx)

	// Connect
	input := skill.ConnectInput{
		ReferenceID:      ref.ID,
		SkillID:          "api-steer",
		SkillVersion:     "1.0",
		SkillContentHash: "abc123",
	}
	conn, err := repo.Connect(ctx, input)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if conn.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if conn.SkillVersion != input.SkillVersion {
		t.Errorf("version = %q, want %q", conn.SkillVersion, input.SkillVersion)
	}

	// GetByID
	got, err := repo.GetByID(ctx, conn.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.SkillID != conn.SkillID {
		t.Errorf("GetByID skill_id = %q, want %q", got.SkillID, conn.SkillID)
	}

	// GetByReferenceAndSkill
	got, err = repo.GetByReferenceAndSkill(ctx, ref.ID, "api-steer")
	if err != nil {
		t.Fatalf("GetByReferenceAndSkill failed: %v", err)
	}
	if got.ID != conn.ID {
		t.Errorf("GetByReferenceAndSkill id = %q, want %q", got.ID, conn.ID)
	}

	// Update
	newVersion := "2.0"
	updated, err := repo.Update(ctx, conn.ID, skill.UpdateInput{SkillVersion: &newVersion})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.SkillVersion != newVersion {
		t.Errorf("Update version = %q, want %q", updated.SkillVersion, newVersion)
	}

	// List
	conns, err := repo.List(ctx, skill.ListOptions{ReferenceID: ref.ID})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(conns) != 1 {
		t.Errorf("List count = %d, want 1", len(conns))
	}

	// Disconnect
	if err := repo.Disconnect(ctx, conn.ID); err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	_, err = repo.GetByID(ctx, conn.ID)
	if err != skill.ErrNotFound {
		t.Errorf("expected ErrNotFound after disconnect, got %v", err)
	}
}

func TestSkillRepository_DisconnectByReferenceAndSkill(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	repo := NewSkillRepository(db)
	ctx := context.Background()

	ref := createTestReference(t, refRepo, ctx)

	_, err := repo.Connect(ctx, skill.ConnectInput{
		ReferenceID: ref.ID,
		SkillID:     "cli-steer",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	err = repo.DisconnectByReferenceAndSkill(ctx, ref.ID, "cli-steer")
	if err != nil {
		t.Fatalf("DisconnectByReferenceAndSkill: %v", err)
	}

	// Second disconnect should return not found
	err = repo.DisconnectByReferenceAndSkill(ctx, ref.ID, "cli-steer")
	if err != skill.ErrNotFound {
		t.Errorf("expected ErrNotFound on second disconnect, got %v", err)
	}
}

func TestSkillRepository_UniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	repo := NewSkillRepository(db)
	ctx := context.Background()

	ref := createTestReference(t, refRepo, ctx)

	input := skill.ConnectInput{
		ReferenceID: ref.ID,
		SkillID:     "api-steer",
	}
	_, err := repo.Connect(ctx, input)
	if err != nil {
		t.Fatalf("first connect: %v", err)
	}

	// Duplicate should fail
	_, err = repo.Connect(ctx, input)
	if err == nil {
		t.Fatal("expected error for duplicate connection")
	}
}

func TestSkillRepository_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	refRepo := NewReferenceRepository(db)
	repo := NewSkillRepository(db)
	ctx := context.Background()

	ref := createTestReference(t, refRepo, ctx)

	conn, err := repo.Connect(ctx, skill.ConnectInput{
		ReferenceID: ref.ID,
		SkillID:     "test-skill",
	})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}

	// Delete the reference - should cascade to skill connections
	if err := refRepo.Delete(ctx, ref.ID); err != nil {
		t.Fatalf("Delete reference: %v", err)
	}

	_, err = repo.GetByID(ctx, conn.ID)
	if err != skill.ErrNotFound {
		t.Errorf("expected ErrNotFound after cascade delete, got %v", err)
	}
}
