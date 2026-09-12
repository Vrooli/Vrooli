package routing

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

type stubSharedReranker struct {
	name      string
	available bool
	scores    []aisearch.RerankScore
	err       error
	onRerank  func(*stubSharedReranker)
	called    bool
	gotQuery  string
	got       []aisearch.RerankCandidate
}

func (s *stubSharedReranker) Name() string { return s.name }

func (s *stubSharedReranker) Available(context.Context) bool { return s.available }

func (s *stubSharedReranker) Rerank(_ context.Context, query string, candidates []aisearch.RerankCandidate) ([]aisearch.RerankScore, error) {
	s.called = true
	s.gotQuery = query
	s.got = append([]aisearch.RerankCandidate(nil), candidates...)
	if s.onRerank != nil {
		s.onRerank(s)
	}
	return s.scores, s.err
}

func TestSharedRerankerMapsHitsToCandidatesAndAppliesScores(t *testing.T) {
	shared := &stubSharedReranker{
		name:      "cross",
		available: true,
		onRerank: func(s *stubSharedReranker) {
			s.scores = []aisearch.RerankScore{
				{ID: s.got[1].ID, Score: 0.91},
				{ID: s.got[0].ID, Score: 0.20},
			}
		},
	}
	rr := NewSharedReranker(shared)

	ranked, err := rr.Rerank(context.Background(), "restart api", []*routingv1.SearchHit{
		{Id: "doc-a", Type: "doc", Title: "Restart service", Snippet: "Use vrooli scenario restart", Path: "docs/restart.md", Score: 0.99},
		{Id: "doc-b", Type: "record", Title: "API restart fixed", Snippet: "Prior work record", Path: "records/api", Score: 0.10},
	})

	require.NoError(t, err)
	require.True(t, shared.called)
	require.Equal(t, "restart api", shared.gotQuery)
	require.Equal(t, []string{"doc-a", "doc-b"}, []string{shared.got[0].ID[strings.LastIndex(shared.got[0].ID, ":")+1:], shared.got[1].ID[strings.LastIndex(shared.got[1].ID, ":")+1:]})
	require.NotEqual(t, shared.got[0].ID, shared.got[1].ID, "candidate keys must remain unique")
	require.Equal(t, []string{"doc-b", "doc-a"}, ids(ranked))
	require.InDelta(t, 0.91, ranked[0].GetRerankScore(), 1e-9)
}

func TestSharedRerankerScoresDuplicateIDsPerCandidate(t *testing.T) {
	shared := &stubSharedReranker{
		name:      "cross",
		available: true,
		onRerank: func(s *stubSharedReranker) {
			s.scores = []aisearch.RerankScore{
				{ID: s.got[0].ID, Score: 0.20},
				{ID: s.got[1].ID, Score: 0.80},
			}
		},
	}
	rr := NewSharedReranker(shared)
	ranked, err := rr.Rerank(context.Background(), "q", []*routingv1.SearchHit{
		{Id: "same-doc", Path: "docs/flows.md", Snippet: "first passage", Score: 0.9},
		{Id: "same-doc", Path: "docs/flows.md", Snippet: "second passage", Score: 0.8},
	})

	require.NoError(t, err)
	require.Len(t, ranked, 2)
	require.Equal(t, "second passage", ranked[0].GetSnippet())
	require.InDelta(t, 0.80, ranked[0].GetRerankScore(), 1e-9)
	require.InDelta(t, 0.20, ranked[1].GetRerankScore(), 1e-9)
}

func TestSharedRerankerChainPrefersCrossEncoder(t *testing.T) {
	cross := &stubSharedReranker{
		name:      "cross-encoder:test",
		available: true,
		onRerank:  scoreCandidate("a", 0.9),
	}
	llm := &stubSharedReranker{
		name:      "llm:test",
		available: true,
		onRerank:  scoreCandidate("b", 0.9),
	}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross, llm))

	ranked, err := rr.Rerank(context.Background(), "q", chainTestHits())

	require.NoError(t, err)
	require.True(t, cross.called)
	require.False(t, llm.called)
	require.Equal(t, []string{"a", "b"}, ids(ranked))
	require.Equal(t, "cross-encoder:test", rr.ActiveName(context.Background()))
}

