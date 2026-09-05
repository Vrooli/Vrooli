package ontology

import (
	"context"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	apicoredb "github.com/vrooli/api-core/database"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

func TestSQLiteRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	repo := NewSQLiteRepository(database)

	sector, err := repo.UpsertCapability(ctx, Capability{
		ID:   "cluster-engineering",
		Slug: "cluster-engineering",
		Name: "Engineering",
		Kind: ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR,
	})
	if err != nil {
		t.Fatalf("UpsertCapability(sector) error = %v", err)
	}
	component, err := repo.UpsertCapability(ctx, Capability{
		ID:       "proto-foundation",
		Slug:     "proto-foundation",
		Name:     "Proto foundation",
		Kind:     ontologyv1.CapabilityKind_CAPABILITY_KIND_COMPONENT,
		ParentID: sector.ID,
	})
	if err != nil {
		t.Fatalf("UpsertCapability(component) error = %v", err)
	}
	if component.ParentID != sector.ID {
		t.Fatalf("component.ParentID = %q, want %q", component.ParentID, sector.ID)
	}

	descendants, err := repo.ListCapabilities(ctx, CapabilityFilter{ParentID: sector.ID, IncludeDescendants: true})
	if err != nil {
		t.Fatalf("ListCapabilities(descendants) error = %v", err)
	}
	if got, want := len(descendants), 1; got != want {
		t.Fatalf("len(descendants) = %d, want %d: %+v", got, want, descendants)
	}

	edge, err := repo.UpsertCapabilityEdge(ctx, CapabilityEdge{
		FromID: sector.ID,
		ToID:   component.ID,
		Type:   ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES,
	})
	if err != nil {
		t.Fatalf("UpsertCapabilityEdge() error = %v", err)
	}
	if edge.Type != ontologyv1.CapabilityEdgeType_CAPABILITY_EDGE_TYPE_DECOMPOSES {
		t.Fatalf("edge.Type = %v", edge.Type)
	}

	fulfillment, err := repo.LinkFulfillment(ctx, Fulfillment{
		CapabilityID: component.ID,
		ScenarioSlug: "proto-health",
		Note:         "discovers proto imports",
	})
	if err != nil {
		t.Fatalf("LinkFulfillment() error = %v", err)
	}
	if fulfillment.ScenarioSlug != "proto-health" {
		t.Fatalf("fulfillment = %+v", fulfillment)
	}
}
