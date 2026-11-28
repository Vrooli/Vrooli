package graph_test

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/graph"
)

func newRepo(t *testing.T) *graph.SQLite {
	t.Helper()
	return graph.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(graph.Schema)))
}

func TestEdgeRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if err := repo.UpsertEdges(ctx, []graph.Edge{{
		SourceID:         "vec-a",
		TargetID:         "vec-b",
		RelationshipType: "semantic_similarity",
		Weight:           0.83,
	}}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	edges, err := repo.ListEdges(ctx, 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("got %d edges, want 1", len(edges))
	}
	got := edges[0]
	if got.ID == "" {
		t.Error("id was not generated")
	}
	if got.SourceID != "vec-a" || got.TargetID != "vec-b" {
		t.Errorf("endpoints = %q/%q, want vec-a/vec-b", got.SourceID, got.TargetID)
	}
	if got.RelationshipType != "semantic_similarity" {
		t.Errorf("relationship_type = %q", got.RelationshipType)
	}
	// weight was DECIMAL(3,2) on Postgres; REAL must not round it.
	if got.Weight != 0.83 {
		t.Errorf("weight = %v, want 0.83", got.Weight)
	}
	if got.DiscoveredAt.IsZero() {
		t.Error("discovered_at was not defaulted")
	}
}

// TestUpsertOnCompositeKeyUpdatesInPlace proves the three-column unique
// constraint carried over: re-discovering an edge updates its weight rather
// than inserting a duplicate.
func TestUpsertOnCompositeKeyUpdatesInPlace(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	for _, w := range []float64{0.2, 0.9} {
		if err := repo.UpsertEdges(ctx, []graph.Edge{{
			SourceID: "a", TargetID: "b", RelationshipType: "semantic_similarity", Weight: w,
		}}); err != nil {
			t.Fatalf("upsert %v: %v", w, err)
		}
	}

	n, err := repo.CountEdges(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
	edges, _ := repo.ListEdges(ctx, 10)
	if edges[0].Weight != 0.9 {
		t.Errorf("weight = %v, want the updated 0.9", edges[0].Weight)
	}
}

// TestDegenerateEdgesAreSkippedNotFailed keeps batch discovery best-effort: one
// bad pair must not discard the whole batch.
func TestDegenerateEdgesAreSkippedNotFailed(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	err := repo.UpsertEdges(ctx, []graph.Edge{
		{SourceID: "a", TargetID: "a", RelationshipType: "self"}, // self-edge
		{SourceID: "", TargetID: "b"},                            // missing source
		{SourceID: "a", TargetID: "b", Weight: 0.5},              // valid, default type
	})
	if err != nil {
		t.Fatalf("upsert must skip degenerate edges, not fail: %v", err)
	}

	edges, err := repo.ListEdges(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) != 1 {
		t.Fatalf("got %d edges, want only the valid one", len(edges))
	}
	if edges[0].RelationshipType != "semantic_similarity" {
		t.Errorf("relationship_type = %q, want the default", edges[0].RelationshipType)
	}
}