func TestSharedRerankerChainFallsBackToLLM(t *testing.T) {
	cross := &stubSharedReranker{name: "cross-encoder:test", available: false}
	llm := &stubSharedReranker{
		name:      "llm:test",
		available: true,
		onRerank:  scoreCandidate("b", 0.9),
	}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross, llm))

	ranked, err := rr.Rerank(context.Background(), "q", chainTestHits())

	require.NoError(t, err)
	require.False(t, cross.called)
	require.True(t, llm.called)
	require.Equal(t, []string{"b", "a"}, ids(ranked))
	require.Equal(t, "llm:test", rr.ActiveName(context.Background()))
}

func TestSharedRerankerRefreshesCachedPrimaryFailure(t *testing.T) {
	cross := &stubSharedReranker{
		name:      "cross-encoder:test",
		available: true,
		err:       errors.New("tei down"),
		onRerank:  func(s *stubSharedReranker) { s.available = false },
	}
	llm := &stubSharedReranker{
		name:      "llm:test",
		available: true,
		onRerank:  scoreCandidate("b", 0.9),
	}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross, llm))

	ranked, err := rr.Rerank(context.Background(), "q", chainTestHits())

	require.NoError(t, err)
	require.True(t, cross.called)
	require.True(t, llm.called)
	require.Equal(t, []string{"b", "a"}, ids(ranked))
	require.Equal(t, "llm:test", rr.ActiveName(context.Background()))
}

func TestSharedRerankerSortsByRerankThenProviderScore(t *testing.T) {
	shared := &stubSharedReranker{
		name:      "cross",
		available: true,
		onRerank: func(s *stubSharedReranker) {
			s.scores = []aisearch.RerankScore{
				{ID: s.got[0].ID, Score: 0.4},
				{ID: s.got[1].ID, Score: 0.4},
				{ID: s.got[2].ID, Score: 0.9},
			}
		},
	}
	rr := NewSharedReranker(shared)

	ranked, err := rr.Rerank(context.Background(), "q", []*routingv1.SearchHit{
		{Id: "low", Title: "low", Score: 0.1},
		{Id: "high", Title: "high", Score: 0.8},
		{Id: "top", Title: "top", Score: 0.2},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"top", "high", "low"}, ids(ranked))
}

func scoreCandidate(id string, score float64) func(*stubSharedReranker) {
	return func(s *stubSharedReranker) {
		for _, candidate := range s.got {
			if strings.HasSuffix(candidate.ID, ":"+id) {
				s.scores = []aisearch.RerankScore{{ID: candidate.ID, Score: score}}
				return
			}
		}
		panic("candidate not found: " + id)
	}
}

func TestSharedRerankerEmptyChainScoresBecomeUnavailableError(t *testing.T) {
	shared := &stubSharedReranker{name: "none", available: false}
	rr := NewSharedReranker(shared)

	_, err := rr.Rerank(context.Background(), "q", singleTestHit())
	require.ErrorContains(t, err, "reranker chain unavailable")
}

func TestSharedRerankerPropagatesSharedError(t *testing.T) {
	shared := &stubSharedReranker{name: "llm", available: true, err: errors.New("llm timeout")}
	rr := NewSharedReranker(shared)

	_, err := rr.Rerank(context.Background(), "q", singleTestHit())
	require.ErrorContains(t, err, "llm timeout")
}

func TestSharedRerankerAvailableDelegatesToChain(t *testing.T) {
	rr := NewSharedReranker(&stubSharedReranker{name: "cross", available: true})
	require.True(t, rr.Available(context.Background()))

	rr = NewSharedReranker(&stubSharedReranker{name: "none", available: false})
	require.False(t, rr.Available(context.Background()))
}

func chainTestHits() []*routingv1.SearchHit {
	return []*routingv1.SearchHit{
		{Id: "a", Title: "a"},
		{Id: "b", Title: "b"},
	}
}

func singleTestHit() []*routingv1.SearchHit {
	return []*routingv1.SearchHit{{Id: "a", Title: "a"}}
}
