package mocks

import (
	"context"
	"reflect"
	"testing"
	"time"

	"code-facts/internal/retrieval"
	"code-facts/internal/targets"
)

func TestPlatformFakesAreDeterministicAndCopyOutputs(t *testing.T) {
	clock := &Clock{Time: time.Unix(42, 0)}
	if got := clock.Now(); !got.Equal(time.Unix(42, 0)) {
		t.Fatalf("clock = %v", got)
	}
	fs := &FileSystem{
		Files: map[string][]byte{"b": []byte("payload")},
		Info:  map[string]targets.FileInfo{"b": {Path: "b"}, "a": {Path: "a"}},
	}
	var visited []string
	if err := fs.Walk(context.Background(), nil, func(info targets.FileInfo) error {
		visited = append(visited, info.Path)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(visited, []string{"a", "b"}) {
		t.Fatalf("visit order = %v", visited)
	}
	embedder := &Embedder{Vectors: [][]float32{{1, 2}}}
	vectors, err := embedder.Embed(context.Background(), []string{"x"})
	if err != nil {
		t.Fatal(err)
	}
	vectors[0][0] = 9
	if embedder.Vectors[0][0] != 1 {
		t.Fatal("fake embedder leaked mutable output")
	}
	store := &VectorStore{Results: []retrieval.Candidate{{ID: "one"}}}
	results, err := store.Query(context.Background(), nil, retrieval.Query{})
	if err != nil || len(results) != 1 || results[0].ID != "one" {
		t.Fatalf("vector results = %#v, err=%v", results, err)
	}
}
