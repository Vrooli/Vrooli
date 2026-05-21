package symbolglossary_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/symbolglossary"
)

func gctx(snap graph.GraphSnapshot, m manifest.ManifestDefinition) signals.GraphContext {
	return signals.NewGraphContext("demo", snap, m)
}

func TestScore_GlossaryHitProducesScore(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{ID: "sym:1", Name: "Detector", FileID: "file:a", Exported: true},
			{ID: "sym:2", Name: "Resolver", FileID: "file:a", Exported: true},
		},
	}
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "conflicts", Glossary: []string{"Detector", "Resolver"}},
			{Name: "graph", Glossary: []string{"Snapshot"}},
		},
	}
	out := symbolglossary.New().Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	if len(out) != 1 {
		t.Fatalf("want 1 score (conflicts), got %+v", out)
	}
	if out[0].Domain != "conflicts" {
		t.Fatalf("unexpected domain: %s", out[0].Domain)
	}
	if out[0].Value <= 0 {
		t.Fatalf("expected positive score, got %f", out[0].Value)
	}
	if len(out[0].Evidence) == 0 {
		t.Fatal("evidence required")
	}
}

func TestScore_NoExportedSymbolsReturnsEmpty(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{Name: "internal", FileID: "file:a", Exported: false},
		},
	}
	out := symbolglossary.New().Score(context.Background(), gctx(snap, manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{{Name: "x", Glossary: []string{"internal"}}},
	}), graph.Chunk{FileID: "file:a"})
	if len(out) != 0 {
		t.Fatalf("expected no scores, got %+v", out)
	}
}

func TestScore_Reproducible(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{Name: "Detector", FileID: "file:a", Exported: true},
		},
	}
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{{Name: "conflicts", Glossary: []string{"Detector"}}},
	}
	sig := symbolglossary.New()
	a := sig.Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	b := sig.Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	if len(a) != len(b) || a[0].Value != b[0].Value {
		t.Fatalf("not reproducible: %+v vs %+v", a, b)
	}
}
