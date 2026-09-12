package symbolglossary_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/symbolglossary"
)

func gctx(snap graph.GraphSnapshot, m domains.DerivedDomainMap) signals.GraphContext {
	return signals.NewGraphContext("demo", snap, m)
}

func TestScore_GlossaryHitProducesScore(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{ID: "sym:1", Name: "Detector", FileID: "file:a", Exported: true},
			{ID: "sym:2", Name: "Resolver", FileID: "file:a", Exported: true},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Glossary: []string{"Detector", "Resolver"}},
			{Name: "graph", Glossary: []string{"Snapshot"}},
		},
	}
	out := symbolglossary.New().Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	if len(out.Scores) != 1 {
		t.Fatalf("want 1 score (conflicts), got %+v", out)
	}
	if out.Scores[0].Domain != "conflicts" {
		t.Fatalf("unexpected domain: %s", out.Scores[0].Domain)
	}
	if out.Scores[0].Value <= 0 {
		t.Fatalf("expected positive score, got %f", out.Scores[0].Value)
	}
	if len(out.Scores[0].Evidence) == 0 {
		t.Fatal("evidence required")
	}
}

func TestScore_NoExportedSymbolsAbstains(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{Name: "internal", FileID: "file:a", Exported: false},
		},
	}
	out := symbolglossary.New().Score(context.Background(), gctx(snap, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{Name: "x", Glossary: []string{"internal"}}},
	}), graph.Chunk{FileID: "file:a"})
	if len(out.Scores) != 0 {
		t.Fatalf("expected no scores, got %+v", out.Scores)
	}
	if out.Abstention == nil || out.Abstention.Reason == "" || len(out.Abstention.Evidence) == 0 {
		t.Fatalf("expected abstention with Reason + Evidence, got %+v", out.Abstention)
	}
}

func TestScore_Reproducible(t *testing.T) {
	snap := graph.GraphSnapshot{
		Symbols: []graph.SymbolNode{
			{Name: "Detector", FileID: "file:a", Exported: true},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{Name: "conflicts", Glossary: []string{"Detector"}}},
	}
	sig := symbolglossary.New()
	a := sig.Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	b := sig.Score(context.Background(), gctx(snap, m), graph.Chunk{FileID: "file:a"})
	if len(a.Scores) != len(b.Scores) || a.Scores[0].Value != b.Scores[0].Value {
		t.Fatalf("not reproducible: %+v vs %+v", a, b)
	}
}
