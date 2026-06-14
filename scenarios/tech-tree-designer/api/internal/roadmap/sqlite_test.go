package roadmap

import (
	"context"
	"testing"

	"tech-tree-designer/internal/testutil/db"

	apicoredb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositoryRoundTrip(t *testing.T) {
	// [REQ:TTD-ROADMAP-001] Roadmap overlay metadata persists in domain-owned tables.
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	repo := NewSQLiteRepository(database)

	sector, err := repo.UpsertSector(ctx, Sector{Slug: "engineering", Name: "Engineering", Description: "Build systems"})
	if err != nil {
		t.Fatalf("UpsertSector() error = %v", err)
	}
	if sector.Slug != "engineering" || sector.Name != "Engineering" {
		t.Fatalf("sector = %+v", sector)
	}

	milestone, err := repo.UpsertMilestone(ctx, Milestone{
		ID:                "proto-foundation",
		Name:              "Proto foundation",
		RequiredScenarios: []string{"tech-tree-designer", "proto-health", "proto-health"},
	})
	if err != nil {
		t.Fatalf("UpsertMilestone() error = %v", err)
	}
	if got, want := len(milestone.RequiredScenarios), 2; got != want {
		t.Fatalf("len(required) = %d, want %d", got, want)
	}

	sectors, err := repo.ListSectors(ctx)
	if err != nil {
		t.Fatalf("ListSectors() error = %v", err)
	}
	if len(sectors) != 1 {
		t.Fatalf("len(sectors) = %d, want 1", len(sectors))
	}

	milestones, err := repo.ListMilestones(ctx)
	if err != nil {
		t.Fatalf("ListMilestones() error = %v", err)
	}
	if len(milestones) != 1 || milestones[0].ID != "proto-foundation" {
		t.Fatalf("milestones = %+v", milestones)
	}
}
