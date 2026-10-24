package aisearch

import (
	"context"
	"testing"
)

type nilSource struct{}

func (nilSource) LoadAll(context.Context) ([]SourceDoc, error) { return nil, nil }

func TestNewDenseBinding(t *testing.T) {
	store := NewVectorStoreWithClient("http://q", "", "c", &capturingDoer{})
	src := nilSource{}
	b := NewDenseBinding("command", "cli-health:", store, src)

	if b.Kind != "command" || b.IDPrefix != "cli-health:" {
		t.Fatalf("unexpected identity fields: %+v", b)
	}
	if b.Store != store {
		t.Fatal("store not wired")
	}
	if b.Source != src {
		t.Fatal("source not wired")
	}
	if b.Sparse != nil {
		t.Fatal("dense binding must have no sparse encoder")
	}
	if b.Composer != nil {
		t.Fatal("dense binding uses the identity composer (nil)")
	}
	// The chunker must be the identity chunker: one doc -> one chunk.
	chunks, err := b.Chunker.Chunk(SourceDoc{ID: "x", Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Body != "hello" {
		t.Fatalf("expected identity chunker (1 chunk verbatim), got %+v", chunks)
	}
}
