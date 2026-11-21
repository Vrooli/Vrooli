package tags

import (
	"context"
	"testing"

	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

func TestRepositoryCreatesAndListsTagsInSQLite(t *testing.T) {
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("apply tags schema: %v", err)
	}
	repo := NewRepository(db.Primary())

	color := "#1188ff"
	description := "Storage related skills"
	if err := repo.Create(&Tag{Name: "storage", Color: &color, Description: &description}); err != nil {
		t.Fatalf("create storage tag: %v", err)
	}
	if err := repo.Create(&Tag{Name: "debugging"}); err != nil {
		t.Fatalf("create debugging tag: %v", err)
	}

	got, err := repo.GetAll()
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tags, got %d: %+v", len(got), got)
	}
	if got[0].Name != "debugging" || got[1].Name != "storage" {
		t.Fatalf("expected name ordering, got %+v", got)
	}
	if got[1].ID == "" || got[1].Color == nil || *got[1].Color != color || got[1].Description == nil || *got[1].Description != description {
		t.Fatalf("expected optional fields to round-trip, got %+v", got[1])
	}
}
