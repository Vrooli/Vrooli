package sqlite

import (
	"context"
	"testing"

	"development-toolchain-validator/domain/reference"
)

func TestReferenceRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReferenceRepository(db)
	ctx := context.Background()

	// Create
	input := reference.CreateInput{
		Slug:        "test-ref",
		Name:        "Test Reference",
		Template:    "react-vite",
		Path:        "/tmp/test-scenario",
		Description: "A test reference",
	}

	ref, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if ref.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if ref.Slug != input.Slug {
		t.Errorf("slug = %q, want %q", ref.Slug, input.Slug)
	}
	if ref.Description != input.Description {
		t.Errorf("description = %q, want %q", ref.Description, input.Description)
	}

	// GetByID
	got, err := repo.GetByID(ctx, ref.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Slug != ref.Slug {
		t.Errorf("GetByID slug = %q, want %q", got.Slug, ref.Slug)
	}

	// GetBySlug
	got, err = repo.GetBySlug(ctx, ref.Slug)
	if err != nil {
		t.Fatalf("GetBySlug failed: %v", err)
	}
	if got.ID != ref.ID {
		t.Errorf("GetBySlug id = %q, want %q", got.ID, ref.ID)
	}

	// Update
	newName := "Updated Name"
	updated, err := repo.Update(ctx, ref.ID, reference.UpdateInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Name != newName {
		t.Errorf("Update name = %q, want %q", updated.Name, newName)
	}

	// List
	refs, err := repo.List(ctx, reference.ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(refs) != 1 {
		t.Errorf("List count = %d, want 1", len(refs))
	}

	// List with template filter
	refs, err = repo.List(ctx, reference.ListOptions{Template: "react-vite"})
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}
	if len(refs) != 1 {
		t.Errorf("List filtered count = %d, want 1", len(refs))
	}

	refs, err = repo.List(ctx, reference.ListOptions{Template: "nonexistent"})
	if err != nil {
		t.Fatalf("List with nonexistent filter failed: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("List nonexistent count = %d, want 0", len(refs))
	}

	// Delete
	if err := repo.Delete(ctx, ref.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = repo.GetByID(ctx, ref.ID)
	if err != reference.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestReferenceRepository_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReferenceRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent-id")
	if err != reference.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	_, err = repo.GetBySlug(ctx, "nonexistent-slug")
	if err != reference.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	err = repo.Delete(ctx, "nonexistent-id")
	if err != reference.ErrNotFound {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}

	_, err = repo.Update(ctx, "nonexistent-id", reference.UpdateInput{})
	if err != reference.ErrNotFound {
		t.Errorf("expected ErrNotFound on update empty, got %v", err)
	}
}

func TestReferenceRepository_UniqueSlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReferenceRepository(db)
	ctx := context.Background()

	input := reference.CreateInput{
		Slug:     "unique-slug",
		Name:     "First",
		Template: "t1",
		Path:     "/tmp/first",
	}
	_, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	// Duplicate slug should fail
	input.Name = "Second"
	input.Path = "/tmp/second"
	_, err = repo.Create(ctx, input)
	if err == nil {
		t.Fatal("expected error for duplicate slug")
	}
}

func TestReferenceRepository_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewReferenceRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := repo.Create(ctx, reference.CreateInput{
			Slug:     "ref-" + string(rune('a'+i)),
			Name:     "Ref " + string(rune('A'+i)),
			Template: "t1",
			Path:     "/tmp/" + string(rune('a'+i)),
		})
		if err != nil {
			t.Fatalf("create ref %d: %v", i, err)
		}
	}

	refs, err := repo.List(ctx, reference.ListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("List limit: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("limit 2 got %d results", len(refs))
	}

	refs, err = repo.List(ctx, reference.ListOptions{Limit: 2, Offset: 3})
	if err != nil {
		t.Fatalf("List offset: %v", err)
	}
	if len(refs) != 2 {
		t.Errorf("offset 3, limit 2 got %d results, want 2", len(refs))
	}
}
