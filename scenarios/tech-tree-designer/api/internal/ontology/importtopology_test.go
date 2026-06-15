package ontology

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"tech-tree-designer/internal/testutil/db"

	apicoredb "github.com/vrooli/api-core/database"
	ontologyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/ontology"
)

func TestParseTopologySeedCounts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "seed", "macro_topology.json"))
	if err != nil {
		t.Fatalf("ReadFile(seed) error = %v", err)
	}
	topology, err := ParseTopology(data)
	if err != nil {
		t.Fatalf("ParseTopology() error = %v", err)
	}
	var sectors int
	var stages int
	for _, capability := range topology.Capabilities {
		if capability.Kind == ontologyv1.CapabilityKind_CAPABILITY_KIND_SECTOR {
			sectors++
			continue
		}
		stages++
	}
	if sectors != 24 || stages != 209 || len(topology.Edges) != 246 {
		t.Fatalf("counts = sectors:%d stages:%d edges:%d, want 24/209/246", sectors, stages, len(topology.Edges))
	}
	var foundA1 bool
	for _, capability := range topology.Capabilities {
		if capability.ID == "a1" && capability.ParentID == "cluster_business" {
			foundA1 = true
			break
		}
	}
	if !foundA1 {
		t.Fatal("A1 capability under cluster_business not found")
	}
}

func TestServiceImportTopologyIsIdempotent(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	if err := apicoredb.EnsureSchemas(ctx, database, apicoredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas() error = %v", err)
	}
	service := NewService(NewSQLiteRepository(database))
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "seed", "macro_topology.json"))
	if err != nil {
		t.Fatalf("ReadFile(seed) error = %v", err)
	}
	first, err := service.ImportTopology(ctx, data)
	if err != nil {
		t.Fatalf("ImportTopology(first) error = %v", err)
	}
	if first.SectorsImported != 24 || first.CapabilitiesImported != 209 || first.EdgesImported != 244 {
		t.Fatalf("first import = %+v, want imported 24/209/244 unique edges", first)
	}
	second, err := service.ImportTopology(ctx, data)
	if err != nil {
		t.Fatalf("ImportTopology(second) error = %v", err)
	}
	if second.SectorsImported != 0 || second.CapabilitiesImported != 0 || second.EdgesImported != 0 {
		t.Fatalf("second import = %+v, want imported 0/0/0", second)
	}
	if second.SectorsTotal != 24 || second.CapabilitiesTotal != 209 || second.EdgesTotal != 244 {
		t.Fatalf("second totals = %+v, want totals 24/209/244 unique edges", second)
	}
}
