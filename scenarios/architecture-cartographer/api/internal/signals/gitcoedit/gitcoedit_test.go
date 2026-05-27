package gitcoedit_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	gitmocks "architecture-cartographer/internal/git/mocks"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/gitcoedit"
)

func TestSignal_DisabledWhenGitUnavailable(t *testing.T) {
	runner := &gitmocks.FakeRunner{Available: false}
	sig := gitcoedit.New(runner)
	ok, reason := sig.IsAvailable(context.Background())
	if ok {
		t.Fatal("want unavailable")
	}
	if reason == "" {
		t.Fatal("want explanation")
	}
}

func TestScore_SilentWhenDisabled(t *testing.T) {
	sig := gitcoedit.New(&gitmocks.FakeRunner{Available: false})
	out := sig.Score(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, domains.DerivedDomainMap{}), graph.Chunk{Path: "a.go"})
	if len(out) != 0 {
		t.Fatalf("disabled signal must score nothing, got %+v", out)
	}
}

func TestScore_CoEditFrequencyMapsToDomain(t *testing.T) {
	log := `abc1234abc1234abc1234abc1234abc1234ab12
internal/conflicts/service.go
internal/conflicts/types.go
target/file.go

def5678def5678def5678def5678def5678ef34
internal/conflicts/registry.go
target/file.go

aaa1111aaa1111aaa1111aaa1111aaa1111aa55
internal/conflicts/types.go
target/file.go
`
	runner := &gitmocks.FakeRunner{Available: true, LogOutput: log}
	sig := gitcoedit.New(runner)
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	out := sig.Score(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, m), graph.Chunk{Path: "target/file.go"})
	if len(out) != 1 {
		t.Fatalf("want 1 score, got %+v", out)
	}
	if out[0].Domain != "conflicts" || out[0].Value <= 0 {
		t.Fatalf("unexpected: %+v", out[0])
	}
}

func TestScore_BelowMinCommitsReturnsEmpty(t *testing.T) {
	log := `abc1234abc1234abc1234abc1234abc1234ab12
target/file.go
internal/conflicts/foo.go
`
	runner := &gitmocks.FakeRunner{Available: true, LogOutput: log}
	sig := gitcoedit.New(runner)
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	out := sig.Score(context.Background(), signals.NewGraphContext("demo", graph.GraphSnapshot{}, m), graph.Chunk{Path: "target/file.go"})
	if len(out) != 0 {
		t.Fatalf("expected empty (below MinCoEditCommits), got %+v", out)
	}
}
