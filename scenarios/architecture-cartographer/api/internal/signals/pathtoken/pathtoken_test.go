package pathtoken_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/pathtoken"
)

func gctx() signals.GraphContext {
	return signals.NewGraphContext("demo", graph.GraphSnapshot{}, domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	})
}

func TestScore_PathMatchesDomain(t *testing.T) {
	sig := pathtoken.New()
	out := sig.Score(context.Background(), gctx(), graph.Chunk{
		ID: "c-1", Path: "internal/conflicts/service.go",
	})
	if len(out.Scores) != 1 {
		t.Fatalf("want 1 score, got %+v", out)
	}
	if out.Scores[0].Domain != "conflicts" {
		t.Fatalf("unexpected domain: %s", out.Scores[0].Domain)
	}
	if out.Scores[0].Value <= 0 || out.Scores[0].Value > 1 {
		t.Fatalf("value out of range: %f", out.Scores[0].Value)
	}
	if len(out.Scores[0].Evidence) == 0 {
		t.Fatal("evidence must not be empty")
	}
}

func TestScore_NoMatchAbstains(t *testing.T) {
	sig := pathtoken.New()
	out := sig.Score(context.Background(), gctx(), graph.Chunk{
		ID: "c-1", Path: "internal/foo/bar.go",
	})
	if len(out.Scores) != 0 {
		t.Fatalf("want no scores, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected an abstention when no domain token matches the path")
	}
	if out.Abstention.Reason == "" || len(out.Abstention.Evidence) == 0 {
		t.Fatalf("abstention must have Reason + ≥1 Evidence: %+v", out.Abstention)
	}
}

func TestScore_Reproducible(t *testing.T) {
	sig := pathtoken.New()
	a := sig.Score(context.Background(), gctx(), graph.Chunk{Path: "internal/conflicts/service.go"})
	b := sig.Score(context.Background(), gctx(), graph.Chunk{Path: "internal/conflicts/service.go"})
	if len(a.Scores) != len(b.Scores) || a.Scores[0].Value != b.Scores[0].Value {
		t.Fatalf("not reproducible: %+v vs %+v", a, b)
	}
}

func TestScore_Bounded(t *testing.T) {
	sig := pathtoken.New()
	out := sig.Score(context.Background(), gctx(), graph.Chunk{Path: "conflicts.go"})
	for _, s := range out.Scores {
		if s.Value < 0 || s.Value > 1 {
			t.Fatalf("out-of-bounds value: %f", s.Value)
		}
	}
}

func TestScore_EmptyPathAbstains(t *testing.T) {
	sig := pathtoken.New()
	out := sig.Score(context.Background(), gctx(), graph.Chunk{})
	if len(out.Scores) != 0 {
		t.Fatalf("want no scores on empty path, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected abstention on empty path")
	}
}
