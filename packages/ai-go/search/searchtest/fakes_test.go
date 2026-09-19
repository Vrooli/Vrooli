package searchtest

import (
	"context"
	"testing"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestPagedSourceReturnsStableBoundedCopies(t *testing.T) {
	source := PagedSource{Documents: []aisearch.SourceDoc{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	first, err := source.LoadPage(context.Background(), aisearch.PageRequest{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if first.Done || first.NextCursor != "2" || len(first.Documents) != 2 {
		t.Fatalf("unexpected first page: %+v", first)
	}
	first.Documents[0].ID = "mutated"
	if source.Documents[0].ID != "a" {
		t.Fatal("page mutation must not mutate the fake corpus")
	}
	second, err := source.LoadPage(context.Background(), aisearch.PageRequest{Cursor: first.NextCursor, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Done || len(second.Documents) != 1 || second.Documents[0].ID != "c" {
		t.Fatalf("unexpected second page: %+v", second)
	}
}

func TestEmbedderAndVectorStoreReturnCopies(t *testing.T) {
	embedder := &Embedder{Vector: []float64{1, 2}, AvailableValue: true}
	vector, err := embedder.Embed(context.Background(), "query")
	if err != nil {
		t.Fatal(err)
	}
	vector[0] = 99
	if embedder.Vector[0] != 1 {
		t.Fatal("embed result must not alias configured vector")
	}

	store := NewVectorStore()
	point := aisearch.Point{ID: "p", Dense: []float64{1}, Payload: map[string]any{"source_id": "s"}}
	if err := store.Upsert(context.Background(), point); err != nil {
		t.Fatal(err)
	}
	point.Dense[0] = 99
	point.Payload["source_id"] = "mutated"
	if store.Points["p"].Dense[0] != 1 || store.Points["p"].Payload["source_id"] != "s" {
		t.Fatal("stored point must not alias caller-owned slices or maps")
	}
}
